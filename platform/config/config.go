// libs/ogl/platform/config/config.go
package oglpfconfig

import (
	"fmt"
)

// The platform config should be provie environnement variables mixing.
type Config interface {
	// GetAppEnv returns the name of the app. Usefull as login prefix.
	GetAppEnv() fmt.Stringer
	// GetAppEnv returns the name of the app. Usefull as login prefix.
	GetAppName() string
	// GetPort returns the port of the HTTP server.
	GetServerPort() string
	// GetHost returns the host of the HTTP server.
	GetServerHost() string
	// GetDatabaseURL returns the URL of the database.
	GetDatabaseURL() string
}
