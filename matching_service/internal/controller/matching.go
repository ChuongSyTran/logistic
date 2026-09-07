package controller

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	pb "github.com/logistic/api/logistic/matching_service/v1"
	"google.golang.org/protobuf/types/known/timestamppb"

	"matching_service/internal/biz"
	cerr "matching_service/internal/common/errors"
	"matching_service/internal/mapper"
	"matching_service/internal/mapper/generated"
)

type matchingController struct {
	pb.UnimplementedMatchingEngineServiceServer
	matchingEngine biz.MatchingEngine
	mapper         mapper.MatchingMapper
}

func NewMatchingController(engine biz.MatchingEngine) *matchingController {
	return &matchingController{
		matchingEngine: engine,
		mapper:         &generated.MatchingMapperImpl{},
	}
}

func (c *matchingController) SubmitBid(ctx context.Context, req *pb.SubmitBidRequest) (*pb.SubmitBidResponse, error) {
	if req.Payload == nil || req.Payload.Origin == nil || req.Payload.Destination == nil {
		return nil, fmt.Errorf("payload, origin, and destination cannot be nil")
	}

	bid, err := c.mapper.PbBidToEntity(req.Payload)
	if err != nil {
		return nil, fmt.Errorf("invalid request data: %w", err)
	}

	matchedBid, err := c.matchingEngine.SubmitBid(ctx, &bid)
	if err != nil {
		return nil, err
	}

	return &pb.SubmitBidResponse{
		BidId:  matchedBid.ID[:],
		Status: matchedBid.StatusString(),
	}, nil
}

func (c *matchingController) SubmitAsk(ctx context.Context, req *pb.SubmitAskRequest) (*pb.SubmitAskResponse, error) {
	if req.Payload == nil || req.Payload.CurrentLocation == nil || req.Payload.Destination == nil {
		return nil, fmt.Errorf("payload, current_location, and destination cannot be nil")
	}

	ask, err := c.mapper.PbAskToEntity(req.Payload)
	if err != nil {
		return nil, fmt.Errorf("invalid request data: %w", err)
	}

	matchedAsk, err := c.matchingEngine.SubmitAsk(ctx, &ask)
	if err != nil {
		return nil, err
	}

	return &pb.SubmitAskResponse{
		AskId:  matchedAsk.ID[:],
		Status: matchedAsk.StatusString(),
	}, nil
}

func (c *matchingController) AcceptMatch(ctx context.Context, req *pb.AcceptMatchRequest) (*pb.AcceptMatchResponse, error) {
	bidID, err := uuid.FromBytes(req.BidId)
	if err != nil {
		return nil, cerr.ErrInvalidID.WithMessage("bid_id không hợp lệ").WithCause(err)
	}
	askID, err := uuid.FromBytes(req.AskId)
	if err != nil {
		return nil, cerr.ErrInvalidID.WithMessage("ask_id không hợp lệ").WithCause(err)
	}

	contract, err := c.matchingEngine.AcceptOffer(ctx, bidID, askID, req.ConsensusPrice, req.ShipperSignature)
	if err != nil {
		return nil, err
	}

	return &pb.AcceptMatchResponse{
		ContractId:       contract.ID[:],
		BidId:            contract.BidID[:],
		AskId:            contract.AskID[:],
		ConsensusPrice:   contract.ConsensusPrice,
		ConsensusDeposit: contract.ConsensusDeposit,
		ShipperSignature: contract.ShipperSignature,
		AgreedAt:         timestamppb.New(contract.AgreedAt),
		CreatedAt:        timestamppb.New(contract.AgreedAt),
	}, nil
}

func (c *matchingController) SubmitOffer(ctx context.Context, req *pb.SubmitOfferRequest) (*pb.SubmitOfferResponse, error) {
	bidID, err := uuid.FromBytes(req.BidId)
	if err != nil {
		return nil, cerr.ErrInvalidID.WithMessage("bid_id không hợp lệ").WithCause(err)
	}
	askID, err := uuid.FromBytes(req.AskId)
	if err != nil {
		return nil, cerr.ErrInvalidID.WithMessage("ask_id không hợp lệ").WithCause(err)
	}
	if req.DesiredPrice <= 0 {
		return nil, cerr.ErrInvalidID.WithMessage("desired_price phải lớn hơn 0")
	}

	if err := c.matchingEngine.SubmitOffer(ctx, bidID, askID, req.DesiredPrice); err != nil {
		return nil, err
	}

	return &pb.SubmitOfferResponse{
		Status:  "SUBMITTED",
		Message: "Đã gửi báo giá tới chủ hàng",
	}, nil
}

func (c *matchingController) RejectOffer(ctx context.Context, req *pb.RejectOfferRequest) (*pb.RejectOfferResponse, error) {
	bidID, err := uuid.FromBytes(req.BidId)
	if err != nil {
		return nil, cerr.ErrInvalidID.WithMessage("bid_id không hợp lệ").WithCause(err)
	}
	askID, err := uuid.FromBytes(req.AskId)
	if err != nil {
		return nil, cerr.ErrInvalidID.WithMessage("ask_id không hợp lệ").WithCause(err)
	}

	if err := c.matchingEngine.RejectOffer(ctx, bidID, askID); err != nil {
		return nil, err
	}

	return &pb.RejectOfferResponse{
		Status:  "REJECTED",
		Message: "Đã từ chối báo giá, đơn hàng mở lại cho tài xế khác",
	}, nil
}
