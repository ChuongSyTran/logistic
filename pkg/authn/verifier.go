package authn

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Verifier struct {
	mu     sync.RWMutex
	keys   map[string]*rsa.PublicKey
	issuer string
}

func NewVerifier(pub *rsa.PublicKey, opts ...VerifierOption) (*Verifier, error) {
	if pub == nil {
		return nil, ErrNoKey
	}
	kid, err := KeyID(pub)
	if err != nil {
		return nil, err
	}

	v := &Verifier{
		keys:   map[string]*rsa.PublicKey{kid: pub},
		issuer: Issuer,
	}
	for _, opt := range opts {
		opt(v)
	}
	return v, nil
}

type VerifierOption func(*Verifier)

func WithVerifyIssuer(v string) VerifierOption {
	return func(ver *Verifier) { ver.issuer = v }
}

func (v *Verifier) AddKey(pub *rsa.PublicKey) error {
	kid, err := KeyID(pub)
	if err != nil {
		return err
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	v.keys[kid] = pub
	return nil
}

func (v *Verifier) VerifyAccess(token string) (*Claims, error) {
	return v.verify(token, TokenAccess)
}

func (v *Verifier) VerifyRefresh(token string) (*Claims, error) {
	return v.verify(token, TokenRefresh)
}

func (v *Verifier) verify(tokenStr string, want TokenType) (*Claims, error) {
	claims := &Claims{}

	_, err := jwt.ParseWithClaims(tokenStr, claims, v.keyFunc,

		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}),
		jwt.WithIssuer(v.issuer),
		jwt.WithExpirationRequired(),

		jwt.WithLeeway(30*time.Second),
	)
	if err != nil {
		switch {
		case errors.Is(err, jwt.ErrTokenExpired):
			return nil, ErrTokenExpired
		case errors.Is(err, jwt.ErrTokenInvalidIssuer):
			return nil, ErrWrongIssuer
		default:
			return nil, fmt.Errorf("%w: %v", ErrTokenInvalid, err)
		}
	}

	if claims.Type != want {
		return nil, ErrWrongTokenType
	}
	if _, err := uuid.Parse(claims.Subject); err != nil {
		return nil, fmt.Errorf("%w: subject không phải UUID", ErrTokenInvalid)
	}
	return claims, nil
}

func (v *Verifier) keyFunc(token *jwt.Token) (any, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if kid, ok := token.Header["kid"].(string); ok && kid != "" {
		key, found := v.keys[kid]
		if !found {
			return nil, fmt.Errorf("authn: không có public key cho kid=%s", kid)
		}
		return key, nil
	}
	if len(v.keys) != 1 {
		return nil, errors.New("authn: token thiếu kid mà đang có nhiều khoá")
	}
	for _, key := range v.keys {
		return key, nil
	}
	return nil, ErrNoKey
}

func (c *Claims) SubjectUUID() (uuid.UUID, error) {
	return uuid.Parse(c.Subject)
}