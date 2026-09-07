package controller

import (
	"context"
	"encoding/base64"
	"errors"
	"math/big"

	"auth_service/internal/biz"
	"auth_service/internal/entity"
	"auth_service/internal/mapper"

	pb "github.com/logistic/api/logistic/auth_service/v1"
	"github.com/logistic/pkg/authn"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type authController struct {
	pb.UnimplementedAuthServiceServer
	authBiz biz.AuthService
	mapper  mapper.AuthMapper
	signer  *authn.Signer
}

func NewAuthController(authBiz biz.AuthService, m mapper.AuthMapper, signer *authn.Signer) pb.AuthServiceServer {
	return &authController{
		authBiz: authBiz,
		mapper:  m,
		signer:  signer,
	}
}

func (h *authController) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}

	profile, err := h.authBiz.Register(ctx, entity.UserRegister{
		Email:    req.Email,
		FullName: req.FullName,
		Password: req.Password,
		GoogleID: req.GoogleId,
		Role:     req.Role,
	})
	if err != nil {
		if errors.Is(err, biz.ErrEmailAlreadyExists) {
			return nil, status.Error(codes.AlreadyExists, "email đã được đăng ký")
		}

		return nil, status.Error(codes.Internal, "không tạo được tài khoản")
	}

	return &pb.RegisterResponse{
		Profile: h.mapper.ToUserProfileProto(profile),
	}, nil
}

func (h *authController) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}

	tokenPair, profile, err := h.authBiz.Login(ctx, entity.UserLogin{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		if errors.Is(err, biz.ErrInvalidCredentials) {
			return nil, status.Error(codes.Unauthenticated, "sai email hoặc mật khẩu")
		}
		return nil, status.Error(codes.Internal, "không đăng nhập được")
	}

	return &pb.LoginResponse{
		TokenPair: h.mapper.ToAuthTokenPairProto(tokenPair),
		Profile:   h.mapper.ToUserProfileProto(profile),
	}, nil
}

func (h *authController) GetGoogleLoginURL(ctx context.Context, req *pb.GetGoogleLoginURLRequest) (*pb.GetGoogleLoginURLResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}

	return &pb.GetGoogleLoginURLResponse{
		Url: h.authBiz.GetGoogleLoginURL(req.State),
	}, nil
}

func (h *authController) GoogleCallback(ctx context.Context, req *pb.GoogleCallbackRequest) (*pb.GoogleCallbackResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}

	tokenPair, profile, err := h.authBiz.GoogleCallback(ctx, req.Code)
	if err != nil {
		return nil, status.Error(codes.Internal, "không hoàn tất đăng nhập Google")
	}

	return &pb.GoogleCallbackResponse{
		TokenPair: h.mapper.ToAuthTokenPairProto(tokenPair),
		Profile:   h.mapper.ToUserProfileProto(profile),
	}, nil
}

func (h *authController) VerifyToken(ctx context.Context, req *pb.VerifyTokenRequest) (*pb.VerifyTokenResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}

	profile, err := h.authBiz.VerifyToken(ctx, req.Token)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "token không hợp lệ hoặc đã hết hạn")
	}

	return &pb.VerifyTokenResponse{
		Profile: h.mapper.ToUserProfileProto(profile),
	}, nil
}

func (h *authController) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	if req == nil || req.RefreshToken == "" {
		return nil, status.Error(codes.InvalidArgument, "thiếu refresh token")
	}

	pair, err := h.authBiz.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		switch {
		case errors.Is(err, biz.ErrSessionRevoked):

			return nil, status.Error(codes.PermissionDenied, "phiên đã bị thu hồi, cần đăng nhập lại")
		case errors.Is(err, biz.ErrInvalidToken):
			return nil, status.Error(codes.Unauthenticated, "refresh token không hợp lệ")
		default:
			return nil, status.Error(codes.Internal, "không làm mới được phiên")
		}
	}

	return &pb.RefreshTokenResponse{
		TokenPair: h.mapper.ToAuthTokenPairProto(pair),
	}, nil
}

func (h *authController) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	if req == nil || req.RefreshToken == "" {
		return nil, status.Error(codes.InvalidArgument, "thiếu refresh token")
	}

	if err := h.authBiz.Logout(ctx, req.RefreshToken); err != nil {
		return nil, status.Error(codes.Internal, "không đăng xuất được")
	}
	return &pb.LogoutResponse{Success: true}, nil
}

func (h *authController) GetPublicKeys(ctx context.Context, req *pb.GetPublicKeysRequest) (*pb.GetPublicKeysResponse, error) {
	pub := h.signer.PublicKey()

	return &pb.GetPublicKeysResponse{
		Keys: []*pb.JSONWebKey{{
			Kid: h.signer.KeyID(),
			Kty: "RSA",
			Alg: "RS256",
			Use: "sig",
			N:   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
			E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
		}},
	}, nil
}