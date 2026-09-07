package authn_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/logistic/pkg/authn"
	"github.com/logistic/pkg/uuidx"
)

func newPair(t *testing.T) (*authn.Signer, *authn.Verifier, *rsa.PrivateKey) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("sinh khoa: %v", err)
	}
	signer, err := authn.NewSigner(priv)
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}
	verifier, err := authn.NewVerifier(&priv.PublicKey)
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}
	return signer, verifier, priv
}

func TestIssueAndVerify(t *testing.T) {
	signer, verifier, _ := newPair(t)
	sub := uuidx.New()

	pair, err := signer.Issue(sub, "tai@xe.vn", authn.RoleDriver)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	claims, err := verifier.VerifyAccess(pair.AccessToken)
	if err != nil {
		t.Fatalf("VerifyAccess: %v", err)
	}
	if claims.Subject != sub.String() {
		t.Fatalf("subject lech: %s != %s", claims.Subject, sub)
	}
	if claims.Role != authn.RoleDriver {
		t.Fatalf("role lech: %q", claims.Role)
	}
	if claims.Email != "tai@xe.vn" {
		t.Fatalf("email lech: %q", claims.Email)
	}
}

func TestRefreshTokenRejectedAsAccess(t *testing.T) {
	signer, verifier, _ := newPair(t)
	pair, _ := signer.Issue(uuidx.New(), "a@b.vn", authn.RoleAdmin)

	if _, err := verifier.VerifyAccess(pair.RefreshToken); err != authn.ErrWrongTokenType {
		t.Fatalf("mong doi ErrWrongTokenType, duoc %v", err)
	}
	if _, err := verifier.VerifyRefresh(pair.AccessToken); err != authn.ErrWrongTokenType {
		t.Fatalf("mong doi ErrWrongTokenType, duoc %v", err)
	}
}

func TestRefreshCarriesNoRole(t *testing.T) {
	signer, verifier, _ := newPair(t)
	pair, _ := signer.Issue(uuidx.New(), "a@b.vn", authn.RoleAdmin)

	claims, err := verifier.VerifyRefresh(pair.RefreshToken)
	if err != nil {
		t.Fatalf("VerifyRefresh: %v", err)
	}
	if claims.Role != "" {
		t.Fatalf("refresh token khong duoc mang role, dang mang %q", claims.Role)
	}
}

func TestRejectsAlgConfusion(t *testing.T) {
	_, verifier, priv := newPair(t)

	pubDER, _ := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})

	forged := jwt.NewWithClaims(jwt.SigningMethodHS256, authn.Claims{
		Role: authn.RoleAdmin,
		Type: authn.TokenAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   uuidx.New().String(),
			Issuer:    authn.Issuer,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})
	signed, err := forged.SignedString(pubPEM)
	if err != nil {
		t.Fatalf("ky token gia: %v", err)
	}

	if _, err := verifier.VerifyAccess(signed); err == nil {
		t.Fatal("LO HONG: token HS256 ky bang public key lai duoc chap nhan")
	}
}

func TestRejectsAlgNone(t *testing.T) {
	_, verifier, _ := newPair(t)

	tok := jwt.NewWithClaims(jwt.SigningMethodNone, authn.Claims{
		Role: authn.RoleAdmin,
		Type: authn.TokenAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   uuidx.New().String(),
			Issuer:    authn.Issuer,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})
	signed, _ := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)

	if _, err := verifier.VerifyAccess(signed); err == nil {
		t.Fatal("LO HONG: token alg=none duoc chap nhan")
	}
}

func TestRejectsForeignKey(t *testing.T) {
	_, verifier, _ := newPair(t)
	otherSigner, _, _ := newPair(t)

	pair, _ := otherSigner.Issue(uuidx.New(), "a@b.vn", authn.RoleAdmin)
	if _, err := verifier.VerifyAccess(pair.AccessToken); err == nil {
		t.Fatal("LO HONG: token ky bang khoa la duoc chap nhan")
	}
}

