package incident

import (
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
	g.GET("/incidents", h.list)
	g.POST("/incidents", h.create, httpx.RequireRole(model.RoleAdmin, model.RoleOperator))
	g.GET("/incidents/:id", h.get)
	g.PATCH("/incidents/:id", h.update, httpx.RequireRole(model.RoleAdmin, model.RoleOperator))
	g.DELETE("/incidents/:id", h.delete, httpx.RequireRole(model.RoleAdmin))
}

type createRequest struct {
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Severity    model.Severity `json:"severity"`
	Source      string         `json:"source"`
}

type updateRequest struct {
	Title       *string               `json:"title"`
	Description *string               `json:"description"`
	Severity    *model.Severity       `json:"severity"`
	Status      *model.IncidentStatus `json:"status"`
}

func (h *Handler) create(c echo.Context) error {
	var req createRequest
	if err := httpx.Bind(c, &req); err != nil {
		return err
	}
	in := CreateInput{Title: req.Title, Description: req.Description, Severity: req.Severity, Source: req.Source}
	if claims, ok := httpx.ClaimsFrom(c); ok {
		id := claims.UserID
		in.ReporterID = &id
	}
	inc, err := h.svc.Create(c.Request().Context(), in)
	if err != nil {
		return mapError(err)
	}
	return httpx.Created(c, inc)
}

func (h *Handler) list(c echo.Context) error {
	f := ListFilter{
		Status:   model.IncidentStatus(c.QueryParam("status")),
		Severity: model.Severity(c.QueryParam("severity")),
	}
	if v := c.QueryParam("monitor_id"); v != "" {
		monitorID, err := uuid.Parse(v)
		if err != nil {
			return httpx.BadRequest("monitor_id must be a UUID")
		}
		f.MonitorID = &monitorID
	}
	f.Limit, _ = strconv.Atoi(c.QueryParam("limit"))
	f.Offset, _ = strconv.Atoi(c.QueryParam("offset"))
	f.normalise()

	items, total, err := h.svc.List(c.Request().Context(), f)
	if err != nil {
		return mapError(err)
	}
	if items == nil {
		items = []model.Incident{}
	}
	return httpx.List(c, items, httpx.Meta{Total: total, Limit: f.Limit, Offset: f.Offset})
}

func (h *Handler) get(c echo.Context) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	inc, err := h.svc.Get(c.Request().Context(), id)
	if err != nil {
		return mapError(err)
	}
	return httpx.OK(c, inc)
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
	inc, err := h.svc.Update(c.Request().Context(), id, UpdateInput(req))
	if err != nil {
		return mapError(err)
	}
	return httpx.OK(c, inc)
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
		return httpx.NotFound("incident not found")
	case errors.Is(err, ErrValidation):
		return httpx.BadRequest(strings.TrimPrefix(err.Error(), ErrValidation.Error()+": "))
	case errors.Is(err, ErrInvalidTransition):
		return httpx.Conflict(strings.TrimPrefix(err.Error(), "incident: "))
	default:
		return err
	}
}
