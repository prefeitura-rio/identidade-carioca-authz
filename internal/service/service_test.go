package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/prefeitura-rio/identidade-carioca-authz/internal/config"
	jwtpkg "github.com/prefeitura-rio/identidade-carioca-authz/internal/jwt"
)

func setupJWKSServer(t *testing.T, key *rsa.PrivateKey) *httptest.Server {
	jwk := jwtpkg.JWK{
		Kid: "test-kid",
		Kty: "RSA",
		Alg: "RS256",
		Use: "sig",
		N:   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
		E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
	}
	jwks := jwtpkg.JWKS{Keys: []jwtpkg.JWK{jwk}}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jwks)
	}))

	return server
}

func generateSignedToken(t *testing.T, key *rsa.PrivateKey, sub, issuer string, roles []string) string {
	roleInterfaces := make([]interface{}, len(roles))
	for i, r := range roles {
		roleInterfaces[i] = r
	}

	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"sub":                sub,
		"iss":                issuer,
		"exp":                now.Add(1 * time.Hour).Unix(),
		"iat":                now.Unix(),
		"preferred_username": sub,
		"cpf":                sub,
		"realm_access": map[string]interface{}{
			"roles": roleInterfaces,
		},
	})
	token.Header["kid"] = "test-kid"

	tokenStr, err := token.SignedString(key)
	if err != nil {
		t.Fatalf("failed to sign test token: %v", err)
	}
	return tokenStr
}

func TestServiceAuthorizeAuthenticatedUser(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	jwksServer := setupJWKSServer(t, key)
	defer jwksServer.Close()

	issuer := "https://id-staging.rio.gov.br/realms/rio"

	cfg := &config.Config{
		MockMode:              true,
		JWKSEndpoint:          jwksServer.URL,
		JWTIssuer:             issuer,
		JWTTimeout:            2 * time.Second,
		CacheTTLSeconds:       30,
		CacheFailedTTLSeconds: 300,
		CircuitBreakerEnabled: true,
		DefaultPolicyScope:    "default.scope",
		DefaultResourceKind:   "generic",
		FailureMode:           "fail_open",
	}

	svc, err := NewService(cfg)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	token := generateSignedToken(t, key, "12345678901", issuer, []string{"servidor", "admin"})

	req := &AuthorizationRequest{
		AuthHeader: "Bearer " + token,
		Service:    "boletim",
		Scope:      "boletim",
		Path:       "/api/v1/boletim",
		Method:     "GET",
		Host:       "boletim.apps.rio.gov.br",
	}

	resp, err := svc.Authorize(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no authorization error, got %v", err)
	}
	if resp == nil {
		t.Fatal("expected response, got nil")
	}

	if !resp.Allowed {
		t.Errorf("expected request to be allowed, got status %s, reason: %s", resp.Status, resp.Reason)
	}
	if resp.PrincipalID != "12345678901" {
		t.Errorf("expected principal 12345678901, got %s", resp.PrincipalID)
	}
	if resp.Action != "api:read" {
		t.Errorf("expected action api:read, got %s", resp.Action)
	}
}

func TestServiceAuthorizeAnonymousUser(t *testing.T) {
	cfg := &config.Config{
		MockMode:              true,
		JWTIssuer:             "https://id-staging.rio.gov.br/realms/rio",
		CacheTTLSeconds:       30,
		CacheFailedTTLSeconds: 300,
		CircuitBreakerEnabled: true,
		FailureMode:           "fail_open",
	}

	svc, err := NewService(cfg)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	req := &AuthorizationRequest{
		AuthHeader: "",
		Service:    "boletim",
		Path:       "/api/v1/boletim",
		Method:     "POST",
		Host:       "boletim.apps.rio.gov.br",
	}

	resp, err := svc.Authorize(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no authorization error, got %v", err)
	}

	if resp.Allowed {
		t.Errorf("expected anonymous write request to be denied in mock mode")
	}
	if resp.PrincipalID != "anonymous" {
		t.Errorf("expected anonymous principal, got %s", resp.PrincipalID)
	}
}

func TestServiceHealthCheck(t *testing.T) {
	cfg := &config.Config{
		MockMode: true,
	}
	svc, err := NewService(cfg)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	health := svc.GetHealth()
	if health == nil {
		t.Fatal("expected health status, got nil")
	}
	if status, ok := health["status"].(string); !ok || status != "healthy" {
		t.Errorf("expected healthy status, got %v", health["status"])
	}
}
