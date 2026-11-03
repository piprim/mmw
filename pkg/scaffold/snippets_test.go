package scaffold_test

import (
	"fmt"
	"testing"

	"github.com/piprim/mmw/pkg/scaffold"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderSnippets(t *testing.T) {
	all := []string{
		scaffold.MarkerField, scaffold.MarkerWire, scaffold.MarkerHandler,
		scaffold.MarkerTopic, scaffold.MarkerEvent, scaffold.MarkerError,
	}
	for _, tc := range []struct {
		shape string
		grpc  bool
		want  []string
	}{
		{"aggregate", true, all},
		{"aggregate", false, []string{
			scaffold.MarkerField, scaffold.MarkerWire, scaffold.MarkerTopic, scaffold.MarkerEvent, scaffold.MarkerError,
		}},
		{"gateway", true, all},
		{"gateway", false, []string{
			scaffold.MarkerField, scaffold.MarkerWire, scaffold.MarkerTopic, scaffold.MarkerEvent, scaffold.MarkerError,
		}},
		{"read-model", true, []string{
			scaffold.MarkerField, scaffold.MarkerWire, scaffold.MarkerHandler, scaffold.MarkerError,
		}},
		{"read-model", false, []string{scaffold.MarkerField, scaffold.MarkerWire, scaffold.MarkerError}},
	} {
		name := fmt.Sprintf("%s grpc=%t", tc.shape, tc.grpc)
		vars := fixtureVars()
		vars["Shape"] = tc.shape
		vars["Grpc"] = tc.grpc
		snippets, err := scaffold.RenderSnippets(vars)
		require.NoError(t, err)
		byMarker := map[string]scaffold.Snippet{}
		markers := make([]string, 0, len(snippets))
		for _, s := range snippets {
			byMarker[s.Marker] = s
			markers = append(markers, s.Marker)
		}

		t.Run(name+" markers", func(t *testing.T) {
			assert.ElementsMatch(t, tc.want, markers)
		})

		t.Run(name+" field and wire land in the composition root", func(t *testing.T) {
			assert.Equal(t, scaffold.Snippet{
				Path: "facturation.go", Marker: scaffold.MarkerField, Text: "facture application.FactureService",
			}, byMarker[scaffold.MarkerField])
			assert.Equal(t, scaffold.Snippet{
				Path: "facturation.go", Marker: scaffold.MarkerWire, Text: "m.facture = newFactureService(cfg, infra)",
			}, byMarker[scaffold.MarkerWire])
		})

		t.Run(name+" error snippet", func(t *testing.T) {
			s := byMarker[scaffold.MarkerError]
			assert.Equal(t, "internal/adapters/inbound/mapper/errors.go", s.Path)
			assert.Equal(t, "if d := domainErrorFrom(err, factureErrorCodes); d != nil {\n\treturn d\n}", s.Text)
		})

		if tc.grpc {
			t.Run(name+" handler snippet", func(t *testing.T) {
				s := byMarker[scaffold.MarkerHandler]
				assert.Equal(t, "facturation.go", s.Path)
				assert.Contains(t, s.Text,
					"mount(facturationv1connect.NewFactureServiceHandler(connecthandler.NewFactureHandler(m.facture), interceptors))")
				assert.Contains(t, s.Text, `health["facture"] = m.facture.Health`)
				assert.Contains(t, s.Text, "names = append(names, facturationv1connect.FactureServiceName)")
			})
		}

		if tc.shape != "read-model" {
			t.Run(name+" topic and event snippets", func(t *testing.T) {
				assert.Equal(t, scaffold.Snippet{
					Path:   "internal/adapters/outbound/events/topics.go",
					Marker: scaffold.MarkerTopic, Text: "m[factureEventType] = factureTopic",
				}, byMarker[scaffold.MarkerTopic])
				assert.Equal(t, scaffold.Snippet{
					Path:   "internal/adapters/outbound/events/proto.go",
					Marker: scaffold.MarkerEvent, Text: "if p := factureToProto(event); p != nil {\n\treturn p\n}",
				}, byMarker[scaffold.MarkerEvent])
			})
		}
	}
}
