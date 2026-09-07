package biz

import (
	"context"
)

type Header struct {
	Key   []byte
	Value []byte
}

type EventMessage struct {
	Header  *Header
	Topic   string
	Key     string
	Payload any
}

type EventPublisher interface {
	Publish(ctx context.Context, msg *EventMessage) error
}

// subject cần thiết vì có luồng mã hoá tham số vào đó: matching.offers.{bidID}.
type EventHandler func(ctx context.Context, subject string, payload []byte) error

type EventConsumer interface {
	Consume(ctx context.Context, topic string, handler EventHandler) error
}
