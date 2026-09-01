package jwt

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func generateTestRSAKey(t *testing.T) *rsa.PrivateKey {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}
	return key
}

func createTestToken(t *testing.T, key *rsa.PrivateKey, claims jwt.MapClaims) string {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "test-key-id"
	tokenString, err := token.SignedString(key)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}
	return tokenString
}

func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name        string
		header      string
		expectedTok string
		expectErr   bool
	}{
		{
			name:        "valid bearer prefix",
			header:      "Bearer my-token-123",
			expectedTok: "my-token-123",
			expectErr:   false,
		},
		{
			name:        "bearer lowercase",
			header:      "bearer my-token-123",
			expectedTok: "my-token-123",
			expectErr:   false,
		},
		{
			name:        "empty header",
			header:      "",
			expectedTok: "",
			expectErr:   true,
		},
		{
			name:        "invalid header format",
			header:      "InvalidHeaderWithoutSpace",
			expectedTok: "",
			expectErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := ExtractBearerToken(tt.header)
			if tt.expectErr && err == nil {
				t.Errorf("ExtractBearerToken(%q) expected error, got nil", tt.header)
			}
			if !tt.expectErr && err != nil {
				t.Errorf("ExtractBearerToken(%q) unexpected error: %v", tt.header, err)
			}
			if token != tt.expectedTok {
				t.Errorf("ExtractBearerToken(%q) = %q, expected %q", tt.header, token, tt.expectedTok)
			}
		})
	}
}

func TestClaimsHelperMethods(t *testing.T) {
	claims := &Claims{
		Sub:               "sub-123",
		PreferredUsername: "12345678901",
		Roles:             []string{"servidor"},
		RealmAccess: &RealmAccess{
			Roles: []string{"realm-admin"},
		},
		Exp: time.Now().Add(1 * time.Hour).Unix(),
	}

	if claims.GetUserID() != "12345678901" {
		t.Errorf("expected preferred_username 12345678901, got %s", claims.GetUserID())
	}
	if !claims.IsAuthenticated() {
		t.Errorf("expected authenticated user")
	}
	if claims.IsExpired() {
		t.Errorf("expected token not to be expired")
	}
	roles := claims.GetRoles()
	if len(roles) == 0 || roles[0] != "servidor" {
		t.Errorf("expected roles [servidor], got %v", roles)
	}

	anonClaims := &Claims{
		Sub: "anonymous",
	}
	if anonClaims.IsAuthenticated() {
		t.Errorf("expected anonymous not to be authenticated")
	}

	pastClaims := &Claims{
		Exp: time.Now().Add(-1 * time.Hour).Unix(),
	}
	if !pastClaims.IsExpired() {
		t.Errorf("expected past token to be expired")
	}
}

func TestClaimsGetRoles(t *testing.T) {
	tests := []struct {
		name   string
		claims Claims
		want   []string
	}{
		{
			name:   "groups only",
			claims: Claims{Groups: []string{"group-a", "group-b"}},
			want:   []string{"group-a", "group-b"},
		},
		{
			name: "realm roles and groups",
			claims: Claims{
				RealmAccess: &RealmAccess{Roles: []string{"realm-a"}},
				Groups:      []string{"group-a", "group-b"},
			},
			want: []string{"realm-a", "group-a", "group-b"},
		},
		{
			name: "direct roles take precedence and combine",
			claims: Claims{
				Roles:       []string{"direct-a", "direct-b"},
				RealmAccess: &RealmAccess{Roles: []string{"realm-a"}},
				Groups:      []string{"group-a"},
			},
			want: []string{"direct-a", "direct-b", "realm-a", "group-a"},
		},
		{
			name:   "empty claims",
			claims: Claims{},
			want:   []string{},
		},
		{
			name: "duplicate roles are removed",
			claims: Claims{
				Roles:       []string{"direct-a", "shared"},
				RealmAccess: &RealmAccess{Roles: []string{"shared", "realm-a"}},
				Groups:      []string{"group-a", "shared", "group-a"},
			},
			want: []string{"direct-a", "shared", "realm-a", "group-a"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.claims.GetRoles()
			if len(got) != len(tt.want) {
				t.Fatalf("GetRoles() = %v, want %v", got, tt.want)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("GetRoles()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
