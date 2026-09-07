package mapper

import (
	"time"

	pb "github.com/logistic/api/logistic/kyc_service/v1"
	"github.com/logistic/pkg/uuidx"
	"kyc_service/internal/entity"
)

type AppMapper interface {
	EntityToPb(doc *entity.KycDocument) *pb.KycDocument
	EntityListToPb(docs []entity.KycDocument) []*pb.KycDocument
	PaginationToPb(p entity.KycPagination) *pb.KycPagination
	PbSubmitToParam(req *pb.SubmitKYCRequest) (*entity.SubmitKYCParam, error)
	PbReviewToParam(req *pb.ReviewKYCRequest) (*entity.ReviewKYCParam, error)
}

type AppMapperImpl struct{}

func NewAppMapper() AppMapper {
	return &AppMapperImpl{}
}

func (m *AppMapperImpl) EntityToPb(doc *entity.KycDocument) *pb.KycDocument {
	if doc == nil {
		return nil
	}
	pbDoc := &pb.KycDocument{
		Id:               doc.ID.String(),
		UserId:           uuidx.ToBytes(doc.UserID),
		IdCardNumber:     doc.IDCardNumber,
		LicenseNumber:    doc.LicenseNumber,
		IdCardFrontUrl:   doc.IDCardFrontURL,
		IdCardBackUrl:    doc.IDCardBackURL,
		LicenseFrontUrl:  doc.LicenseFrontURL,
		LicenseBackUrl:   doc.LicenseBackURL,
		Status:           doc.Status,
		Note:             doc.Note,
		CreatedAt:        doc.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        doc.UpdatedAt.Format(time.RFC3339),
	}
	if doc.ReviewerID != nil {
		pbDoc.ReviewerId = uuidx.ToBytes(*doc.ReviewerID)
	}
	if doc.ReviewedAt != nil {
		pbDoc.ReviewedAt = doc.ReviewedAt.Format(time.RFC3339)
	}
	return pbDoc
}

func (m *AppMapperImpl) EntityListToPb(docs []entity.KycDocument) []*pb.KycDocument {
	res := make([]*pb.KycDocument, len(docs))
	for i := range docs {
		res[i] = m.EntityToPb(&docs[i])
	}
	return res
}

func (m *AppMapperImpl) PaginationToPb(p entity.KycPagination) *pb.KycPagination {
	return &pb.KycPagination{
		Page:       int32(p.Page),
		PageSize:   int32(p.PageSize),
		TotalItems: p.TotalItems,
		TotalPages: int32(p.TotalPages),
	}
}

func (m *AppMapperImpl) PbSubmitToParam(req *pb.SubmitKYCRequest) (*entity.SubmitKYCParam, error) {
	userID, err := uuidx.FromBytes(req.GetUserId())
	if err != nil {
		return nil, entity.ErrInvalidUserID
	}
	return &entity.SubmitKYCParam{
		UserID:          userID,
		IDCardNumber:    req.GetIdCardNumber(),
		LicenseNumber:   req.GetLicenseNumber(),
		IDCardFrontURL:  req.GetIdCardFrontUrl(),
		IDCardBackURL:   req.GetIdCardBackUrl(),
		LicenseFrontURL: req.GetLicenseFrontUrl(),
		LicenseBackURL:  req.GetLicenseBackUrl(),
		Status:          req.GetStatus(),
		Note:            req.GetNote(),
	}, nil
}

func (m *AppMapperImpl) PbReviewToParam(req *pb.ReviewKYCRequest) (*entity.ReviewKYCParam, error) {
	userID, err := uuidx.FromBytes(req.GetUserId())
	if err != nil {
		return nil, entity.ErrInvalidUserID
	}
	reviewerID, err := uuidx.FromBytes(req.GetReviewerId())
	if err != nil {
		// reviewer can be optional or nil if not provided
		reviewerID = userID
	}
	return &entity.ReviewKYCParam{
		UserID:     userID,
		Approved:   req.GetApproved(),
		Note:       req.GetNote(),
		ReviewerID: reviewerID,
	}, nil
}
