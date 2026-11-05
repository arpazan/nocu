package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Pzquzer/Nocu/shared/config"
)

func testManager() *Manager {
	return NewManager(config.JWT{
		Secret:     "test-secret-that-is-long-enough",
		Issuer:     "test",
		AccessTTL:  time.Minute,
		RefreshTTL: time.Hour,
	})
}

func TestIssueAndParseRoundTrip(t *testing.T) {
	m := testManager()
	id := uuid.New()

	pair, err := m.Issue(id, "a@b.c", []string{"admin"})
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	claims, err := m.Parse(pair.AccessToken, TokenAccess)
	if err != nil {
		t.Fatalf("parse access: %v", err)
	}
	if claims.UserID != id || claims.Email != "a@b.c" || !claims.HasRole("admin") {
		t.Fatalf("unexpected claims: %+v", claims)
	}

	if _, err := m.Parse(pair.RefreshToken, TokenRefresh); err != nil {
		t.Fatalf("parse refresh: %v", err)
	}
}

func TestParseRejectsWrongType(t *testing.T) {
	m := testManager()
	pair, _ := m.Issue(uuid.New(), "a@b.c", nil)

	if _, err := m.Parse(pair.RefreshToken, TokenAccess); !errors.Is(err, ErrWrongTokenType) {
		t.Fatalf("expected ErrWrongTokenType, got %v", err)
	}
}

func TestParseRejectsExpired(t *testing.T) {
	m := testManager()
	pair, _ := m.Issue(uuid.New(), "a@b.c", nil)

	m.now = func() time.Time { return time.Now().Add(2 * time.Minute) }
	if _, err := m.Parse(pair.AccessToken, TokenAccess); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestParseRejectsOtherSecret(t *testing.T) {
	m := testManager()
	pair, _ := m.Issue(uuid.New(), "a@b.c", nil)

	other := NewManager(config.JWT{Secret: "different-secret", Issuer: "test", AccessTTL: time.Minute, RefreshTTL: time.Hour})
	if _, err := other.Parse(pair.AccessToken, TokenAccess); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}
