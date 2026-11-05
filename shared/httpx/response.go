package httpx

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// Envelope wraps every successful JSON response.
type Envelope struct {
	Data any   `json:"data"`
	Meta *Meta `json:"meta,omitempty"`
}

// Meta carries pagination details for list responses.
type Meta struct {
	Total  int64 `json:"total"`
	Limit  int   `json:"limit"`
	Offset int   `json:"offset"`
}

// OK writes a 200 with data.
func OK(c echo.Context, data any) error {
	return c.JSON(http.StatusOK, Envelope{Data: data})
}

// Created writes a 201 with data.
func Created(c echo.Context, data any) error {
	return c.JSON(http.StatusCreated, Envelope{Data: data})
}

// List writes a 200 with data and pagination meta.
func List(c echo.Context, data any, meta Meta) error {
	return c.JSON(http.StatusOK, Envelope{Data: data, Meta: &meta})
}

// NoContent writes a 204.
func NoContent(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}

// Bind decodes the request body into v and converts failures to a 400.
func Bind(c echo.Context, v any) error {
	if err := c.Bind(v); err != nil {
		return BadRequest("invalid request body").WithCause(err)
	}
	return nil
}
