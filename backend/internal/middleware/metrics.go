package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"data-platform/internal/metrics"
)

func HTTPMetrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		if shouldSkipMetrics(c.Request.URL.Path) {
			c.Next()
			return
		}

		metrics.BackendHTTPInFlightRequests.Inc()
		start := time.Now()
		defer metrics.BackendHTTPInFlightRequests.Dec()

		c.Next()

		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		method := c.Request.Method
		statusClass := httpStatusClass(c.Writer.Status())

		metrics.BackendHTTPRequestsTotal.WithLabelValues(method, route, statusClass).Inc()
		metrics.BackendHTTPRequestDurationSeconds.WithLabelValues(method, route, statusClass).Observe(time.Since(start).Seconds())
	}
}

func shouldSkipMetrics(path string) bool {
	switch path {
	case "/metrics", "/healthz", "/healthz/outbox":
		return true
	default:
		return false
	}
}

func httpStatusClass(status int) string {
	if status <= 0 {
		return "unknown"
	}
	return strconv.Itoa(status/100) + "xx"
}
