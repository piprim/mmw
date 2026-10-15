package connect_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "[[.ContractsPath]]/go/network/auth/v1"

	. "[[.OrgPrefix]]/[[.Name]]/internal/adapters/inbound/connect"
)

type mockAuthPrivateService struct {
	userID  uuid.UUID
	invalid bool
	err     error
}

func (m *mockAuthPrivateService) ValidateToken(
	_ context.Context, _ *authv1.ValidateTokenRequest,
) (*authv1.ValidateTokenResponse, error) {
	if m.err != nil {
		return nil, m.err
	}

	return &authv1.ValidateTokenResponse{IsValid: !m.invalid, UserId: m.userID.String()}, nil
}

func TestNewTokenValidator_ValidService_ReturnsUUID(t *testing.T) {
	userID := uuid.New()
	svc := &mockAuthPrivateService{userID: userID}

	validate := NewTokenValidator(svc)
	got, err := validate(context.Background(), "any-token")

	t.Run("returns the user ID", func(t *testing.T) {
		require.NoError(t, err)
		assert.Equal(t, userID, got)
	})
}

func TestNewTokenValidator_ServiceError_PropagatesError(t *testing.T) {
	svc := &mockAuthPrivateService{err: errors.New("invalid token")}

	validate := NewTokenValidator(svc)
	_, err := validate(context.Background(), "bad-token")

	t.Run("returns an error", func(t *testing.T) {
		assert.Error(t, err)
	})
}

func TestNewTokenValidator_InvalidResponse_ReturnsError(t *testing.T) {
	svc := &mockAuthPrivateService{userID: uuid.New(), invalid: true}

	validate := NewTokenValidator(svc)
	got, err := validate(context.Background(), "expired-token")

	t.Run("returns an error", func(t *testing.T) {
		assert.Error(t, err)
	})
	t.Run("returns no user ID", func(t *testing.T) {
		assert.Equal(t, uuid.Nil, got)
	})
}
