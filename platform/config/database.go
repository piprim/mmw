package oglpfconfig

import (
	"fmt"
	"net/url"
)

type Database struct {
	Scheme string `mapstructure:"scheme"`
	User   string `mapstructure:"user"`
	// Do not export as json and set fron env vars.
	Password string `env:"DB_PASSWORD" json:"-"`
	Host     string `mapstructure:"host"`
	Port     Port   `mapstructure:"port"`
	Name     string `mapstructure:"name"`
}

func (d *Database) URL() string {
	u := &url.URL{
		Scheme: d.Scheme,
		Host:   fmt.Sprintf("%s%s", d.Host, d.Port.String()),
		Path:   d.Name,
	}

	q := u.Query()
	q.Add("sslmode", "disable")
	u.RawQuery = q.Encode()

	if d.User != "" {
		if d.Password != "" {
			u.User = url.UserPassword(d.User, d.Password)
		} else {
			u.User = url.User(d.User)
		}
	}

	return u.String()
}
