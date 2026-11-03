package scaffold

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/piprim/goplt"
)

// Snippet is the text one slice inserts before Marker in Path, relative to
// the module directory.
type Snippet struct {
	Path, Marker, Text string
}

// snippetSource is one snippet template; skip tells whether it does not
// apply to a slice.
type snippetSource struct {
	path, marker, text string
	skip               func(shape string, grpc bool) bool
}

func never(string, bool) bool { return false }

// rootFile is the module's composition root.
const rootFile = "[[.Name]].go"

// Snippets name only symbols of the package they land in: no import is ever
// inserted, and a module whose slices are all read models must compile.
var snippetSources = []snippetSource{
	{rootFile, MarkerField, "[[.Entity]] application.[[.Entity | pascal]]Service", never},
	{rootFile, MarkerWire, "m.[[.Entity]] = new[[.Entity | pascal]]Service(cfg, infra)", never},
	{
		rootFile, MarkerHandler,
		// Two lines: one would pass lll only for short module and entity names.
		"mount([[.Name]]v1connect.New[[.Entity | pascal]]ServiceHandler(\n" +
			"\tconnecthandler.New[[.Entity | pascal]]Handler(m.[[.Entity]]), interceptors))\n" +
			`health["[[.Entity]]"] = m.[[.Entity]].Health` + "\n" +
			"names = append(names, [[.Name]]v1connect.[[.Entity | pascal]]ServiceName)",
		func(_ string, grpc bool) bool { return !grpc },
	},
	{
		"internal/adapters/outbound/events/topics.go", MarkerTopic,
		// m[<e>EventType]: printf keeps "m[" away from the "[[" delimiter.
		`m[[printf "[%sEventType]" .Entity]] = [[.Entity]]Topic`, isReadModel,
	},
	{
		"internal/adapters/outbound/events/proto.go", MarkerEvent,
		"if p := [[.Entity]]ToProto(event); p != nil {\n\treturn p\n}", isReadModel,
	},
	{
		"internal/adapters/inbound/mapper/errors.go", MarkerError,
		"if d := domainErrorFrom(err, [[.Entity]]ErrorCodes); d != nil {\n\treturn d\n}", never,
	},
}

// isReadModel skips the event snippets: a read model raises no event.
func isReadModel(shape string, _ bool) bool { return shape == "read-model" }

// RenderSnippets renders the snippets that apply to the slice described by
// vars (Name, Entity, Shape, Grpc).
func RenderSnippets(vars map[string]any) ([]Snippet, error) {
	shape, _ := vars["Shape"].(string)
	grpc, _ := vars["Grpc"].(bool)

	snippets := make([]Snippet, 0, len(snippetSources))
	for _, src := range snippetSources {
		if src.skip(shape, grpc) {
			continue
		}
		path, err := renderSnippet(src.path, vars)
		if err != nil {
			return nil, err
		}
		text, err := renderSnippet(src.text, vars)
		if err != nil {
			return nil, err
		}
		snippets = append(snippets, Snippet{Path: path, Marker: src.marker, Text: text})
	}

	return snippets, nil
}

func renderSnippet(src string, vars map[string]any) (string, error) {
	tmpl, err := template.New("snippet").Delims("[[", "]]").Funcs(goplt.DefaultFuncMap()).Parse(src)
	if err != nil {
		return "", fmt.Errorf("parse snippet %q: %w", src, err)
	}

	var b strings.Builder
	if err := tmpl.Execute(&b, vars); err != nil {
		return "", fmt.Errorf("render snippet %q: %w", src, err)
	}

	return b.String(), nil
}
