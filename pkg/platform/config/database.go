package config

import (
	"fmt"
	"net/url"
)

type Database struct {
	Scheme string `mapstructure:"scheme"`
	User   string `mapstructure:"user"`
	// Do not export as json and set fron env vars.
	Password string `env:"DB_PASSWORD" json:"-" mapstructure:"-"`
	Host     string `mapstructure:"host"`
	Port     Port   `mapstructure:"port"`
	Name     string `mapstructure:"name"`
	SSLMode  string `mapstructure:"sslmode"`
}

func (d *Database) URL() string {
	scheme := d.Scheme
	if scheme == "" {
		scheme = "postgres"
	}

	u := &url.URL{
		Scheme: scheme,
		Host:   fmt.Sprintf("%s%s", d.Host, d.Port.String()),
		Path:   d.Name,
	}

	if d.SSLMode != "" {
		q := u.Query()
		q.Add("sslmode", d.SSLMode)
		u.RawQuery = q.Encode()
	}

	if d.User != "" {
		if d.Password != "" {
			u.User = url.UserPassword(d.User, d.Password)
		} else {
			u.User = url.User(d.User)
		}
	}

	return u.String()
}
