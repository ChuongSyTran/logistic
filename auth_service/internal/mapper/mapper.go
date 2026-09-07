package mapper

import (
	"time"

	"auth_service/ent"
	"auth_service/internal/entity"

	pb "github.com/logistic/api/logistic/auth_service/v1"

	"github.com/logistic/pkg/uuidx"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AuthMapper interface {
	ToUserProfile(source *ent.Users) *entity.UserProfile
	ToUserProfileProto(source *entity.UserProfile) *pb.UserProfile
	ToAuthTokenPairProto(source *entity.AuthTokenPair) *pb.AuthTokenPair
}

type authMapper struct{}

func NewAuthMapper() AuthMapper { return authMapper{} }

func (authMapper) ToUserProfile(source *ent.Users) *entity.UserProfile {
	if source == nil {
		return nil
	}
	return &entity.UserProfile{
		Id:        source.ID,
		Email:     source.Email,
		FullName:  source.FullName,
		Avatar:    source.Avatar,
		Role:      string(source.Role),
		CreatedAt: timePtr(source.CreatedAt),
		UpdatedAt: timePtr(source.UpdatedAt),
	}
}

func (authMapper) ToUserProfileProto(source *entity.UserProfile) *pb.UserProfile {
	if source == nil {
		return nil
	}
	return &pb.UserProfile{
		Id:        uuidx.ToBytes(source.Id),
		Email:     source.Email,
		FullName:  source.FullName,
		Avatar:    source.Avatar,
		Role:      source.Role,
		CreatedAt: toTimestamp(source.CreatedAt),
		UpdatedAt: toTimestamp(source.UpdatedAt),
	}
}

func (authMapper) ToAuthTokenPairProto(source *entity.AuthTokenPair) *pb.AuthTokenPair {
	if source == nil {
		return nil
	}
	return &pb.AuthTokenPair{
		AccessToken:  source.AccessToken,
		RefreshToken: source.RefreshToken,
		ExpiresAt:    source.ExpiresAt,
	}
}

func toTimestamp(t *time.Time) *timestamppb.Timestamp {
	if t == nil || t.IsZero() {
		return nil
	}
	return timestamppb.New(*t)
}

func timePtr(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}