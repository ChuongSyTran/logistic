package grpcserver

import (
	"context"

	"vehicle_service/internal/app"
	"vehicle_service/internal/entity"
	"vehicle_service/internal/mapper"

	"github.com/google/uuid"
	pb "github.com/logistic/api/logistic/vehicle_service/v1"
	"github.com/logistic/pkg/uuidx"
)

type vehicleServer struct {
	pb.UnimplementedVehicleServiceServer
	engine app.VehicleEngine
	mapper mapper.AppMapper
}

func NewVehicleServer(engine app.VehicleEngine, appMapper mapper.AppMapper) pb.VehicleServiceServer {
	return &vehicleServer{engine: engine, mapper: appMapper}
}

func parseID(raw []byte, invalid error) (uuid.UUID, error) {
	id, err := uuidx.FromBytes(raw)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, invalid
	}
	return id, nil
}

func parseOptionalID(raw []byte, invalid error) (uuid.UUID, error) {
	if len(raw) == 0 {
		return uuid.Nil, nil
	}
	return parseID(raw, invalid)
}

func (s *vehicleServer) RegisterVehicle(ctx context.Context, req *pb.RegisterVehicleRequest) (*pb.RegisterVehicleResponse, error) {
	param, err := s.mapper.PbRegisterVehicleToParam(req)
	if err != nil {
		return nil, entity.ErrInvalidDriverID.WithCause(err)
	}

	v, err := s.engine.RegisterVehicle(ctx, &param)
	if err != nil {
		return nil, err
	}

	return &pb.RegisterVehicleResponse{
		Id:      uuidx.ToBytes(v.ID),
		Message: "Đăng ký phương tiện thành công",
		Vehicle: s.mapper.EntityVehicleToPbVehicle(*v),
	}, nil
}

func (s *vehicleServer) GetVehicle(ctx context.Context, req *pb.GetVehicleRequest) (*pb.GetVehicleResponse, error) {
	id, err := parseID(req.Id, entity.ErrInvalidVehicleID)
	if err != nil {
		return nil, err
	}

	driverID, err := parseOptionalID(req.DriverId, entity.ErrInvalidDriverID)
	if err != nil {
		return nil, err
	}

	v, err := s.engine.GetVehicle(ctx, id, driverID)
	if err != nil {
		return nil, err
	}
	return &pb.GetVehicleResponse{Vehicle: s.mapper.EntityVehicleToPbVehicle(*v)}, nil
}

func (s *vehicleServer) ListVehicles(ctx context.Context, req *pb.ListVehiclesRequest) (*pb.ListVehiclesResponse, error) {
	param, err := s.mapper.PbListVehiclesToParam(req)
	if err != nil {
		return nil, entity.ErrInvalidDriverID.WithCause(err)
	}

	res, err := s.engine.ListVehicles(ctx, &param)
	if err != nil {
		return nil, err
	}

	return &pb.ListVehiclesResponse{
		Vehicles:   s.mapper.EntityVehicleListToPbVehicleList(res.Vehicles),
		Pagination: s.mapper.EntityPaginationToPb(res.Pagination),
	}, nil
}

func (s *vehicleServer) UpdateVehicle(ctx context.Context, req *pb.UpdateVehicleRequest) (*pb.UpdateVehicleResponse, error) {
	param, err := s.mapper.PbUpdateVehicleToParam(req)
	if err != nil {
		return nil, entity.ErrInvalidVehicleID.WithCause(err)
	}

	v, err := s.engine.UpdateVehicle(ctx, &param)
	if err != nil {
		return nil, err
	}

	return &pb.UpdateVehicleResponse{
		Vehicle: s.mapper.EntityVehicleToPbVehicle(*v),
		Message: "Cập nhật phương tiện thành công",
	}, nil
}

func (s *vehicleServer) DeleteVehicle(ctx context.Context, req *pb.DeleteVehicleRequest) (*pb.DeleteVehicleResponse, error) {
	id, err := parseID(req.Id, entity.ErrInvalidVehicleID)
	if err != nil {
		return nil, err
	}
	driverID, err := parseOptionalID(req.DriverId, entity.ErrInvalidDriverID)
	if err != nil {
		return nil, err
	}

	if err := s.engine.DeleteVehicle(ctx, id, driverID); err != nil {
		return nil, err
	}
	return &pb.DeleteVehicleResponse{Message: "Xoá phương tiện thành công"}, nil
}

