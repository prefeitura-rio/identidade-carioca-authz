package config

import (
	"os"
	"testing"
	"time"
)

func TestConfigLoadDefaults(t *testing.T) {
	_ = os.Setenv("JWKS_ENDPOINT", "https://id-staging.rio.gov.br/realms/rio/protocol/openid-connect/certs")
	_ = os.Setenv("JWT_ISSUER", "https://id-staging.rio.gov.br/realms/rio")
	_ = os.Unsetenv("DEFAULT_POLICY_SCOPE")
	_ = os.Unsetenv("DEFAULT_RESOURCE_KIND")
	defer func() {
		_ = os.Unsetenv("JWKS_ENDPOINT")
		_ = os.Unsetenv("JWT_ISSUER")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.DefaultPolicyScope != "" {
		t.Errorf("expected empty default policy scope, got %s", cfg.DefaultPolicyScope)
	}
	if cfg.DefaultResourceKind != "generic" {
		t.Errorf("expected generic default resource kind, got %s", cfg.DefaultResourceKind)
	}
	if cfg.OTelServiceName != "identidade-carioca-authz" {
		t.Errorf("expected identidade-carioca-authz service name, got %s", cfg.OTelServiceName)
	}
}

func TestConfigLoadCustomMultiTenant(t *testing.T) {
	_ = os.Setenv("JWKS_ENDPOINT", "https://id-staging.rio.gov.br/realms/rio/protocol/openid-connect/certs")
	_ = os.Setenv("JWT_ISSUER", "https://id-staging.rio.gov.br/realms/rio")
	_ = os.Setenv("DEFAULT_POLICY_SCOPE", "superapp.services")
	_ = os.Setenv("DEFAULT_RESOURCE_KIND", "api:endpoint")
	defer func() {
		_ = os.Unsetenv("JWKS_ENDPOINT")
		_ = os.Unsetenv("JWT_ISSUER")
		_ = os.Unsetenv("DEFAULT_POLICY_SCOPE")
		_ = os.Unsetenv("DEFAULT_RESOURCE_KIND")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.DefaultPolicyScope != "superapp.services" {
		t.Errorf("expected superapp.services, got %s", cfg.DefaultPolicyScope)
	}
	if cfg.DefaultResourceKind != "api:endpoint" {
		t.Errorf("expected api:endpoint, got %s", cfg.DefaultResourceKind)
	}
	if cfg.CerbosTimeout != 2*time.Second {
		t.Errorf("expected 2s timeout, got %v", cfg.CerbosTimeout)
	}
}
