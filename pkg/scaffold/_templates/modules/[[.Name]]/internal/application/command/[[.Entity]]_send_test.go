package command_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"[[.PlatformPath]]/pkg/platform/authctx"
	"[[.OrgPrefix]]/[[.Name]]/internal/application/command"
	"[[.OrgPrefix]]/[[.Name]]/internal/domain/[[.Entity]]"
	"[[.OrgPrefix]]/[[.Name]]/internal/testhelpers"
)

var testUser = uuid.MustParse("00000000-0000-0000-0000-000000000001")

func authedCtx() context.Context {
	return authctx.WithUserID(context.Background(), testUser)
}

func TestSend[[.Entity | pascal]]Command_Execute(t *testing.T) {
	t.Run("sends through the gateway and publishes one Sent event", func(t *testing.T) {
		gateway := &testhelpers.Fake[[.Entity | pascal]]Gateway{}
		publisher := &testhelpers.RecordingEventPublisher{}
		send := command.NewSend[[.Entity | pascal]]Command(gateway, publisher)

		if err := send.Execute(authedCtx(), "Bonjour"); err != nil {
			t.Fatalf("Execute() unexpected error: %v", err)
		}
		if len(gateway.Sent) != 1 || gateway.Sent[0].Label().String() != "Bonjour" {
			t.Errorf("gateway.Sent = %v", gateway.Sent)
		}
		if len(publisher.Events) != 1 || publisher.Events[0].EventType() != [[.Entity]].EventTypeSent {
			t.Fatalf("published = %v, want one Sent event", publisher.Events)
		}
		if publisher.Events[0].GetUserID() != testUser || publisher.Events[0].GetAggregateID() != "Bonjour" {
			t.Errorf("event user/aggregate = %s/%s", publisher.Events[0].GetUserID(), publisher.Events[0].GetAggregateID())
		}
	})

	t.Run("returns [[.Entity]].ErrInvalidLabel before touching the gateway", func(t *testing.T) {
		gateway := &testhelpers.Fake[[.Entity | pascal]]Gateway{}
		send := command.NewSend[[.Entity | pascal]]Command(gateway, &testhelpers.RecordingEventPublisher{})

		if err := send.Execute(authedCtx(), ""); !errors.Is(err, [[.Entity]].ErrInvalidLabel) {
			t.Errorf("Execute() error = %v, want ErrInvalidLabel", err)
		}
		if len(gateway.Sent) != 0 {
			t.Errorf("gateway called %d times, want 0", len(gateway.Sent))
		}
	})

	t.Run("a failing gateway returns its error and publishes nothing", func(t *testing.T) {
		gateway := &testhelpers.Fake[[.Entity | pascal]]Gateway{Err: [[.Entity]].ErrGatewayUnavailable}
		publisher := &testhelpers.RecordingEventPublisher{}
		send := command.NewSend[[.Entity | pascal]]Command(gateway, publisher)

		if err := send.Execute(authedCtx(), "Bonjour"); !errors.Is(err, [[.Entity]].ErrGatewayUnavailable) {
			t.Errorf("Execute() error = %v, want ErrGatewayUnavailable", err)
		}
		if len(publisher.Events) != 0 {
			t.Errorf("published %d events, want 0", len(publisher.Events))
		}
	})

	t.Run("fails without an authenticated user", func(t *testing.T) {
		gateway := &testhelpers.Fake[[.Entity | pascal]]Gateway{}
		send := command.NewSend[[.Entity | pascal]]Command(gateway, &testhelpers.RecordingEventPublisher{})

		if err := send.Execute(context.Background(), "Bonjour"); err == nil {
			t.Error("Execute() without user expected an error, got nil")
		}
		if len(gateway.Sent) != 0 {
			t.Errorf("gateway called %d times, want 0", len(gateway.Sent))
		}
	})
}
