package di

import (
	"context"
	"fmt"

	"auth_service/internal/adapter/grpcserver"
	"auth_service/internal/adapter/persistence"
	"auth_service/internal/app"
	"auth_service/internal/conf"
	"auth_service/internal/mapper"

	pb "github.com/logistic/api/logistic/auth_service/v1"
	"github.com/logistic/pkg/authn"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/grpc"
)

func Injection(grpcServer *grpc.Server, cfg *conf.Config) error {
	clientDb, err := persistence.NewConnection(cfg.Database)
	if err != nil {
		return err
	}

	oauthConfig := &oauth2.Config{
		ClientID:     cfg.Google.ClientID,
		ClientSecret: cfg.Google.ClientSecret,
		RedirectURL:  cfg.Google.RedirectURL,
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}

	privateKey, err := authn.LoadPrivateKey(cfg.JWT.PrivateKey)
	if err != nil {
		return fmt.Errorf("auth_service: nạp private key: %w", err)
	}

	signer, err := authn.NewSigner(privateKey,
		authn.WithAccessTTL(cfg.JWT.AccessTTL),
		authn.WithRefreshTTL(cfg.JWT.RefreshTTL),
	)
	if err != nil {
		return fmt.Errorf("auth_service: dựng signer: %w", err)
	}

	verifier, err := authn.NewVerifier(signer.PublicKey())
	if err != nil {
		return fmt.Errorf("auth_service: dựng verifier: %w", err)
	}

	authMapper := mapper.NewAuthMapper()
	authRepo := persistence.NewAuthRepo(clientDb, authMapper)
	sessionRepo := persistence.NewSessionRepo(clientDb)
	authService := app.NewAuthService(authRepo, sessionRepo, signer, verifier, oauthConfig)

	bootstrapAdmin(context.Background(), authService, cfg.Bootstrap)

	pb.RegisterAuthServiceServer(grpcServer, grpcserver.NewAuthServer(authService, authMapper, signer))

	return nil
}