func (s *vehicleServer) UpdateVehicleStatus(ctx context.Context, req *pb.UpdateVehicleStatusRequest) (*pb.UpdateVehicleStatusResponse, error) {
	id, err := parseID(req.Id, entity.ErrInvalidVehicleID)
	if err != nil {
		return nil, err
	}

	driverID, err := parseOptionalID(req.DriverId, entity.ErrInvalidDriverID)
	if err != nil {
		return nil, err
	}

	v, err := s.engine.UpdateVehicleStatus(ctx, id, driverID, req.Status)
	if err != nil {
		return nil, err
	}

	return &pb.UpdateVehicleStatusResponse{
		Message: "Cập nhật trạng thái phương tiện thành công",
		Vehicle: s.mapper.EntityVehicleToPbVehicle(*v),
	}, nil
}

func (s *vehicleServer) UploadVehicleDocument(ctx context.Context, req *pb.UploadVehicleDocumentRequest) (*pb.UploadVehicleDocumentResponse, error) {
	param, err := s.mapper.PbUploadDocumentToParam(req)
	if err != nil {
		return nil, entity.ErrInvalidVehicleID.WithCause(err)
	}

	doc, err := s.engine.UploadDocument(ctx, &param)
	if err != nil {
		return nil, err
	}

	return &pb.UploadVehicleDocumentResponse{
		Document: s.mapper.EntityDocumentToPb(*doc),
		Message:  "Tải lên giấy tờ thành công, đang chờ duyệt",
	}, nil
}

func (s *vehicleServer) ListVehicleDocuments(ctx context.Context, req *pb.ListVehicleDocumentsRequest) (*pb.ListVehicleDocumentsResponse, error) {
	param, err := s.mapper.PbListDocumentsToParam(req)
	if err != nil {
		return nil, entity.ErrInvalidVehicleID.WithCause(err)
	}

	docs, err := s.engine.ListDocuments(ctx, &param)
	if err != nil {
		return nil, err
	}
	return &pb.ListVehicleDocumentsResponse{Documents: s.mapper.EntityDocumentListToPbList(docs)}, nil
}

func (s *vehicleServer) DeleteVehicleDocument(ctx context.Context, req *pb.DeleteVehicleDocumentRequest) (*pb.DeleteVehicleDocumentResponse, error) {
	id, err := parseID(req.Id, entity.ErrInvalidDocumentID)
	if err != nil {
		return nil, err
	}
	driverID, err := parseOptionalID(req.DriverId, entity.ErrInvalidDriverID)
	if err != nil {
		return nil, err
	}
	if err := s.engine.DeleteDocument(ctx, id, driverID); err != nil {
		return nil, err
	}
	return &pb.DeleteVehicleDocumentResponse{Message: "Xoá giấy tờ thành công"}, nil
}

func (s *vehicleServer) ReportLocation(ctx context.Context, req *pb.ReportLocationRequest) (*pb.ReportLocationResponse, error) {
	param, err := s.mapper.PbReportLocationToParam(req)
	if err != nil {
		return nil, entity.ErrInvalidVehicleID.WithCause(err)
	}

	loc, err := s.engine.ReportLocation(ctx, &param)
	if err != nil {
		return nil, err
	}

	return &pb.ReportLocationResponse{
		Message: "Cập nhật vị trí thành công",
		ZoneId:  loc.ZoneID,
	}, nil
}

func (s *vehicleServer) GetVehicleLocation(ctx context.Context, req *pb.GetVehicleLocationRequest) (*pb.GetVehicleLocationResponse, error) {
	id, err := parseID(req.VehicleId, entity.ErrInvalidVehicleID)
	if err != nil {
		return nil, err
	}

	driverID, err := parseOptionalID(req.DriverId, entity.ErrInvalidDriverID)
	if err != nil {
		return nil, err
	}

	loc, err := s.engine.GetLocation(ctx, id, driverID)
	if err != nil {
		return nil, err
	}
	return &pb.GetVehicleLocationResponse{Location: s.mapper.EntityLocationToPb(*loc)}, nil
}

func (s *vehicleServer) SetDriverAvailability(ctx context.Context, req *pb.SetDriverAvailabilityRequest) (*pb.SetDriverAvailabilityResponse, error) {
	param, err := s.mapper.PbSetAvailabilityToParam(req)
	if err != nil {
		return nil, entity.ErrInvalidDriverID.WithCause(err)
	}

	avail, err := s.engine.SetAvailability(ctx, &param)
	if err != nil {
		return nil, err
	}

	message := "Đã tắt nhận đơn"
	if avail.IsOnline {
		message = "Đã bật nhận đơn, xe của bạn sẽ xuất hiện trong kết quả tìm kiếm"
	}

	return &pb.SetDriverAvailabilityResponse{
		Availability: s.mapper.EntityAvailabilityToPb(*avail),
		Message:      message,
	}, nil
}

