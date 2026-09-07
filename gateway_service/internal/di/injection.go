package di

import (
	"fmt"
	"log"

	"gateway_service/internal/adapter/httpserver"
	"gateway_service/internal/conf"
	"gateway_service/internal/middleware"

	pbauth "github.com/logistic/api/logistic/auth_service/v1"
	pbmatching "github.com/logistic/api/logistic/matching_service/v1"
	pbmedia "github.com/logistic/api/logistic/media_service/v1"
	pbnotification "github.com/logistic/api/logistic/notification_service/v1"
	pbuser "github.com/logistic/api/logistic/user_service/v1"
	pbvehicle "github.com/logistic/api/logistic/vehicle_service/v1"
	pbkyc "github.com/logistic/api/logistic/kyc_service/v1"
	"github.com/logistic/pkg/authn"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Container struct {
	conns []*grpc.ClientConn
}

func (c *Container) Close() {
	if c == nil {
		return
	}
	for _, conn := range c.conns {
		if conn == nil {
			continue
		}
		if err := conn.Close(); err != nil {
			log.Printf("[gateway] closing grpc conn failed: %v", err)
		}
	}
}

const roundRobinConfig = `{
  "loadBalancingConfig": [{"round_robin":{}}],
  "healthCheckConfig": {"serviceName": ""}
}`

func dial(cfg *conf.Config, name, addr string) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(
		"dns:///"+addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(roundRobinConfig),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
		grpc.WithChainUnaryInterceptor(
			concurrencyLimiter(cfg.Upstreams.MaxConcurrentPerUpstream),
			timeoutInterceptor(cfg.Upstreams.CallTimeout),
			authn.OutgoingIdentityInterceptor(),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("gateway: không tạo được client gRPC tới %s (%s): %w", name, addr, err)
	}

	log.Printf("[gateway] client gRPC %s -> %s (round_robin, timeout=%s, max_inflight=%d)",
		name, addr, cfg.Upstreams.CallTimeout, cfg.Upstreams.MaxConcurrentPerUpstream)
	return conn, nil
}

func Injection(ginEngine *gin.Engine, cfg *conf.Config) (*Container, error) {
	container := &Container{}

	verifier, err := buildVerifier(cfg.Auth)
	if err != nil {
		return nil, err
	}
	authenticator := middleware.NewAuthenticator(verifier)

	targets := []struct {
		name string
		addr string
	}{
		{"auth_service", cfg.Upstreams.Auth},
		{"media_service", cfg.Upstreams.Media},
		{"matching_service", cfg.Upstreams.Matching},
		{"user_service", cfg.Upstreams.User},
		{"vehicle_service", cfg.Upstreams.Vehicle},
		{"notification_service", cfg.Upstreams.Notification},
		{"kyc_service", cfg.Upstreams.Kyc},
	}

	conns := make([]*grpc.ClientConn, 0, len(targets))
	for _, t := range targets {
		conn, err := dial(cfg, t.name, t.addr)
		if err != nil {
			container.Close()
			return nil, err
		}
		conns = append(conns, conn)
		container.conns = append(container.conns, conn)
	}

	httpserver.RegisterGatewayRoutes(ginEngine, httpserver.Clients{
		Auth:         pbauth.NewAuthServiceClient(conns[0]),
		Media:        pbmedia.NewMediaServiceClient(conns[1]),
		Matching:     pbmatching.NewMatchingEngineServiceClient(conns[2]),
		User:         pbuser.NewUserServiceClient(conns[3]),
		Vehicle:      pbvehicle.NewVehicleServiceClient(conns[4]),
		Notification: pbnotification.NewNotificationServiceClient(conns[5]),
		Kyc:          pbkyc.NewKycServiceClient(conns[6]),
	}, authenticator, cfg)

	return container, nil
}

func buildVerifier(cfg conf.AuthConfig) (*authn.Verifier, error) {
	pub, err := authn.LoadPublicKey(cfg.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("gateway: nạp public key: %w", err)
	}

	verifier, err := authn.NewVerifier(pub)
	if err != nil {
		return nil, fmt.Errorf("gateway: dựng verifier: %w", err)
	}

	if cfg.PreviousPublicKey != "" {
		prev, err := authn.LoadPublicKey(cfg.PreviousPublicKey)
		if err != nil {
			return nil, fmt.Errorf("gateway: nạp public key cũ: %w", err)
		}
		if err := verifier.AddKey(prev); err != nil {
			return nil, fmt.Errorf("gateway: thêm public key cũ: %w", err)
		}
		log.Println("[gateway] đang xoay khoá: chấp nhận cả khoá cũ và khoá mới")
	}

	return verifier, nil
}