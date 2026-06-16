package metrics_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/andrejsstepanovs/prowiki/internal/metrics"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// Feature: prowiki-gap-analysis
// Validates: Requirements 11.1

// TestIncJobsTotal verifies counter increments produce correct label values.
func TestIncJobsTotal(t *testing.T) {
	r := metrics.NewRegistry()

	r.IncJobsTotal("stage_1", "success")
	r.IncJobsTotal("stage_1", "success")
	r.IncJobsTotal("stage_2", "error")

	c1, err := r.JobsTotal.GetMetricWithLabelValues("stage_1", "success")
	if err != nil {
		t.Fatalf("unexpected error getting metric: %v", err)
	}
	if got := testutil.ToFloat64(c1); got != 2.0 {
		t.Errorf("expected 2.0 for stage_1/success, got %v", got)
	}

	c2, err := r.JobsTotal.GetMetricWithLabelValues("stage_2", "error")
	if err != nil {
		t.Fatalf("unexpected error getting metric: %v", err)
	}
	if got := testutil.ToFloat64(c2); got != 1.0 {
		t.Errorf("expected 1.0 for stage_2/error, got %v", got)
	}
}

// TestIncLLMRequests verifies counter increments produce correct label values.
func TestIncLLMRequests(t *testing.T) {
	r := metrics.NewRegistry()

	r.IncLLMRequests("gpt-4o", "success")
	r.IncLLMRequests("gpt-3.5-turbo", "error")

	c1, err := r.LLMRequests.GetMetricWithLabelValues("gpt-4o", "success")
	if err != nil {
		t.Fatalf("unexpected error getting metric: %v", err)
	}
	if got := testutil.ToFloat64(c1); got != 1.0 {
		t.Errorf("expected 1.0 for gpt-4o/success, got %v", got)
	}

	c2, err := r.LLMRequests.GetMetricWithLabelValues("gpt-3.5-turbo", "error")
	if err != nil {
		t.Fatalf("unexpected error getting metric: %v", err)
	}
	if got := testutil.ToFloat64(c2); got != 1.0 {
		t.Errorf("expected 1.0 for gpt-3.5-turbo/error, got %v", got)
	}
}

// TestHandler verifies Handler() returns a valid http.Handler that serves metrics.
func TestHandler(t *testing.T) {
	r := metrics.NewRegistry()
	r.IncJobsTotal("test_stage", "success")

	handler := r.Handler()
	if handler == nil {
		t.Fatal("expected non-nil http.Handler")
	}

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	body := rr.Body.String()
	if !strings.Contains(body, `prowiki_jobs_total{stage="test_stage",status="success"} 1`) {
		t.Errorf("expected body to contain metric, got:\n%s", body)
	}
}
