package kafka

import (
	"context"
	"errors"
	"log"
	"matching_service/internal/biz"

	"github.com/IBM/sarama"
)

type consumerGroupHandler struct {
	bizHandler biz.EventHandler
}

func (cgh *consumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

func (cgh *consumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (cgh *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case msg, ok := <-claim.Messages():
			if !ok {
				log.Println("Channel is closed (Rebalancing consumer or Server is down)")
				return nil
			}

			err := cgh.bizHandler(session.Context(), msg.Topic, msg.Value)
			if err != nil {
				if errors.Is(err, biz.ErrNonRetryable) {
					log.Printf("Business error: %v", err)
					session.MarkMessage(msg, "process message failed")
					continue
				}
				log.Printf("System error, close Session for Kafka auto retry: %v", err)
				return err
			}

			session.MarkMessage(msg, "")

		case <-session.Context().Done():
			log.Println("Kafka session context cancelled, exiting ConsumeClaim")
			return nil
		}
	}
}

type kafkaConsumer struct {
	consumerGroup sarama.ConsumerGroup
}

var _ biz.EventConsumer = (*kafkaConsumer)(nil)

func NewKafkaConsumer(brokers []string, groupId string) (biz.EventConsumer, error) {
	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true

	config.Consumer.Offsets.Initial = sarama.OffsetOldest

	consumerGroup, err := sarama.NewConsumerGroup(brokers, groupId, config)
	if err != nil {
		return nil, err
	}

	return &kafkaConsumer{consumerGroup: consumerGroup}, nil
}

func (c *kafkaConsumer) Consume(ctx context.Context, topic string, handler biz.EventHandler) error {
	saramaHandler := &consumerGroupHandler{
		bizHandler: handler,
	}

	go func() {
		for {
			err := c.consumerGroup.Consume(ctx, []string{topic}, saramaHandler)
			if err != nil {
				log.Printf("Kafka consumer error: %v", err)
			}
			if ctx.Err() != nil {
				return
			}
		}
	}()

	return nil
}
