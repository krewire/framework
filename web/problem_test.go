package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	ftest "github.com/krewire/framework/test"
)

func TestWebProblem_WriteProblem(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteProblem(rec, http.StatusBadRequest, "Invalid account balance", func(p *Problem) {
		p.Instance = "/accounts/123/withdraw"
		p.Errors = map[string]string{"amount": "must be positive"}
	})

	ftest.Equal(t, http.StatusBadRequest, rec.Code)
	ftest.Equal(t, MimeProblemJSON, rec.Header().Get("Content-Type"))

	var prob Problem
	if err := json.NewDecoder(rec.Body).Decode(&prob); err != nil {
		t.Fatalf("failed to decode Problem JSON: %v", err)
	}

	ftest.Equal(t, http.StatusBadRequest, prob.Status)
	ftest.Equal(t, "Bad Request", prob.Title)
	ftest.Equal(t, "Invalid account balance", prob.Detail)
	ftest.Equal(t, "/accounts/123/withdraw", prob.Instance)
}

func TestWebProblem_ResponseBuilder(t *testing.T) {
	resp := ProblemResponse(http.StatusNotFound, "Resource not found", func(p *Problem) {
		p.Type = "https://example.com/errors/not-found"
	})

	rec := httptest.NewRecorder()
	resp.Write(rec)

	ftest.Equal(t, http.StatusNotFound, rec.Code)
	ftest.Equal(t, MimeProblemJSON, rec.Header().Get("Content-Type"))
}
