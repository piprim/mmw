package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDatabase_URL(t *testing.T) {
	t.Run("uses postgres scheme by default", func(t *testing.T) {
		d := &Database{Host: "localhost", Port: 5432, Name: "mydb"}
		assert.Contains(t, d.URL(), "postgres://")
	})

	t.Run("uses custom scheme when specified", func(t *testing.T) {
		d := &Database{Scheme: "postgresql", Host: "localhost", Port: 5432, Name: "mydb"}
		assert.Contains(t, d.URL(), "postgresql://")
	})

	t.Run("includes user without colon when no password", func(t *testing.T) {
		d := &Database{Host: "localhost", Port: 5432, Name: "mydb", User: "admin"}
		url := d.URL()
		assert.Contains(t, url, "admin@")
		assert.NotContains(t, url, ":@")
	})

	t.Run("includes user:password when both set", func(t *testing.T) {
		d := &Database{Host: "localhost", Port: 5432, Name: "mydb", User: "admin", Password: "secret"}
		assert.Contains(t, d.URL(), "admin:secret@")
	})

	t.Run("appends sslmode query param when set", func(t *testing.T) {
		d := &Database{Host: "localhost", Port: 5432, Name: "mydb", SSLMode: "require"}
		assert.Contains(t, d.URL(), "sslmode=require")
	})

	t.Run("omits sslmode query param when empty", func(t *testing.T) {
		d := &Database{Host: "localhost", Port: 5432, Name: "mydb"}
		assert.NotContains(t, d.URL(), "sslmode")
	})

	t.Run("full URL with all fields", func(t *testing.T) {
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
	})
}
