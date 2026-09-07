package biz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"auth_service/internal/entity"

	"github.com/google/uuid"
	"github.com/logistic/pkg/authn"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
)

type AuthService interface {
	Register(ctx context.Context, req entity.UserRegister) (*entity.UserProfile, error)
	Login(ctx context.Context, req entity.UserLogin) (*entity.AuthTokenPair, *entity.UserProfile, error)
	GetGoogleLoginURL(state string) string
	GoogleCallback(ctx context.Context, code string) (*entity.AuthTokenPair, *entity.UserProfile, error)
	VerifyToken(ctx context.Context, tokenString string) (*entity.UserProfile, error)
	RefreshToken(ctx context.Context, refreshToken string) (*entity.AuthTokenPair, error)
	Logout(ctx context.Context, refreshToken string) error
}

var (
	ErrEmailAlreadyExists  = errors.New("biz: email already registered")
	ErrInvalidCredentials  = errors.New("biz: invalid email or password")
	ErrTokenGenerationFail = errors.New("biz: failed to generate auth token")
	ErrInvalidToken        = errors.New("biz: invalid token")
	ErrSessionRevoked      = errors.New("biz: phiên đăng nhập đã bị thu hồi")
)

type authServiceImpl struct {
	authRepo    AuthRepo
	sessionRepo SessionRepo

	signer   *authn.Signer
	verifier *authn.Verifier

	oauthConfig *oauth2.Config
}

func NewAuthService(
	authRepo AuthRepo,
	sessionRepo SessionRepo,
	signer *authn.Signer,
	verifier *authn.Verifier,
	oauthConfig *oauth2.Config,
) AuthService {
	return &authServiceImpl{
		authRepo:    authRepo,
		sessionRepo: sessionRepo,
		signer:      signer,
		verifier:    verifier,
		oauthConfig: oauthConfig,
	}
}

func (s *authServiceImpl) Register(ctx context.Context, req entity.UserRegister) (*entity.UserProfile, error) {
	exists, err := s.authRepo.ExistsByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("biz register: checking email existence: %w", err)
	}
	if exists {
		return nil, ErrEmailAlreadyExists
	}

	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		return nil, fmt.Errorf("biz register: hashing password: %w", err)
	}

	profile, err := s.authRepo.Save(ctx, req, string(hashedBytes))
	if err != nil {
		return nil, fmt.Errorf("biz register: persisting user: %w", err)
	}

	return profile, nil
}

func (s *authServiceImpl) Login(ctx context.Context, req entity.UserLogin) (*entity.AuthTokenPair, *entity.UserProfile, error) {
	profile, hashedPassword, err := s.authRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		bcrypt.CompareHashAndPassword(dummyHash, []byte(req.Password))
		return nil, nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(req.Password)); err != nil {
		return nil, nil, ErrInvalidCredentials
	}

	tokenPair, err := s.issue(ctx, profile)
	if err != nil {
		return nil, nil, fmt.Errorf("biz login: %w: %v", ErrTokenGenerationFail, err)
	}

	return tokenPair, profile, nil
}

var dummyHash = []byte("$2a$12$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy")

func (s *authServiceImpl) GetGoogleLoginURL(state string) string {
	return s.oauthConfig.AuthCodeURL(state)
}

func (s *authServiceImpl) GoogleCallback(ctx context.Context, code string) (*entity.AuthTokenPair, *entity.UserProfile, error) {
	token, err := s.oauthConfig.Exchange(ctx, code)
	if err != nil {
		return nil, nil, fmt.Errorf("google callback: exchange code failed: %w", err)
	}

	client := s.oauthConfig.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, nil, fmt.Errorf("google callback: failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	var userInfo struct {
		Id    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, nil, fmt.Errorf("google callback: failed to decode user info: %w", err)
	}

	profile, _, err := s.authRepo.FindByEmail(ctx, userInfo.Email)
	if err != nil {
		profile, err = s.authRepo.Save(ctx, entity.UserRegister{
			Email:    userInfo.Email,
			Password: "",
			FullName: userInfo.Name,
			GoogleID: userInfo.Id,
			Role:     authn.RoleShipper,
		}, "")
		if err != nil {
			return nil, nil, fmt.Errorf("google callback: failed to create oauth user: %w", err)
		}
	}

	pair, err := s.issue(ctx, profile)
	if err != nil {
		return nil, nil, err
	}
	return pair, profile, nil
}

func (s *authServiceImpl) issue(ctx context.Context, profile *entity.UserProfile) (*entity.AuthTokenPair, error) {
	pair, err := s.signer.Issue(profile.Id, profile.Email, profile.Role)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTokenGenerationFail, err)
	}

	refreshID, err := uuid.Parse(pair.RefreshID)
	if err != nil {
		return nil, fmt.Errorf("%w: jti không hợp lệ: %v", ErrTokenGenerationFail, err)
	}

	if err := s.sessionRepo.Create(ctx, entity.RefreshSession{
		ID:        refreshID,
		UserID:    profile.Id,
		ExpiresAt: time.Now().Add(authn.DefaultRefreshTTL),
	}); err != nil {
		return nil, fmt.Errorf("biz issue: lưu phiên refresh: %w", err)
	}

	return &entity.AuthTokenPair{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresAt:    pair.ExpiresAt,
	}, nil
}

func (s *authServiceImpl) RefreshToken(ctx context.Context, refreshToken string) (*entity.AuthTokenPair, error) {
	claims, err := s.verifier.VerifyRefresh(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	sessionID, err := uuid.Parse(claims.ID)
	if err != nil {
		return nil, ErrInvalidToken
	}
	userID, err := claims.SubjectUUID()
	if err != nil {
		return nil, ErrInvalidToken
	}

	session, err := s.sessionRepo.Get(ctx, sessionID)
	if err != nil {
		return nil, ErrSessionRevoked
	}
	if !session.IsUsable(time.Now()) {
		return nil, ErrSessionRevoked
	}

	if session.UsedAt != nil {
		if err := s.sessionRepo.RevokeAllForUser(ctx, userID); err != nil {
			return nil, fmt.Errorf("biz refresh: thu hồi phiên sau khi phát hiện dùng lại: %w", err)
		}
		return nil, ErrSessionRevoked
	}

	profile, err := s.authRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, ErrInvalidToken
	}

	if err := s.sessionRepo.MarkUsed(ctx, sessionID); err != nil {
		return nil, fmt.Errorf("biz refresh: đánh dấu phiên đã dùng: %w", err)
	}

	return s.issue(ctx, profile)
}

func (s *authServiceImpl) Logout(ctx context.Context, refreshToken string) error {
	claims, err := s.verifier.VerifyRefresh(refreshToken)
	if err != nil {
		return nil
	}
	sessionID, err := uuid.Parse(claims.ID)
	if err != nil {
		return nil
	}
	return s.sessionRepo.Revoke(ctx, sessionID)
}

func (s *authServiceImpl) VerifyToken(ctx context.Context, tokenString string) (*entity.UserProfile, error) {
	claims, err := s.verifier.VerifyAccess(tokenString)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	userID, err := claims.SubjectUUID()
	if err != nil {
		return nil, ErrInvalidToken
	}

	profile, err := s.authRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("%w: không tìm thấy người dùng", ErrInvalidToken)
	}
	return profile, nil
}