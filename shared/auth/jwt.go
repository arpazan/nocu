// Package auth issues and verifies the JWTs shared between identity and the other services.
package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/Pzquzer/Nocu/shared/config"
)

// TokenType distinguishes access tokens from refresh tokens.
type TokenType string

const (
	TokenAccess  TokenType = "access"
	TokenRefresh TokenType = "refresh"
)

var (
	ErrInvalidToken   = errors.New("auth: invalid token")
	ErrWrongTokenType = errors.New("auth: wrong token type")
)

// Claims is the payload embedded in every token.
type Claims struct {
	UserID uuid.UUID `json:"uid"`
	Email  string    `json:"email"`
	Roles  []string  `json:"roles"`
	Type   TokenType `json:"typ"`
	jwt.RegisteredClaims
}

// HasRole reports whether the subject carries the given role.
func (c *Claims) HasRole(role string) bool {
	for _, r := range c.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// TokenPair is what a login or refresh returns.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

// Manager signs and parses tokens with an HMAC secret.
type Manager struct {
	secret     []byte
	issuer     string
	accessTTL  time.Duration
	refreshTTL time.Duration
	now        func() time.Time
}

// NewManager builds a Manager from config.
func NewManager(cfg config.JWT) *Manager {
	return &Manager{
		secret:     []byte(cfg.Secret),
		issuer:     cfg.Issuer,
		accessTTL:  cfg.AccessTTL,
		refreshTTL: cfg.RefreshTTL,
		now:        time.Now,
	}
}

// Issue creates an access and refresh token pair for a subject.
func (m *Manager) Issue(userID uuid.UUID, email string, roles []string) (TokenPair, error) {
	access, err := m.sign(userID, email, roles, TokenAccess, m.accessTTL)
	if err != nil {
		return TokenPair{}, err
	}
	refresh, err := m.sign(userID, email, roles, TokenRefresh, m.refreshTTL)
	if err != nil {
		return TokenPair{}, err
	}
	return TokenPair{
		AccessToken:  access,
		RefreshToken: refresh,
		TokenType:    "Bearer",
		ExpiresIn:    int64(m.accessTTL.Seconds()),
	}, nil
}

func (m *Manager) sign(userID uuid.UUID, email string, roles []string, typ TokenType, ttl time.Duration) (string, error) {
	now := m.now()
	if roles == nil {
		roles = []string{}
	}
	claims := Claims{
		UserID: userID,
		Email:  email,
		Roles:  roles,
		Type:   typ,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			ID:        uuid.NewString(),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("auth: sign: %w", err)
	}
	return signed, nil
}

// Parse validates a token and checks it is of the expected type.
func (m *Manager) Parse(raw string, expected TokenType) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method %q", t.Header["alg"])
		}
		return m.secret, nil
	},
		jwt.WithIssuer(m.issuer),
		jwt.WithTimeFunc(m.now),
		jwt.WithExpirationRequired(),
	)
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	if claims.Type != expected {
		return nil, ErrWrongTokenType
	}
	return claims, nil
}
