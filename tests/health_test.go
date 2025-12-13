package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/Aunali321/korus/internal/api/handlers"
)

func TestHealth(t *testing.T) {
	e := echo.New()
	h := &handlers.Handler{}
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	if err := h.Health(c); err != nil {
		t.Fatalf("health handler error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}
