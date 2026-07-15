package rest

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"nexora/internal/pkg/logger"
)

type Router struct {
	*chi.Mux
	log *logger.Logger
}

func NewRouter(log *logger.Logger) *Router {
	r := chi.NewRouter()

	router := &Router{
		Mux: r,
		log: log,
	}

	router.registerMiddleware()
	return router
}

func (rt *Router) registerMiddleware() {
	rt.Use(middleware.RequestID)
	rt.Use(middleware.RealIP)
	rt.Use(middleware.Logger)
	rt.Use(middleware.Recoverer)
	rt.Use(middleware.Heartbeat("/ping"))
	rt.Use(middleware.Timeout(60))

	rt.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://localhost:3001"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	rt.Use(rt.requestLogger)
}

func (rt *Router) requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rt.log.Debug("incoming request",
			"method", r.Method,
			"path", r.URL.Path,
			"remote", r.RemoteAddr,
		)
		next.ServeHTTP(w, r)
	})
}
