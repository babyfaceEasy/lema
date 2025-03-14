package middlewares

import (
	"net/http"
	"strings"
)

func (m Middleware) CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			originMap := make(map[string]string)
			// assume M.config.CorsWhiteLisT returns a comma-separated string
			origins := strings.Split(m.config.CorsWhiteList, ",")
			for _, s := range origins {
				trimmed := strings.TrimSpace(s)
				originMap[trimmed] = trimmed
			}

			if allowed, ok := originMap[origin]; ok {
				w.Header().Set("Access-Control-Allow-Origin", allowed)
			}
		}

		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Accept, origin, Cache-Control, X-Requested-With")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Range")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, PATCH, DELETE")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
