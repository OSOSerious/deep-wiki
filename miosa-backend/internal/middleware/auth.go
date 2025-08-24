package middleware

import (
	"net/http"
	"strings"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	apiRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "profile_api_requests_total",
			Help: "Total profile API requests",
		},
		[]string{"method", "endpoint", "status"},
	)
)

func init() {
	prometheus.MustRegister(apiRequests)
}

// ValidateAuth checks JWT token
func ValidateAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if !strings.HasPrefix(token, "Bearer ") {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			apiRequests.WithLabelValues(r.Method, r.URL.Path, "401").Inc()
			return
		}
		
		// TODO: Validate JWT token
		apiRequests.WithLabelValues(r.Method, r.URL.Path, "200").Inc()
		next(w, r)
	}
}