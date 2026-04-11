package stub_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/joaoGMPereira/autocut/server/internal/stub"
)

func TestNotImplemented(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	stub.NotImplemented(rr, req)

	if rr.Code != http.StatusNotImplemented {
		t.Errorf("expected 501, got %d", rr.Code)
	}
	if rr.Body.String() != `{"error":"not implemented"}` {
		t.Errorf("unexpected body: %s", rr.Body.String())
	}
}
