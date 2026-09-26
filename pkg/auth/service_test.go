package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	return &Service{path: filepath.Join(t.TempDir(), "users.json")}
}

func TestBootstrapAdminIsOneTime(t *testing.T) {
	s := newTestService(t)
	if s.HasAdmin() {
		t.Fatal("fresh store must not contain a default administrator")
	}
	if err := s.BootstrapAdmin("admin_user", "strong-password"); err != nil {
		t.Fatalf("bootstrap failed: %v", err)
	}
	if err := s.BootstrapAdmin("other_admin", "another-password"); err == nil {
		t.Fatal("second bootstrap must be rejected")
	}
	u, err := s.Login("admin_user", "strong-password")
	if err != nil || u.Role != "admin" {
		t.Fatalf("bootstrapped administrator cannot log in: %v", err)
	}
}

func TestConcurrentBootstrapCreatesOneAdmin(t *testing.T) {
	s := newTestService(t)
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, name := range []string{"admin_one", "admin_two"} {
		wg.Add(1)
		go func(username string) {
			defer wg.Done()
			results <- s.BootstrapAdmin(username, "strong-password")
		}(name)
	}
	wg.Wait()
	close(results)
	successes := 0
	for err := range results {
		if err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("expected one successful bootstrap, got %d", successes)
	}
}

func TestPublicUserDoesNotSerializePasswordHash(t *testing.T) {
	public := ToPublicUser(User{ID: "1", Username: "user", PasswordHash: "secret", Role: "user"})
	encoded, err := json.Marshal(public)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "secret") || strings.Contains(string(encoded), "passwordHash") {
		t.Fatalf("public user leaked password hash: %s", encoded)
	}
}

func TestCredentialLimits(t *testing.T) {
	if err := validCredential("valid.user", "12345678"); err != nil {
		t.Fatalf("valid credentials rejected: %v", err)
	}
	if err := validCredential("bad@example", "12345678"); err == nil {
		t.Fatal("username with email delimiter must be rejected")
	}
	if err := validCredential("valid", strings.Repeat("a", 73)); err == nil {
		t.Fatal("password over bcrypt limit must be rejected")
	}
}

func TestRemoteLoginRollbackAndSafeError(t *testing.T) {
	const userID = "12345678-1234-1234-1234-123456789abc"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/v1/token":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "new-token",
				"user":         map[string]string{"id": userID},
			})
		case "/rest/v1/profiles":
			http.Error(w, "database password leaked", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := newRemoteClient(server.URL, "anon")
	oldUser := User{ID: userID, Username: "existing", Enabled: true}
	client.token = "old-token"
	client.user = &oldUser
	_, err := client.login("new-user", "12345678")
	if err == nil {
		t.Fatal("login unexpectedly succeeded")
	}
	if strings.Contains(err.Error(), "database") || strings.Contains(err.Error(), "password") {
		t.Fatalf("remote error leaked backend details: %v", err)
	}
	if got := client.current(); got == nil || got.Username != oldUser.Username {
		t.Fatalf("failed login replaced current user: %#v", got)
	}
	if client.token != "old-token" {
		t.Fatalf("failed login replaced token: %q", client.token)
	}
}

func TestRemoteRejectsInvalidUserID(t *testing.T) {
	client := newRemoteClient("http://127.0.0.1:1", "anon")
	client.user = &User{Role: "admin"}
	client.token = "token"
	if err := client.setEnabled("id&enabled=eq.true", false); err == nil {
		t.Fatal("invalid profile ID was accepted")
	}
	if err := client.deleteUser("../other"); err == nil {
		t.Fatal("invalid delete ID was accepted")
	}
}

func TestRemoteRejectsOversizedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", maxRemoteResponseBytes+1)))
	}))
	defer server.Close()
	client := newRemoteClient(server.URL, "anon")
	if err := client.request("GET", "", nil, false, nil); err == nil || !strings.Contains(err.Error(), "过大") {
		t.Fatalf("oversized response was not rejected: %v", err)
	}
}
