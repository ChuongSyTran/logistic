package nats_jetstream

import (
	"errors"
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

// JetStream chỉ nhận publish vào subject thuộc một stream; thiếu stream thì mọi
// Publish trả "no response from stream".
const (
	MatchingStream  = "MATCHING"
	MatchingSubject = "matching.>"

	// Sự kiện thời gian thực, giữ hạn để stream tự co.
	matchingMaxAge   = 24 * time.Hour
	matchingMaxBytes = 512 << 20
)

// EnsureStream gọi lúc khởi động, trước khi phát sự kiện đầu tiên.
func EnsureStream(js nats.JetStreamContext) error {
	cfg := &nats.StreamConfig{
		Name:      MatchingStream,
		Subjects:  []string{MatchingSubject},
		Storage:   nats.FileStorage,
		Retention: nats.LimitsPolicy,
		Discard:   nats.DiscardOld,
		MaxAge:    matchingMaxAge,
		MaxBytes:  matchingMaxBytes,
	}

	_, err := js.StreamInfo(MatchingStream)
	switch {
	case err == nil:
		if _, err := js.UpdateStream(cfg); err != nil {
			return err
		}
		log.Printf("[matching_service] stream %s đã có, đã đồng bộ cấu hình", MatchingStream)
		return nil

	case errors.Is(err, nats.ErrStreamNotFound):
		if _, err := js.AddStream(cfg); err != nil {
			return err
		}
		log.Printf("[matching_service] đã tạo stream %s cho subject %s", MatchingStream, MatchingSubject)
		return nil

	default:
		return err
	}
}