func TestRejectsExpired(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	signer, _ := authn.NewSigner(priv, authn.WithAccessTTL(-time.Hour))
	verifier, _ := authn.NewVerifier(&priv.PublicKey)

	pair, _ := signer.Issue(uuidx.New(), "a@b.vn", authn.RoleDriver)
	if _, err := verifier.VerifyAccess(pair.AccessToken); err != authn.ErrTokenExpired {
		t.Fatalf("mong doi ErrTokenExpired, duoc %v", err)
	}
}

func TestRejectsWrongIssuer(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	signer, _ := authn.NewSigner(priv, authn.WithIssuer("ke-mao-danh"))
	verifier, _ := authn.NewVerifier(&priv.PublicKey)

	pair, _ := signer.Issue(uuidx.New(), "a@b.vn", authn.RoleDriver)
	if _, err := verifier.VerifyAccess(pair.AccessToken); err != authn.ErrWrongIssuer {
		t.Fatalf("mong doi ErrWrongIssuer, duoc %v", err)
	}
}

func TestRejectsSmallKey(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 1024)
	if _, err := authn.NewSigner(priv); err == nil {
		t.Fatal("khoa 1024 bit phai bi tu choi")
	}
}

func TestKeyRotation(t *testing.T) {
	oldPriv, _ := rsa.GenerateKey(rand.Reader, 2048)
	newPriv, _ := rsa.GenerateKey(rand.Reader, 2048)

	oldSigner, _ := authn.NewSigner(oldPriv)
	newSigner, _ := authn.NewSigner(newPriv)

	verifier, _ := authn.NewVerifier(&newPriv.PublicKey)
	if err := verifier.AddKey(&oldPriv.PublicKey); err != nil {
		t.Fatalf("AddKey: %v", err)
	}

	for name, s := range map[string]*authn.Signer{"khoa cu": oldSigner, "khoa moi": newSigner} {
		pair, _ := s.Issue(uuidx.New(), "a@b.vn", authn.RoleDriver)
		if _, err := verifier.VerifyAccess(pair.AccessToken); err != nil {
			t.Fatalf("%s phai verify duoc trong luc xoay: %v", name, err)
		}
	}
}

func TestLoadKeysFromPEM(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)

	privDER, _ := x509.MarshalPKCS8PrivateKey(priv)
	privPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privDER}))
	pubDER, _ := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	pubPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER}))

	loadedPriv, err := authn.LoadPrivateKey(privPEM)
	if err != nil {
		t.Fatalf("LoadPrivateKey tu PEM: %v", err)
	}
	if !loadedPriv.Equal(priv) {
		t.Fatal("private key doc lai khong khop")
	}

	loadedPub, err := authn.LoadPublicKey(pubPEM)
	if err != nil {
		t.Fatalf("LoadPublicKey tu PEM: %v", err)
	}
	if !loadedPub.Equal(&priv.PublicKey) {
		t.Fatal("public key doc lai khong khop")
	}
}

func TestSubjectMustBeUUID(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	verifier, _ := authn.NewVerifier(&priv.PublicKey)

	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, authn.Claims{
		Type: authn.TokenAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "12345",
			Issuer:    authn.Issuer,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})
	signed, _ := tok.SignedString(priv)

	if _, err := verifier.VerifyAccess(signed); err == nil {
		t.Fatal("subject dang int64 kieu cu phai bi tu choi")
	}
}

func TestIssueRejectsNilSubject(t *testing.T) {
	signer, _, _ := newPair(t)
	if _, err := signer.Issue(uuid.Nil, "a@b.vn", authn.RoleDriver); err == nil {
		t.Fatal("subject rong phai bi tu choi")
	}
}

func BenchmarkVerifyAccess(b *testing.B) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	signer, _ := authn.NewSigner(priv)
	verifier, _ := authn.NewVerifier(&priv.PublicKey)
	pair, _ := signer.Issue(uuidx.New(), "a@b.vn", authn.RoleDriver)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := verifier.VerifyAccess(pair.AccessToken); err != nil {
			b.Fatal(err)
		}
	}
}