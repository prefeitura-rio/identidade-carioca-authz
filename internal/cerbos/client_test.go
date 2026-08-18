package cerbos

import (
	"context"
	"testing"
	"time"
)

func TestMockClientScopeAndAllowed(t *testing.T) {
	cfg := &Config{
		Timeout:  2 * time.Second,
		MockMode: true,
	}
	client := NewClient(cfg)

	req := &CheckResourcesRequest{
		RequestID: "req-123",
		Principal: Principal{
			ID:    "12345678901",
			Roles: []string{"admin"},
		},
		Resources: []Resource{
			{
				Resource: ResourceInfo{
					Kind:  "api:endpoint",
					ID:    "resource-1",
					Scope: "boletim",
					Attr: map[string]interface{}{
						"service": "boletim",
						"path":    "/api/v1/boletim",
						"method":  "GET",
					},
				},
				Actions: []string{"boletim:read"},
			},
		},
	}

	resp, err := client.CheckResources(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp == nil {
		t.Fatal("expected response, got nil")
	}

	if !resp.IsAllowed("boletim:read") {
		t.Errorf("expected boletim:read to be allowed for authenticated user")
	}
	if resp.GetDecision("boletim:read") != "EFFECT_ALLOW" {
		t.Errorf("expected EFFECT_ALLOW, got %s", resp.GetDecision("boletim:read"))
	}
	if resp.IsAllowed("nonexistent:action") {
		t.Errorf("expected nonexistent action to be denied")
	}
}

func TestMockClientAnonymousDenied(t *testing.T) {
	cfg := &Config{
		Timeout:  2 * time.Second,
		MockMode: true,
	}
	client := NewClient(cfg)

	req := &CheckResourcesRequest{
		RequestID: "req-anon",
		Principal: Principal{
			ID:    "anonymous",
			Roles: []string{"anonymous"},
		},
		Resources: []Resource{
			{
				Resource: ResourceInfo{
					Kind:  "generic",
					ID:    "resource",
					Scope: "superapp",
				},
				Actions: []string{"api:write"},
			},
		},
	}

	resp, err := client.CheckResources(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.IsAllowed("api:write") {
		t.Errorf("expected anonymous user to be denied for api:write")
	}
}
