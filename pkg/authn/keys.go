package authn

import (
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"strings"
)

func LoadPrivateKey(source string) (*rsa.PrivateKey, error) {
	block, err := readPEM(source)
	if err != nil {
		return nil, fmt.Errorf("authn: đọc private key: %w", err)
	}

	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("authn: private key không phải RSA")
		}
		return rsaKey, nil
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("authn: private key không đọc được (thử cả PKCS#8 và PKCS#1): %w", err)
	}
	return key, nil
}

func LoadPublicKey(source string) (*rsa.PublicKey, error) {
	block, err := readPEM(source)
	if err != nil {
		return nil, fmt.Errorf("authn: đọc public key: %w", err)
	}

	if key, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		rsaKey, ok := key.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("authn: public key không phải RSA")
		}
		return rsaKey, nil
	}
	key, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("authn: public key không đọc được: %w", err)
	}
	return key, nil
}

func readPEM(source string) (*pem.Block, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return nil, ErrNoKey
	}

	var raw []byte
	switch {
	case strings.HasPrefix(source, "-----BEGIN"):
		raw = []byte(source)
	case strings.Contains(source, "-----BEGIN"):

		raw = []byte(strings.ReplaceAll(source, `\n`, "\n"))
	default:
		if decoded, err := base64.StdEncoding.DecodeString(source); err == nil &&
			strings.Contains(string(decoded), "-----BEGIN") {
			raw = decoded
			break
		}
		content, err := os.ReadFile(source)
		if err != nil {
			return nil, fmt.Errorf("không phải PEM, cũng không đọc được như đường dẫn file: %w", err)
		}
		raw = content
	}

	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, errors.New("nội dung không phải khối PEM hợp lệ")
	}
	return block, nil
}

func KeyID(pub *rsa.PublicKey) (string, error) {
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(der)
	return base64.RawURLEncoding.EncodeToString(sum[:8]), nil
}