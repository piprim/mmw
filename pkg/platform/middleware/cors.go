package middleware

import (
	"net/http"

	connectcors "connectrpc.com/cors"
	pfconfig "github.com/piprim/mmw/platform/config"

	"github.com/rs/cors"
)

// CORSMiddleware adds CORS support for Connect, gRPC, and gRPC-Web
func CORSMiddleware(cfg *pfconfig.Server) Middleware {
	// Strict host if nothing is provided
	allowed := []string{cfg.Host}

	if len(cfg.AllowedOrigins) > 0 {
		allowed = cfg.AllowedOrigins
	}

	c := cors.New(cors.Options{
		AllowedOrigins: allowed,
		AllowedMethods: connectcors.AllowedMethods(),
		AllowedHeaders: connectcors.AllowedHeaders(),
		ExposedHeaders: connectcors.ExposedHeaders(),
		MaxAge:         maxAge,
	})

	return func(next http.Handler) http.Handler {
		return c.Handler(next)
	}
}
