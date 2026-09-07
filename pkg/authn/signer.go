package authn

import (
	"crypto/rsa"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/logistic/pkg/uuidx"
)

type Signer struct {
	priv       *rsa.PrivateKey
	kid        string
	issuer     string
	accessTTL  time.Duration
	refreshTTL time.Duration
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string

	ExpiresAt int64

	RefreshID string
}

type SignerOption func(*Signer)

func WithAccessTTL(d time.Duration) SignerOption  { return func(s *Signer) { s.accessTTL = d } }
func WithRefreshTTL(d time.Duration) SignerOption { return func(s *Signer) { s.refreshTTL = d } }
func WithIssuer(v string) SignerOption            { return func(s *Signer) { s.issuer = v } }

func NewSigner(priv *rsa.PrivateKey, opts ...SignerOption) (*Signer, error) {
	if priv == nil {
		return nil, ErrNoKey
	}

	if priv.N.BitLen() < 2048 {
		return nil, fmt.Errorf("authn: RSA key chỉ %d bit, tối thiểu phải 2048", priv.N.BitLen())
	}

	kid, err := KeyID(&priv.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("authn: tính kid: %w", err)
	}

	s := &Signer{
		priv:       priv,
		kid:        kid,
		issuer:     Issuer,
		accessTTL:  DefaultAccessTTL,
		refreshTTL: DefaultRefreshTTL,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s, nil
}

func (s *Signer) KeyID() string { return s.kid }

func (s *Signer) PublicKey() *rsa.PublicKey { return &s.priv.PublicKey }

func (s *Signer) Issue(subject uuid.UUID, email, role string) (*TokenPair, error) {
	if subject == uuid.Nil {
		return nil, fmt.Errorf("authn: subject rỗng")
	}

	now := time.Now()
	accessExp := now.Add(s.accessTTL)
	refreshExp := now.Add(s.refreshTTL)
	refreshID := uuidx.New().String()

	access, err := s.sign(Claims{
		Email: email,
		Role:  role,
		Type:  TokenAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject.String(),
			Issuer:    s.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(accessExp),
			ID:        uuidx.New().String(),
		},
	})
	if err != nil {
		return nil, err
	}

	refresh, err := s.sign(Claims{
		Type: TokenRefresh,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject.String(),
			Issuer:    s.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(refreshExp),
			ID:        refreshID,
		},
	})
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresAt:    accessExp.Unix(),
		RefreshID:    refreshID,
	}, nil
}

func (s *Signer) sign(claims Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = s.kid
	signed, err := token.SignedString(s.priv)
	if err != nil {
		return "", fmt.Errorf("authn: ký token: %w", err)
	}
	return signed, nil
}