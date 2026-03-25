package db

import (
	"fmt"
	"reflect"
	"strings"
)

// StructArgs converts a struct with `db` tags into a map[string]any suitable
// for use as named parameters in database queries.
//
// When used with pgx, callers cast: pgx.NamedArgs(db.StructArgs(v)).
//
// Fields without a `db` tag, with db:"-", or with an empty name are skipped.
// Tag options such as db:"name,omitempty" are parsed but ignored — the database
// handles NULL natively so omitempty is a no-op.
//
// Panics if v (after pointer dereference) is not a struct.
func StructArgs[T any](v T) map[string]any {
	args := make(map[string]any)
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		panic(fmt.Sprintf("StructArgs: expected a struct, got %s", rv.Kind()))
	}
	rt := rv.Type()
	for i := range rt.NumField() {
		field := rt.Field(i)
		tag := field.Tag.Get("db")
		if tag == "" || tag == "-" {
			continue
		}
		name := strings.SplitN(tag, ",", 2)[0]
		if name == "" || name == "-" {
			continue
		}
		args[name] = rv.Field(i).Interface()
	}

	return args
}
