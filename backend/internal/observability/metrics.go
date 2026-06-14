package observability

import (
	"net/http"
	_ "net/http/pprof"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	httpRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Current number of HTTP requests in flight",
		},
	)

	femSolvesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "fem_solves_total",
			Help: "Total number of FEM solves by type",
		},
		[]string{"analysis_type", "bridge_id"},
	)

	femSolveDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "fem_solve_duration_seconds",
			Help:    "Duration of FEM solves in seconds",
			Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0},
		},
		[]string{"analysis_type"},
	)

	craftAnalysisTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "craft_analysis_total",
			Help: "Total number of craft analysis runs",
		},
	)

	alertsTriggeredTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "alerts_triggered_total",
			Help: "Total alerts triggered by level",
		},
		[]string{"alert_level", "alert_type"},
	)

	alertsPublishedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "alerts_published_total",
			Help: "Total alerts published to MQTT",
		},
		[]string{"status"},
	)

	sensorIngestTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sensor_ingest_total",
			Help: "Total sensor readings ingested",
		},
		[]string{"bridge_id", "sensor_type"},
	)

	mqttOfflineQueueSize = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "mqtt_offline_queue_size",
			Help: "Current MQTT offline alert queue size",
		},
	)
)

func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		httpRequestsInFlight.Inc()
		defer httpRequestsInFlight.Dec()

		c.Next()

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())

		httpRequestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		httpRequestDuration.WithLabelValues(c.Request.Method, path).Observe(duration)
	}
}

func MetricsHandler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

func PprofHandler() http.Handler {
	return http.DefaultServeMux
}

func RecordFEMSolve(analysisType string, bridgeID int, duration time.Duration) {
	femSolvesTotal.WithLabelValues(analysisType, strconv.Itoa(bridgeID)).Inc()
	femSolveDuration.WithLabelValues(analysisType).Observe(duration.Seconds())
}

func RecordCraftAnalysis() {
	craftAnalysisTotal.Inc()
}

func RecordAlertTriggered(alertLevel, alertType string) {
	alertsTriggeredTotal.WithLabelValues(alertLevel, alertType).Inc()
}

func RecordAlertPublished(status string) {
	alertsPublishedTotal.WithLabelValues(status).Inc()
}

func RecordSensorIngest(bridgeID int, sensorType string) {
	sensorIngestTotal.WithLabelValues(strconv.Itoa(bridgeID), sensorType).Inc()
}

func SetMQTTQueueSize(size int) {
	mqttOfflineQueueSize.Set(float64(size))
}
