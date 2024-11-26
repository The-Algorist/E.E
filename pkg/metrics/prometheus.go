package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics represents all application metrics
type Metrics struct {
	// HTTP metrics
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec

	// Encryption metrics
	EncryptionJobsTotal    *prometheus.CounterVec
	EncryptionJobsDuration *prometheus.HistogramVec
	ActiveEncryptionJobs   prometheus.Gauge
}

// NewMetrics creates and registers all application metrics
func NewMetrics(namespace string) *Metrics {
	m := &Metrics{}

	// HTTP metrics
	m.HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "http_requests_total",
			Help:      "Total number of HTTP requests processed",
		},
		[]string{"method", "path", "status"},
	)

	m.HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "http_request_duration_seconds",
			Help:      "Duration of HTTP requests in seconds",
			Buckets:   []float64{0.001, 0.01, 0.1, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path"},
	)

	// Encryption metrics
	m.EncryptionJobsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "encryption_jobs_total",
			Help:      "Total number of encryption jobs processed",
		},
		[]string{"status"},
	)

	m.EncryptionJobsDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "encryption_job_duration_seconds",
			Help:      "Duration of encryption jobs in seconds",
			Buckets:   prometheus.LinearBuckets(1, 5, 20), // 20 buckets, starting at 1, width 5
		},
		[]string{"status"},
	)

	m.ActiveEncryptionJobs = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "encryption_jobs_active",
			Help:      "Number of currently active encryption jobs",
		},
	)

	return m
}

// RecordHTTPRequest records metrics for an HTTP request
func (m *Metrics) RecordHTTPRequest(method, path string, status int) {
	m.HTTPRequestsTotal.WithLabelValues(method, path, string(rune(status))).Inc()
}

// ObserveHTTPRequestDuration records the duration of an HTTP request
func (m *Metrics) ObserveHTTPRequestDuration(method, path string, duration float64) {
	m.HTTPRequestDuration.WithLabelValues(method, path).Observe(duration)
}

// RecordEncryptionJob records metrics for an encryption job
func (m *Metrics) RecordEncryptionJob(status string) {
	m.EncryptionJobsTotal.WithLabelValues(status).Inc()
}

// ObserveEncryptionJobDuration records the duration of an encryption job
func (m *Metrics) ObserveEncryptionJobDuration(status string, duration float64) {
	m.EncryptionJobsDuration.WithLabelValues(status).Observe(duration)
}

// SetActiveEncryptionJobs sets the current number of active encryption jobs
func (m *Metrics) SetActiveEncryptionJobs(count int) {
	m.ActiveEncryptionJobs.Set(float64(count))
}

// IncrementActiveEncryptionJobs increments the active jobs counter
func (m *Metrics) IncrementActiveEncryptionJobs() {
	m.ActiveEncryptionJobs.Inc()
}

// DecrementActiveEncryptionJobs decrements the active jobs counter
func (m *Metrics) DecrementActiveEncryptionJobs() {
	m.ActiveEncryptionJobs.Dec()
}