func (s *vehicleServer) GetDriverAvailability(ctx context.Context, req *pb.GetDriverAvailabilityRequest) (*pb.GetDriverAvailabilityResponse, error) {
	id, err := parseID(req.DriverId, entity.ErrInvalidDriverID)
	if err != nil {
		return nil, err
	}

	avail, err := s.engine.GetAvailability(ctx, id)
	if err != nil {
		return nil, err
	}
	return &pb.GetDriverAvailabilityResponse{Availability: s.mapper.EntityAvailabilityToPb(*avail)}, nil
}

func (s *vehicleServer) SearchNearbyVehicles(ctx context.Context, req *pb.SearchNearbyVehiclesRequest) (*pb.SearchNearbyVehiclesResponse, error) {
	param := s.mapper.PbSearchNearbyToParam(req)

	list, err := s.engine.SearchNearby(ctx, &param)
	if err != nil {
		return nil, err
	}

	return &pb.SearchNearbyVehiclesResponse{
		Vehicles:   s.mapper.EntityNearbyListToPbList(list),
		TotalFound: int32(len(list)),
	}, nil
}

func (s *vehicleServer) AdminListVehicles(ctx context.Context, req *pb.AdminListVehiclesRequest) (*pb.AdminListVehiclesResponse, error) {
	param := s.mapper.PbAdminListVehiclesToParam(req)

	res, err := s.engine.AdminListVehicles(ctx, &param)
	if err != nil {
		return nil, err
	}

	return &pb.AdminListVehiclesResponse{
		Vehicles:   s.mapper.EntityVehicleListToPbVehicleList(res.Vehicles),
		Pagination: s.mapper.EntityPaginationToPb(res.Pagination),
	}, nil
}

func (s *vehicleServer) AdminVerifyVehicle(ctx context.Context, req *pb.AdminVerifyVehicleRequest) (*pb.AdminVerifyVehicleResponse, error) {
	param, err := s.mapper.PbVerifyVehicleToParam(req)
	if err != nil {
		return nil, entity.ErrInvalidVehicleID.WithCause(err)
	}
	if param.ReviewerID, err = parseOptionalID(req.ReviewerId, entity.ErrInvalidDriverID); err != nil {
		return nil, err
	}

	v, err := s.engine.AdminVerifyVehicle(ctx, &param)
	if err != nil {
		return nil, err
	}

	message := "Đã từ chối duyệt phương tiện"
	if req.Approved {
		message = "Đã duyệt phương tiện"
	}

	return &pb.AdminVerifyVehicleResponse{
		Vehicle: s.mapper.EntityVehicleToPbVehicle(*v),
		Message: message,
	}, nil
}

func (s *vehicleServer) AdminListPendingDocuments(ctx context.Context, req *pb.AdminListPendingDocumentsRequest) (*pb.AdminListPendingDocumentsResponse, error) {
	res, err := s.engine.AdminListPendingDocuments(ctx, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, err
	}

	return &pb.AdminListPendingDocumentsResponse{
		Documents:  s.mapper.EntityDocumentListToPbList(res.Documents),
		Pagination: s.mapper.EntityPaginationToPb(res.Pagination),
	}, nil
}

func (s *vehicleServer) AdminReviewDocument(ctx context.Context, req *pb.AdminReviewDocumentRequest) (*pb.AdminReviewDocumentResponse, error) {
	param, err := s.mapper.PbReviewDocumentToParam(req)
	if err != nil {
		return nil, entity.ErrInvalidDocumentID.WithCause(err)
	}
	if param.ReviewerID, err = parseOptionalID(req.ReviewerId, entity.ErrInvalidDriverID); err != nil {
		return nil, err
	}

	doc, err := s.engine.AdminReviewDocument(ctx, &param)
	if err != nil {
		return nil, err
	}

	message := "Đã từ chối giấy tờ"
	if req.Approved {
		message = "Đã duyệt giấy tờ"
	}

	return &pb.AdminReviewDocumentResponse{
		Document: s.mapper.EntityDocumentToPb(*doc),
		Message:  message,
	}, nil
}

func (s *vehicleServer) AdminGetVehicleStats(ctx context.Context, _ *pb.AdminGetVehicleStatsRequest) (*pb.AdminGetVehicleStatsResponse, error) {
	stats, err := s.engine.AdminGetStats(ctx)
	if err != nil {
		return nil, err
	}

	return &pb.AdminGetVehicleStatsResponse{
		TotalVehicles:       stats.TotalVehicles,
		ActiveVehicles:      stats.ActiveVehicles,
		MaintenanceVehicles: stats.MaintenanceVehicles,
		PendingVerification: stats.PendingVerification,
		OnlineDrivers:       stats.OnlineDrivers,
		PendingDocuments:    stats.PendingDocuments,
	}, nil
}
