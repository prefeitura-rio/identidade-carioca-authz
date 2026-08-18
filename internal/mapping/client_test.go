package mapping

import (
	"context"
	"testing"
	"time"
)

func TestMockMappingResolveAction(t *testing.T) {
	cfg := &Config{
		MockMode: true,
		Timeout:  1 * time.Second,
	}
	client := NewClient(cfg)

	tests := []struct {
		name     string
		service  string
		path     string
		method   string
		expected string
	}{
		{
			name:     "health endpoint",
			service:  "any",
			path:     "/health",
			method:   "GET",
			expected: "health:check",
		},
		{
			name:     "api read get",
			service:  "boletim",
			path:     "/api/v1/boletim",
			method:   "GET",
			expected: "api:read",
		},
		{
			name:     "api write post",
			service:  "boletim",
			path:     "/api/v1/boletim",
			method:   "POST",
			expected: "api:create",
		},
		{
			name:     "api write delete",
			service:  "rmi",
			path:     "/api/v1/processos/1",
			method:   "DELETE",
			expected: "api:delete",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action, _, err := client.GetAction(context.Background(), tt.path, tt.method)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if action != tt.expected {
				t.Errorf("GetAction(%s, %s) = %s, expected %s",
					tt.path, tt.method, action, tt.expected)
			}
		})
	}
}
