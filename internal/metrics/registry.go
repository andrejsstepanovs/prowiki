package metrics

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Registry holds all Prometheus metrics for the prowiki daemon.
type Registry struct {
	// JobsTotal counts job completions by pipeline stage and outcome.
	JobsTotal *prometheus.CounterVec
	// LLMRequests counts outgoing LLM calls by model name and outcome.
	LLMRequests *prometheus.CounterVec
	// QueueDepth tracks current job counts in the queue by status.
	QueueDepth *prometheus.GaugeVec

	promReg *prometheus.Registry
}

// NewRegistry creates a new Registry with all metrics registered on a fresh
// Prometheus registry (not the global default, to allow multiple instances in
// tests without duplicate-registration panics).
func NewRegistry() *Registry {
	reg := prometheus.NewRegistry()

	jobsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "prowiki_jobs_total",
			Help: "Total number of jobs processed, partitioned by stage and status.",
		},
		[]string{"stage", "status"},
	)

	llmRequests := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "prowiki_llm_requests_total",
			Help: "Total number of outgoing LLM requests, partitioned by model and status.",
		},
		[]string{"model", "status"},
	)

	queueDepth := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "prowiki_queue_depth",
			Help: "Current number of jobs in the queue, partitioned by status.",
		},
		[]string{"status"},
	)

	reg.MustRegister(jobsTotal, llmRequests, queueDepth)

	return &Registry{
		JobsTotal:   jobsTotal,
		LLMRequests: llmRequests,
		QueueDepth:  queueDepth,
		promReg:     reg,
	}
}

// IncJobsTotal increments the prowiki_jobs_total counter for the given stage
// and status label combination (e.g. stage="level_1_overview", status="success").
func (r *Registry) IncJobsTotal(stage, status string) {
	r.JobsTotal.WithLabelValues(stage, status).Inc()
}

// IncLLMRequests increments the prowiki_llm_requests_total counter for the
// given model and status label combination (e.g. model="gpt-4o", status="error").
func (r *Registry) IncLLMRequests(model, status string) {
	r.LLMRequests.WithLabelValues(model, status).Inc()
}

// SetQueueDepth queries the job_queue table and updates the prowiki_queue_depth
// gauge for each status value present in the database.
func (r *Registry) SetQueueDepth(ctx context.Context, db *sql.DB) {
	rows, err := db.QueryContext(ctx, `SELECT status, COUNT(*) FROM job_queue GROUP BY status`)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			continue
		}
		r.QueueDepth.WithLabelValues(status).Set(float64(count))
	}
}

// Handler returns an http.Handler that serves the Prometheus metrics page.
// Wire this to GET /metrics in the API server.
func (r *Registry) Handler() http.Handler {
	return promhttp.HandlerFor(r.promReg, promhttp.HandlerOpts{})
}
