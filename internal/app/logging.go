package app

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

func slogRequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)

			status := ww.Status()
			if status == 0 {
				status = http.StatusOK
			}

			level := slog.LevelInfo
			switch {
			case status >= http.StatusInternalServerError:
				level = slog.LevelError
			case status >= http.StatusBadRequest:
				level = slog.LevelWarn
			}

			logger.LogAttrs(
				r.Context(),
				level,
				"http_request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("query", r.URL.RawQuery),
				slog.String("host", r.Host),
				slog.String("remote_addr", r.RemoteAddr),
				slog.String("user_agent", r.UserAgent()),
				slog.String("request_id", middleware.GetReqID(r.Context())),
				slog.Int("status", status),
				slog.Int("bytes", ww.BytesWritten()),
				slog.Int64("duration_ms", time.Since(started).Milliseconds()),
			)
		})
	}
}

func slogRecoverer(logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.ErrorContext(
						r.Context(),
						"http_panic",
						slog.String("method", r.Method),
						slog.String("path", r.URL.Path),
						slog.String("request_id", middleware.GetReqID(r.Context())),
						slog.String("error", fmt.Sprint(recovered)),
						slog.String("stack", string(debug.Stack())),
					)
					writeHTTPError(w, http.StatusInternalServerError, "internal server error")
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
