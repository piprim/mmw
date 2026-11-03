package connect

import (
	"context"
	"errors"

	"github.com/google/uuid"
	pfmiddleware "[[.PlatformPath]]/pkg/platform/middleware"
	defauth "[[.ContractsPath]]/go/application/auth"
	authv1 "[[.ContractsPath]]/go/network/auth/v1"
)

// NewTokenValidator wraps an AuthPrivateService as a platform TokenValidator.
// The returned function validates a bearer token by calling svc.ValidateToken
// and parses the user UUID from the response. A token renewed by auth is passed
// on for the middleware to send back. A response with IsValid false (the HTTP
// transport's way of rejecting a token) is an error.
func NewTokenValidator(svc defauth.AuthPrivateService) pfmiddleware.TokenValidator {
	return func(ctx context.Context, token string) (pfmiddleware.TokenValidation, error) {
		resp, err := svc.ValidateToken(ctx, &authv1.ValidateTokenRequest{Token: token})
		if err != nil {
			//nolint:wrapcheck // err is not wrapped.
			return pfmiddleware.TokenValidation{}, err
		}
		if !resp.GetIsValid() {
			return pfmiddleware.TokenValidation{}, errors.New("token is invalid or expired")
		}

		userID, err := uuid.Parse(resp.GetUserId())
		if err != nil {
			return pfmiddleware.TokenValidation{}, err //nolint:wrapcheck // err is not wrapped.
		}

		return pfmiddleware.TokenValidation{UserID: userID, RenewedToken: resp.GetRenewedToken()}, nil
	}
}
