package monitor

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/Pzquzer/Nocu/shared/httpx"
	"github.com/Pzquzer/Nocu/shared/model"
)

// Handler exposes the service over HTTP.
type Handler struct{ svc *Service }

// NewHandler builds a Handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Register mounts the routes on an already-authenticated group.
func (h *Handler) Register(g *echo.Group) {
	write := httpx.RequireRole(model.RoleAdmin, model.RoleOperator)
	g.GET("/monitors", h.list)
	g.POST("/monitors", h.create, write)
	g.GET("/monitors/:id", h.get)
	g.PATCH("/monitors/:id", h.update, write)
	g.POST("/monitors/:id/pause", h.pause, write)
	g.POST("/monitors/:id/resume", h.resume, write)
	g.DELETE("/monitors/:id", h.delete, httpx.RequireRole(model.RoleAdmin))
	g.GET("/monitors/:id/heartbeats", h.heartbeats)
}

type notificationRequest struct {
	Type    model.NotificationType `json:"type"`
	Enabled *bool                  `json:"enabled"`
	Config  json.RawMessage        `json:"config"`
}

type createRequest struct {
	Type            model.MonitorType `json:"type"`
	Name            string            `json:"name"`
	URL             string            `json:"url"`
	IntervalSeconds *int              `json:"interval_seconds"`
	Retries         *int              `json:"retries"`
	TimeoutSeconds  *int              `json:"timeout_seconds"`

	CertExpiryNotification   *bool `json:"cert_expiry_notification"`
	DomainExpiryNotification *bool `json:"domain_expiry_notification"`
	MaxRedirects             *int  `json:"max_redirects"`
	SaveErrorResponse        *bool `json:"save_error_response"`

	Method       model.HTTPMethod   `json:"method"`
	BodyEncoding model.BodyEncoding `json:"body_encoding"`
	Body         string             `json:"body"`
	Headers      json.RawMessage    `json:"headers"`

	AuthMethod    model.AuthMethod `json:"auth_method"`
	BasicUsername string           `json:"basic_username"`
	BasicPassword string           `json:"basic_password"`
	BearerToken   string           `json:"bearer_token"`

	Notifications []notificationRequest `json:"notifications"`
}

type updateRequest struct {
	Type            *model.MonitorType `json:"type"`
	Name            *string            `json:"name"`
	URL             *string            `json:"url"`
	IntervalSeconds *int               `json:"interval_seconds"`
	Retries         *int               `json:"retries"`
	TimeoutSeconds  *int               `json:"timeout_seconds"`

	CertExpiryNotification   *bool `json:"cert_expiry_notification"`
	DomainExpiryNotification *bool `json:"domain_expiry_notification"`
	MaxRedirects             *int  `json:"max_redirects"`
	SaveErrorResponse        *bool `json:"save_error_response"`

	Method       *model.HTTPMethod   `json:"method"`
	BodyEncoding *model.BodyEncoding `json:"body_encoding"`
	Body         *string             `json:"body"`
	Headers      json.RawMessage     `json:"headers"`

	AuthMethod    *model.AuthMethod `json:"auth_method"`
	BasicUsername *string           `json:"basic_username"`
	BasicPassword *string           `json:"basic_password"`
	BearerToken   *string           `json:"bearer_token"`

	Notifications *[]notificationRequest `json:"notifications"`
}

func toNotificationInputs(in []notificationRequest) []NotificationInput {
	out := make([]NotificationInput, 0, len(in))
	for _, n := range in {
		out = append(out, NotificationInput{Type: n.Type, Enabled: n.Enabled, Config: n.Config})
	}
	return out
}

func (h *Handler) create(c echo.Context) error {
	var req createRequest
	if err := httpx.Bind(c, &req); err != nil {
		return err
	}
	in := CreateInput{
		Type:                     req.Type,
		Name:                     req.Name,
		URL:                      req.URL,
		IntervalSeconds:          req.IntervalSeconds,
		Retries:                  req.Retries,
		TimeoutSeconds:           req.TimeoutSeconds,
		CertExpiryNotification:   req.CertExpiryNotification,
		DomainExpiryNotification: req.DomainExpiryNotification,
		MaxRedirects:             req.MaxRedirects,
		SaveErrorResponse:        req.SaveErrorResponse,
		Method:                   req.Method,
		BodyEncoding:             req.BodyEncoding,
		Body:                     req.Body,
		Headers:                  req.Headers,
		AuthMethod:               req.AuthMethod,
		BasicUsername:            req.BasicUsername,
		BasicPassword:            req.BasicPassword,
		BearerToken:              req.BearerToken,
		Notifications:            toNotificationInputs(req.Notifications),
	}
	if claims, ok := httpx.ClaimsFrom(c); ok {
		id := claims.UserID
		in.OwnerID = &id
	}
	m, err := h.svc.Create(c.Request().Context(), in)
	if err != nil {
		return mapError(err)
	}
	return httpx.Created(c, m)
}

