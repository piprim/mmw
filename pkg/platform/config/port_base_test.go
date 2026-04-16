package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPort_String(t *testing.T) {
	tests := []struct {
		port     Port
		expected string
	}{
		{5432, ":5432"},
		{8080, ":8080"},
		{80, ":80"},
		{443, ":443"},
		{0, ":0"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.port.String())
		})
	}
}

func TestBase_GetAppEnv(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	b := &Base{Environment: EnvironmentProduction}
	stringer := b.GetAppEnv()

	assert.NotNil(t, stringer)
	assert.Equal(t, "production", stringer.String())
}

func TestBase_GetAppEnv_Development(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	b := &Base{Environment: EnvironmentDevelopment}
	assert.Equal(t, "development", b.GetAppEnv().String())
}
