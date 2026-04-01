package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDatabase_URL_DefaultScheme(t *testing.T) {
	d := &Database{
		Host: "localhost",
		Port: 5432,
		Name: "mydb",
	}
	assert.Contains(t, d.URL(), "postgres://")
}

func TestDatabase_URL_CustomScheme(t *testing.T) {
	d := &Database{
		Scheme: "postgresql",
		Host:   "localhost",
		Port:   5432,
		Name:   "mydb",
	}
	assert.Contains(t, d.URL(), "postgresql://")
}

func TestDatabase_URL_WithUserOnly(t *testing.T) {
	d := &Database{
		Host: "localhost",
		Port: 5432,
		Name: "mydb",
		User: "admin",
	}
	url := d.URL()
	assert.Contains(t, url, "admin@")
	assert.NotContains(t, url, ":@")
}

func TestDatabase_URL_WithUserAndPassword(t *testing.T) {
	d := &Database{
		Host:     "localhost",
		Port:     5432,
		Name:     "mydb",
		User:     "admin",
		Password: "secret",
	}
	url := d.URL()
	assert.Contains(t, url, "admin:secret@")
}

func TestDatabase_URL_WithSSLMode(t *testing.T) {
	d := &Database{
		Host:    "localhost",
		Port:    5432,
		Name:    "mydb",
		SSLMode: "require",
	}
	url := d.URL()
	assert.Contains(t, url, "sslmode=require")
}

func TestDatabase_URL_NoSSLMode(t *testing.T) {
	d := &Database{
		Host: "localhost",
		Port: 5432,
		Name: "mydb",
	}
	url := d.URL()
	assert.NotContains(t, url, "sslmode")
}

func TestDatabase_URL_Full(t *testing.T) {
	d := &Database{
		Scheme:   "postgres",
		Host:     "db.example.com",
		Port:     5432,
		Name:     "appdb",
		User:     "user",
		Password: "pass",
		SSLMode:  "disable",
	}
	url := d.URL()
	assert.Contains(t, url, "postgres://")
	assert.Contains(t, url, "user:pass@")
	assert.Contains(t, url, "db.example.com:5432")
	assert.Contains(t, url, "/appdb")
	assert.Contains(t, url, "sslmode=disable")
}
