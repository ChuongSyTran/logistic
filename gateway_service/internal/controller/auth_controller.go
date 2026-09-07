package controller

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"log"
	"net/http"
	"time"

	"gateway_service/internal/middleware"
	"gateway_service/internal/response"

	pb "github.com/logistic/api/logistic/auth_service/v1"
	pbuser "github.com/logistic/api/logistic/user_service/v1"
	"github.com/logistic/pkg/uuidx"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthController struct {
	authClient pb.AuthServiceClient
	userClient pbuser.UserServiceClient

	secureCookies bool
}

func NewAuthController(authClient pb.AuthServiceClient, userClient pbuser.UserServiceClient, isProduction bool) *AuthController {
	return &AuthController{
		authClient:    authClient,
		userClient:    userClient,
		secureCookies: isProduction,
	}
}

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	FullName string `json:"full_name"`
	Password string `json:"password" binding:"required,min=8"`
	Role     string `json:"role" binding:"omitempty,oneof=driver shipper"`
	Phone    string `json:"phone"`
}

// ensureProfile dựng hồ sơ user_service dưới đúng id auth_service cấp — không có
// nó thì token và hồ sơ trỏ về hai id khác nhau, mọi /users/* trả 404 hoặc 403.
// Gọi cả ở login để vá dần tài khoản cũ; đã có hồ sơ thì bỏ qua ALREADY_EXISTS.
func (c *AuthController) ensureProfile(ctx context.Context, profile *pb.UserProfile, phone string) {
	if c.userClient == nil || profile == nil || len(profile.Id) == 0 {
		return
	}

	_, err := c.userClient.RegisterUser(ctx, &pbuser.RegisterUserRequest{
		Id:       profile.Id,
		Email:    profile.Email,
		FullName: profile.GetFullName(),
		Role:     profile.Role,
		Phone:    phone,
	})
	if err == nil {
		return
	}
	if status.Code(err) == codes.AlreadyExists {
		return
	}

	log.Printf("[gateway] không dựng được hồ sơ user_service cho %s: %v — "+
		"các endpoint /api/v1/users/* sẽ trả 404 tới khi lần đăng nhập sau vá lại",
		uuidx.String(profile.Id), err)
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Register godoc
// @Summary      Đăng ký tài khoản
// @Description  Tạo tài khoản mới bằng email và mật khẩu. Gateway chuyển tiếp request tới auth_service qua gRPC.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request body RegisterRequest true "Thông tin đăng ký"
// @Success      201 {object} response.Envelope "Tạo tài khoản thành công"
// @Failure      400 {object} response.ErrorBody "Lỗi dữ liệu đầu vào"
// @Failure      409 {object} response.ErrorBody "Email đã tồn tại"
// @Router       /api/v1/auth/register [post]
func (c *AuthController) Register(ctx *gin.Context) {
	var req RegisterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, "VALIDATION_FAILED", err.Error())
		return
	}

	resp, err := c.authClient.Register(ctx.Request.Context(), &pb.RegisterRequest{
		Email:    req.Email,
		FullName: req.FullName,
		Password: req.Password,
		Role:     req.Role,
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}

	c.ensureProfile(ctx.Request.Context(), resp.Profile, req.Phone)

	response.Created(ctx, gin.H{"user": toAuthProfileDTO(resp.Profile)}, "Đăng ký tài khoản thành công")
}

// Login godoc
// @Summary      Đăng nhập
// @Description  Đăng nhập bằng email và mật khẩu. Trả về access_token, refresh_token và expires_at.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request body LoginRequest true "Thông tin đăng nhập"
// @Success      200 {object} response.Envelope "Đăng nhập thành công"
// @Failure      401 {object} response.ErrorBody "Sai email hoặc mật khẩu"
// @Router       /api/v1/auth/login [post]
func (c *AuthController) Login(ctx *gin.Context) {
	var req LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, "VALIDATION_FAILED", err.Error())
		return
	}

	resp, err := c.authClient.Login(ctx.Request.Context(), &pb.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}

	c.setAuthCookies(ctx, resp.TokenPair)
	c.ensureProfile(ctx.Request.Context(), resp.Profile, "")

	response.OK(ctx, gin.H{
		"access_token":  resp.TokenPair.AccessToken,
		"refresh_token": resp.TokenPair.RefreshToken,
		"expires_at":    resp.TokenPair.ExpiresAt,
		"user":          toAuthProfileDTO(resp.Profile),
	})
}