func (h *Handler) list(c echo.Context) error {
	f := ListFilter{Status: model.MonitorStatus(c.QueryParam("status"))}
	if v := c.QueryParam("active"); v != "" {
		active, err := strconv.ParseBool(v)
		if err != nil {
			return httpx.BadRequest("active must be true or false")
		}
		f.Active = &active
	}
	f.Limit, _ = strconv.Atoi(c.QueryParam("limit"))
	f.Offset, _ = strconv.Atoi(c.QueryParam("offset"))
	f.normalise()

	items, total, err := h.svc.List(c.Request().Context(), f)
	if err != nil {
		return mapError(err)
	}
	if items == nil {
		items = []model.Monitor{}
	}
	return httpx.List(c, items, httpx.Meta{Total: total, Limit: f.Limit, Offset: f.Offset})
}

func (h *Handler) get(c echo.Context) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	m, err := h.svc.Get(c.Request().Context(), id)
	if err != nil {
		return mapError(err)
	}
	return httpx.OK(c, m)
}

func (h *Handler) update(c echo.Context) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	var req updateRequest
	if err := httpx.Bind(c, &req); err != nil {
		return err
	}
	in := UpdateInput{
		Type:                     req.Type,
		Name:                     req.Name,
		URL:                      req.URL,
		IntervalSeconds:          req.IntervalSeconds,
		Retries:                  req.Retries,
		TimeoutSeconds:           req.TimeoutSeconds,
		CertExpiryNotification:   req.CertExpiryNotification,
		DomainExpiryNotification: req.DomainExpiryNotification,
		MaxRedirects:             req.MaxRedirects,
		SaveErrorResponse:        req.SaveErrorResponse,
		Method:                   req.Method,
		BodyEncoding:             req.BodyEncoding,
		Body:                     req.Body,
		Headers:                  req.Headers,
		AuthMethod:               req.AuthMethod,
		BasicUsername:            req.BasicUsername,
		BasicPassword:            req.BasicPassword,
		BearerToken:              req.BearerToken,
	}
	if req.Notifications != nil {
		n := toNotificationInputs(*req.Notifications)
		in.Notifications = &n
	}
	m, err := h.svc.Update(c.Request().Context(), id, in)
	if err != nil {
		return mapError(err)
	}
	return httpx.OK(c, m)
}

func (h *Handler) pause(c echo.Context) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	m, err := h.svc.Pause(c.Request().Context(), id)
	if err != nil {
		return mapError(err)
	}
	return httpx.OK(c, m)
}

func (h *Handler) resume(c echo.Context) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	m, err := h.svc.Resume(c.Request().Context(), id)
	if err != nil {
		return mapError(err)
	}
	return httpx.OK(c, m)
}

func (h *Handler) delete(c echo.Context) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	if err := h.svc.Delete(c.Request().Context(), id); err != nil {
		return mapError(err)
	}
	return httpx.NoContent(c)
}

func (h *Handler) heartbeats(c echo.Context) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	f := HeartbeatFilter{}
	f.Important, _ = strconv.ParseBool(c.QueryParam("important"))
	f.Limit, _ = strconv.Atoi(c.QueryParam("limit"))
	f.Offset, _ = strconv.Atoi(c.QueryParam("offset"))
	f.normalise()

	items, total, err := h.svc.ListHeartbeats(c.Request().Context(), id, f)
	if err != nil {
		return mapError(err)
	}
	if items == nil {
		items = []model.MonitorHeartbeat{}
	}
	return httpx.List(c, items, httpx.Meta{Total: total, Limit: f.Limit, Offset: f.Offset})
}

func parseID(c echo.Context) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return uuid.Nil, httpx.BadRequest("id must be a UUID")
	}
	return id, nil
}

func mapError(err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return httpx.NotFound("monitor not found")
	case errors.Is(err, ErrValidation):
		return httpx.BadRequest(strings.TrimPrefix(err.Error(), ErrValidation.Error()+": "))
	default:
		return err
	}
}
