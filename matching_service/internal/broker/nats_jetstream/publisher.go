package nats_jetstream

import (
	"context"
	"errors"
	"fmt"
	"log"
	"matching_service/internal/biz"
	"matching_service/internal/broker"
	"matching_service/internal/entity"
	"matching_service/internal/mapper"

	pb "github.com/logistic/api/logistic/matching_service/v1"

	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

var _ biz.EventPublisher = (*natsPublisher)(nil)

type natsPublisher struct {
	natJetStreamContext nats.JetStreamContext
	mapper              mapper.MatchingMapper
}

func InitPublisher(natJetStreamContext nats.JetStreamContext, appMapper mapper.MatchingMapper) biz.EventPublisher {
	return &natsPublisher{
		natJetStreamContext: natJetStreamContext,
		mapper:              appMapper,
	}
}

func (n *natsPublisher) payloadToBytes(payload any) ([]byte, error) {
	switch v := payload.(type) {
	case []byte:
		return v, nil
	case entity.Bid:
		pbBid := n.mapper.EntityBidToPbBid(v)
		return proto.Marshal(pbBid)
	case entity.Ask:
		pbAsk := n.mapper.EntityAskToPbAsk(v)
		return proto.Marshal(pbAsk)
	case []entity.Bid:
		if v == nil {
			return nil, broker.ErrNilMessage
		}
		payload := &pb.Bids{Bids: n.mapper.EntityBidListToPbBidList(v)}
		return proto.Marshal(payload)
	case []entity.Ask:
		if v == nil {
			return nil, broker.ErrNilMessage
		}
		payload := &pb.Asks{Asks: n.mapper.EntityAskListToPbAskList(v)}
		return proto.Marshal(payload)
	default:
		return nil, fmt.Errorf("unsupported payload type for protobuf serialization")
	}
}

func (n *natsPublisher) Publish(ctx context.Context, msg *biz.EventMessage) error {
	if msg == nil {
		return broker.ErrNilMessage
	}

	payloadBytes, err := n.payloadToBytes(msg.Payload)
	if err != nil {
		return err
	}

	ack, err := n.natJetStreamContext.Publish(msg.Topic, payloadBytes)
	if err != nil {
		return parseNatsError(err)
	}

	log.Printf("Message published successfully! Stream: %s, Sequence (ID): %d, Duplicate: %v",
		ack.Stream, ack.Sequence, ack.Duplicate)

	return nil
}

func parseNatsError(err error) error {
	if errors.Is(err, nats.ErrTimeout) ||
		errors.Is(err, nats.ErrConnectionClosed) ||
		errors.Is(err, nats.ErrNoServers) ||
		errors.Is(err, nats.ErrStaleConnection) ||
		errors.Is(err, nats.ErrDisconnected) {
		log.Printf("NATS network/timeout error, retry requested: %v", err)
		return fmt.Errorf("%w: %v", biz.ErrRetryWithDelay, err)
	}

	if errors.Is(err, nats.ErrMaxPayload) ||
		errors.Is(err, nats.ErrNoResponders) ||
		errors.Is(err, nats.ErrBadSubject) ||
		errors.Is(err, nats.ErrAuthorization) {
		log.Printf("NATS configuration/payload error, do not retry: %v", err)
		return fmt.Errorf("%w: %v", biz.ErrNonRetryable, err)
	}

	if apiErr, ok := errors.AsType[*nats.APIError](err); ok {
		log.Printf("NATS JetStream API Error, do not retry: %v", apiErr)
		return fmt.Errorf("%w: %v", biz.ErrNonRetryable, apiErr)
	}

	return fmt.Errorf("%w: unknown NATS error: %v", biz.ErrRetryWithDelay, err)
}
