package main

import (
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
	"github.com/prefeitura-rio/identidade-carioca-authz/internal/service"
)

func setupTestJWKS(t *testing.T, key *rsa.PrivateKey) *httptest.Server {
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

func generateValidToken(t *testing.T, key *rsa.PrivateKey, sub, issuer string, roles []string) string {
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
		t.Fatalf("failed to sign token: %v", err)
	}
	return tokenStr
}

func TestHTTPServerEndToEnd(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	jwksServer := setupTestJWKS(t, key)
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

	svc, err := service.NewService(cfg)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	server := &ExtAuthzServer{
		service: svc,
	}

	t.Run("HealthCheck", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		rec := httptest.NewRecorder()

		server.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 OK, got %d", rec.Code)
		}
	})

	t.Run("OptionsPreflightAllowed", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/api/v1/boletim", nil)
		req.Host = "boletim.apps.rio.gov.br"
		rec := httptest.NewRecorder()

		server.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 OK for OPTIONS preflight, got %d", rec.Code)
		}
		if rec.Header().Get(resultHeader) != resultAllowed {
			t.Errorf("expected result allowed, got %s", rec.Header().Get(resultHeader))
		}
	})

	t.Run("AuthenticatedRequestAllowed", func(t *testing.T) {
		token := generateValidToken(t, key, "12345678901", issuer, []string{"servidor", "admin"})

		req := httptest.NewRequest("GET", "/api/v1/boletim", nil)
		req.Host = "boletim.apps.rio.gov.br"
		req.Header.Set("X-Envoy-Original-Path", "/api/v1/boletim")
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("x-policy-scope", "boletim")
		rec := httptest.NewRecorder()

		server.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 OK, got %d. Body: %s", rec.Code, rec.Body.String())
		}
		if rec.Header().Get(resultHeader) != resultAllowed {
			t.Errorf("expected allowed header, got %s", rec.Header().Get(resultHeader))
		}
		if rec.Header().Get("X-Cerbos-Action") != "api:read" {
			t.Errorf("expected X-Cerbos-Action api:read, got %s", rec.Header().Get("X-Cerbos-Action"))
		}
	})

	t.Run("AnonymousWriteDenied", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/boletim", nil)
		req.Host = "boletim.apps.rio.gov.br"
		req.Header.Set("X-Envoy-Original-Path", "/api/v1/boletim")
		req.Header.Set("x-policy-scope", "boletim")
		rec := httptest.NewRecorder()

		server.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403 Forbidden, got %d", rec.Code)
		}
		if rec.Header().Get(resultHeader) != resultDenied {
			t.Errorf("expected denied header, got %s", rec.Header().Get(resultHeader))
		}
	})
}
