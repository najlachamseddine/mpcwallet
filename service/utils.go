package service

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	HeaderKeyVersion = "X-MPCWallet-Version"
)

type responseWriter struct {
	http.ResponseWriter
	status int
}

func LoggingMiddlewareLogrus(logger *logrus.Entry, next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					w.WriteHeader(http.StatusInternalServerError)

					method := ""
					url := ""
					if r != nil {
						method = r.Method
						url = r.URL.EscapedPath()
					}

					logger.WithFields(logrus.Fields{
						"err":    err,
						"trace":  string(debug.Stack()),
						"method": r.Method,
					}).Error(fmt.Sprintf("http request panic: %s %s", method, url))
				}
			}()
			start := time.Now()
			wrapped := wrapResponseWriter(w)
			next.ServeHTTP(wrapped, r)
			logger.WithFields(logrus.Fields{
				"status":   wrapped.status,
				"method":   r.Method,
				"path":     r.URL.EscapedPath(),
				"duration": fmt.Sprintf("%f", time.Since(start).Seconds()),
			}).Info(fmt.Sprintf("http: %s %s %d", r.Method, r.URL.EscapedPath(), wrapped.status))
		},
	)
}

func wrapResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w}
}
