package oglmiddleware

import (
	"net/http"

	connectcors "connectrpc.com/cors"
	oglpfconfig "github.com/ovya/ogl/platform/config"

	"github.com/rs/cors"
)

// CORSMiddleware adds CORS support for Connect, gRPC, and gRPC-Web
func CORSMiddleware(cfg *oglpfconfig.Server, environment string) Middleware {
	var allowed []string

	if environment == "development" {
		allowed = []string{"*"}
	} else if len(cfg.AllowedOrigins) > 0 {
		allowed = cfg.AllowedOrigins
	} else {
		// Fallback to strict host if nothing is provided
		allowed = []string{cfg.Host}
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
