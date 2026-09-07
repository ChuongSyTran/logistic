package http

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"gateway_service/internal/conf"
	"gateway_service/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/logistic/pkg/authn"
)

var (
	testPriv     *rsa.PrivateKey
	testSigner   *authn.Signer
	testVerifier *authn.Verifier
)

func init() {
	var err error
	testPriv, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	if testSigner, err = authn.NewSigner(testPriv); err != nil {
		panic(err)
	}
	if testVerifier, err = authn.NewVerifier(&testPriv.PublicKey); err != nil {
		panic(err)
	}
}

func testConfig() *conf.Config {
	return &conf.Config{
		Server: conf.ServerConfig{Port: "8080"},
		Upstreams: conf.UpstreamConfig{
			CallTimeout:              5 * time.Second,
			MaxConcurrentPerUpstream: 16,
		},
	}
}

func newTestEngine() *gin.Engine {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	RegisterGatewayRoutes(engine, Clients{}, middleware.NewAuthenticator(testVerifier), testConfig())
	return engine
}

func mintToken(t *testing.T, role string) string {
	t.Helper()
	pair, err := testSigner.Issue(uuid.Must(uuid.NewV7()), "test@logistic.vn", role)
	if err != nil {
		t.Fatalf("phát hành token: %v", err)
	}
	return pair.AccessToken
}