package nats_jetstream

import (
	"context"
	"errors"
	"log"
	"matching_service/internal/biz"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
)

type natsSubcriber struct {
	natJetStreamContext nats.JetStreamContext
}

func InitSubcriber(natJetStreamContext nats.JetStreamContext) biz.EventConsumer {
	return &natsSubcriber{
		natJetStreamContext: natJetStreamContext,
	}
}

var _ biz.EventConsumer = (*natsSubcriber)(nil)

func (n *natsSubcriber) Consume(ctx context.Context, topic string, handler biz.EventHandler) error {
	name := durableName(topic)

	_, err := n.natJetStreamContext.QueueSubscribe(topic, name, func(msg *nats.Msg) {
		err := handler(ctx, msg.Subject, msg.Data)
		if err != nil {
			if errors.Is(err, biz.ErrNonRetryable) {
				log.Printf("Discarding poison pill message: %v", err)
				msg.Ack()
				return
			}

			if errors.Is(err, biz.ErrRetryWithDelay) {
				log.Printf("Service busy, retrying in 5s: %v", err)
				msg.NakWithDelay(5 * time.Second)
				return
			}

			log.Printf("Transient error, retrying immediately: %v", err)
			msg.Nak()
			return

		}
		msg.Ack()

		// DeliverNew chỉ áp dụng lúc TẠO consumer: lần đầu không tua lại bản tin cũ
		// còn trong stream, khởi động lại vẫn nhận phần phát ra lúc service tắt.
	}, nats.Durable(name), nats.ManualAck(), nats.DeliverNew())

	return err
}

// NATS không cho dấu chấm hay ký tự đại diện trong tên durable.
func durableName(topic string) string {
	replaced := strings.Map(func(r rune) rune {
		switch r {
		case '.', '*', '>':
			return '-'
		}
		return r
	}, topic)

	for strings.Contains(replaced, "--") {
		replaced = strings.ReplaceAll(replaced, "--", "-")
	}
	return strings.Trim(replaced, "-")
}