// Refresh godoc
// @Summary      Làm mới phiên đăng nhập
// @Description  Đổi refresh token lấy cặp token mới. Refresh token dùng MỘT LẦN — mỗi lần gọi trả về refresh token mới, token cũ hết hiệu lực ngay.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request body RefreshRequest false "Refresh token (nếu không dùng cookie)"
// @Success      200 {object} response.Envelope
// @Failure      401 {object} response.ErrorBody "Refresh token không hợp lệ"
// @Failure      403 {object} response.ErrorBody "Phiên đã bị thu hồi, cần đăng nhập lại"
// @Router       /api/v1/auth/refresh [post]
func (c *AuthController) Refresh(ctx *gin.Context) {
	token := c.extractRefreshToken(ctx)
	if token == "" {
		response.Unauthorized(ctx, "thiếu refresh token")
		return
	}

	resp, err := c.authClient.RefreshToken(ctx.Request.Context(), &pb.RefreshTokenRequest{
		RefreshToken: token,
	})
	if err != nil {
		c.clearAuthCookies(ctx)
		response.Error(ctx, err)
		return
	}

	c.setAuthCookies(ctx, resp.TokenPair)

	response.OK(ctx, gin.H{
		"access_token":  resp.TokenPair.AccessToken,
		"refresh_token": resp.TokenPair.RefreshToken,
		"expires_at":    resp.TokenPair.ExpiresAt,
	})
}

// Logout godoc
// @Summary      Đăng xuất
// @Description  Thu hồi refresh token của phiên hiện tại. Access token đang cầm vẫn còn hiệu lực tới khi hết hạn (tối đa 15 phút).
// @Tags         Auth
// @Produce      json
// @Success      200 {object} response.Envelope
// @Router       /api/v1/auth/logout [post]
func (c *AuthController) Logout(ctx *gin.Context) {
	token := c.extractRefreshToken(ctx)
	if token != "" {
		if _, err := c.authClient.Logout(ctx.Request.Context(), &pb.LogoutRequest{
			RefreshToken: token,
		}); err != nil {
			response.Error(ctx, err)
			return
		}
	}

	c.clearAuthCookies(ctx)
	response.OKMessage(ctx, nil, "Đã đăng xuất")
}

// GoogleLogin godoc
// @Summary      Đăng nhập bằng Google OAuth2
// @Description  Khởi tạo luồng OAuth2 với Google. Tạo state cookie để chống CSRF.
// @Tags         Auth - OAuth2
// @Produce      json
// @Success      307 {string} string "Redirect đến Google OAuth consent screen"
// @Router       /api/v1/auth/google/login [get]
func (c *AuthController) GoogleLogin(ctx *gin.Context) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		response.Error(ctx, err)
		return
	}
	state := base64.RawURLEncoding.EncodeToString(b)

	resp, err := c.authClient.GetGoogleLoginURL(ctx.Request.Context(), &pb.GetGoogleLoginURLRequest{
		State: state,
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}

	ctx.SetCookie("oauth_state", state, int(5*time.Minute/time.Second), "/", "", c.secureCookies, true)
	ctx.Redirect(http.StatusTemporaryRedirect, resp.Url)
}

