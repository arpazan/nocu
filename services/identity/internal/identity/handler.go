package identity

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/Pzquzer/Nocu/shared/httpx"
	"github.com/Pzquzer/Nocu/shared/model"
)

// Handler exposes the service over HTTP.
type Handler struct {
	svc *Service
}

// NewHandler builds a Handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Register mounts the routes. auth is the middleware that protects /me.
func (h *Handler) Register(g *echo.Group, auth echo.MiddlewareFunc) {
	g.POST("/auth/register", h.register)
	g.POST("/auth/login", h.login)
	g.POST("/auth/refresh", h.refresh)
	g.GET("/me", h.me, auth)
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// UserResponse is the public shape of a user.
type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	IsActive  bool      `json:"is_active"`
	Roles     []string  `json:"roles"`
	CreatedAt time.Time `json:"created_at"`
}

func toUserResponse(u *model.User) UserResponse {
	return UserResponse{
		ID:        u.ID,
		Email:     u.Email,
		Name:      u.Name,
		IsActive:  u.IsActive,
		Roles:     u.RoleNames(),
		CreatedAt: u.CreatedAt,
	}
}

func (h *Handler) register(c echo.Context) error {
	var req registerRequest
	if err := httpx.Bind(c, &req); err != nil {
		return err
	}
	user, err := h.svc.Register(c.Request().Context(), RegisterInput(req))
	if err != nil {
		return mapError(err)
	}
	return httpx.Created(c, toUserResponse(user))
}

func (h *Handler) login(c echo.Context) error {
	var req loginRequest
	if err := httpx.Bind(c, &req); err != nil {
		return err
	}
	pair, err := h.svc.Login(c.Request().Context(), req.Email, req.Password)
	if err != nil {
		return mapError(err)
	}
	return httpx.OK(c, pair)
}

func (h *Handler) refresh(c echo.Context) error {
	var req refreshRequest
	if err := httpx.Bind(c, &req); err != nil {
		return err
	}
	if req.RefreshToken == "" {
		return httpx.BadRequest("refresh_token is required")
	}
	pair, err := h.svc.Refresh(c.Request().Context(), req.RefreshToken)
	if err != nil {
		return mapError(err)
	}
	return httpx.OK(c, pair)
}

func (h *Handler) me(c echo.Context) error {
	claims, ok := httpx.ClaimsFrom(c)
	if !ok {
		return httpx.Unauthorized("missing credentials")
	}
	user, err := h.svc.Me(c.Request().Context(), claims.UserID)
	if err != nil {
		return mapError(err)
	}
	return httpx.OK(c, toUserResponse(user))
}

func mapError(err error) error {
	switch {
	case errors.Is(err, ErrValidation):
		return httpx.BadRequest(trimPrefix(err))
	case errors.Is(err, ErrEmailTaken):
		return httpx.Conflict("email already registered")
	case errors.Is(err, ErrInvalidCredentials):
		return httpx.Unauthorized("invalid credentials")
	case errors.Is(err, ErrInactive):
		return httpx.Forbidden("account is inactive")
	case errors.Is(err, ErrNotFound):
		return httpx.NotFound("user not found")
	default:
		return err
	}
}

// trimPrefix strips the "identity: validation failed: " prefix for client messages.
func trimPrefix(err error) string {
	return strings.TrimPrefix(err.Error(), ErrValidation.Error()+": ")
}
