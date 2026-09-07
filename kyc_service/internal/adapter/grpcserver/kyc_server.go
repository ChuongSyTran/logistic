package grpcserver

import (
	"context"

	pb "github.com/logistic/api/logistic/kyc_service/v1"
	"github.com/logistic/pkg/uuidx"
	"kyc_service/internal/app"
	"kyc_service/internal/entity"
	"kyc_service/internal/mapper"
)

type kycServer struct {
	pb.UnimplementedKycServiceServer
	useCase app.KycUseCase
	mapper  mapper.AppMapper
}

func NewKycServer(useCase app.KycUseCase, mapper mapper.AppMapper) pb.KycServiceServer {
	return &kycServer{useCase: useCase, mapper: mapper}
}

func (s *kycServer) SubmitKYC(ctx context.Context, req *pb.SubmitKYCRequest) (*pb.SubmitKYCResponse, error) {
	param, err := s.mapper.PbSubmitToParam(req)
	if err != nil {
		return nil, err
	}
	doc, err := s.useCase.SubmitKYC(ctx, param)
	if err != nil {
		return nil, err
	}
	return &pb.SubmitKYCResponse{
		Kyc: s.mapper.EntityToPb(doc),
	}, nil
}

func (s *kycServer) GetKYC(ctx context.Context, req *pb.GetKYCRequest) (*pb.GetKYCResponse, error) {
	userID, err := uuidx.FromBytes(req.GetUserId())
	if err != nil {
		return nil, entity.ErrInvalidUserID
	}
	doc, err := s.useCase.GetKYC(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &pb.GetKYCResponse{
		Kyc: s.mapper.EntityToPb(doc),
	}, nil
}

func (s *kycServer) ListPendingKYC(ctx context.Context, req *pb.ListPendingKYCRequest) (*pb.ListPendingKYCResponse, error) {
	res, err := s.useCase.ListPendingKYC(ctx, int(req.GetPage()), int(req.GetPageSize()))
	if err != nil {
		return nil, err
	}
	return &pb.ListPendingKYCResponse{
		Items:      s.mapper.EntityListToPb(res.Items),
		Pagination: s.mapper.PaginationToPb(res.Pagination),
	}, nil
}

func (s *kycServer) ReviewKYC(ctx context.Context, req *pb.ReviewKYCRequest) (*pb.ReviewKYCResponse, error) {
	param, err := s.mapper.PbReviewToParam(req)
	if err != nil {
		return nil, err
	}
	doc, err := s.useCase.ReviewKYC(ctx, param)
	if err != nil {
		return nil, err
	}
	return &pb.ReviewKYCResponse{
		Kyc: s.mapper.EntityToPb(doc),
	}, nil
}

func (s *kycServer) CountPendingKYC(ctx context.Context, req *pb.CountPendingKYCRequest) (*pb.CountPendingKYCResponse, error) {
	count, err := s.useCase.CountPendingKYC(ctx)
	if err != nil {
		return nil, err
	}
	return &pb.CountPendingKYCResponse{
		Count: count,
	}, nil
}