// GoogleCallback godoc
// @Summary      Google OAuth2 Callback
// @Description  Xử lý callback từ Google, xác thực state, đổi code lấy token rồi set cookie.
// @Tags         Auth - OAuth2
// @Produce      json
// @Param        state query string true "OAuth state parameter (CSRF protection)"
// @Param        code  query string true "Authorization code từ Google"
// @Success      307 {string} string "Redirect về frontend với cookie đã set"
// @Router       /api/v1/auth/google/callback [get]
func (c *AuthController) GoogleCallback(ctx *gin.Context) {
	urlState := ctx.Query("state")
	cookieState, err := ctx.Cookie("oauth_state")
	if err != nil || urlState == "" || urlState != cookieState {
		response.BadRequest(ctx, "INVALID_OAUTH_STATE", "State không hợp lệ.")
		return
	}

	ctx.SetCookie("oauth_state", "", -1, "/", "", c.secureCookies, true)

	resp, err := c.authClient.GoogleCallback(ctx.Request.Context(), &pb.GoogleCallbackRequest{
		Code: ctx.Query("code"),
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}

	c.setAuthCookies(ctx, resp.TokenPair)
	ctx.Redirect(http.StatusTemporaryRedirect, "http://127.0.0.1:3000/")
}

// GetInfo godoc
// @Summary      Lấy thông tin người dùng hiện tại
// @Description  Trả về profile của chủ nhân access token.
// @Tags         Auth
// @Produce      json
// @Param        Authorization header string false "Bearer token (nếu không dùng cookie)"
// @Success      200 {object} response.Envelope
// @Failure      401 {object} response.ErrorBody
// @Router       /api/v1/auth/me [get]
func (c *AuthController) GetInfo(ctx *gin.Context) {
	identity, ok := middleware.CurrentIdentity(ctx)
	if !ok {
		response.Unauthorized(ctx, "cần đăng nhập")
		return
	}

	resp, err := c.authClient.VerifyToken(ctx.Request.Context(), &pb.VerifyTokenRequest{
		Token: extractAccessToken(ctx),
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}

	profile := toAuthProfileDTO(resp.Profile)

	profile.Role = identity.Role

	response.OKMessage(ctx, gin.H{"user": profile}, "Get info successfully")
}

type AuthProfileDTO struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
	Avatar   string `json:"avatar"`
	Role     string `json:"role"`
}

func toAuthProfileDTO(p *pb.UserProfile) *AuthProfileDTO {
	if p == nil {
		return nil
	}
	dto := &AuthProfileDTO{
		ID:    uuidx.String(p.Id),
		Email: p.Email,
		Role:  p.Role,
	}
	if p.FullName != nil {
		dto.FullName = *p.FullName
	}
	if p.Avatar != nil {
		dto.Avatar = *p.Avatar
	}
	return dto
}

func (c *AuthController) setAuthCookies(ctx *gin.Context, pair *pb.AuthTokenPair) {
	if pair == nil {
		return
	}

	ctx.SetSameSite(http.SameSiteLaxMode)

	accessMaxAge := int(time.Until(time.Unix(pair.ExpiresAt, 0)).Seconds())
	if accessMaxAge < 0 {
		accessMaxAge = 0
	}
	refreshMaxAge := int(7 * 24 * time.Hour / time.Second)

	ctx.SetCookie("access_token", pair.AccessToken, accessMaxAge, "/", "", c.secureCookies, true)
	ctx.SetCookie("refresh_token", pair.RefreshToken, refreshMaxAge, "/", "", c.secureCookies, true)
}

func (c *AuthController) clearAuthCookies(ctx *gin.Context) {
	ctx.SetSameSite(http.SameSiteLaxMode)
	ctx.SetCookie("access_token", "", -1, "/", "", c.secureCookies, true)
	ctx.SetCookie("refresh_token", "", -1, "/", "", c.secureCookies, true)
}

func (c *AuthController) extractRefreshToken(ctx *gin.Context) string {
	var req RefreshRequest
	if err := ctx.ShouldBindJSON(&req); err == nil && req.RefreshToken != "" {
		return req.RefreshToken
	}
	if cookie, err := ctx.Cookie("refresh_token"); err == nil {
		return cookie
	}
	return ""
}

func extractAccessToken(ctx *gin.Context) string {
	if h := ctx.GetHeader("Authorization"); len(h) > 7 {
		return h[7:]
	}
	if cookie, err := ctx.Cookie("access_token"); err == nil {
		return cookie
	}
	return ""
}
