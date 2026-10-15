package sqlc

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// Type arguments of the presence.Of[T] overrides declared in sqlc.yaml.
//
// sqlc adds an import for the override's own package only, never for the
// packages named inside a type argument, so `presence.Of[time.Time]` would
// not compile. These aliases give the arguments a name that resolves within
// the dao package; being aliases, presence.Of[Time] is presence.Of[time.Time].
//
// This file is hand-written: sqlc only rewrites the files it generates.
type (
	Time     = time.Time
	UUID     = uuid.UUID
	Numeric  = pgtype.Numeric
	Interval = pgtype.Interval
	Hstore   = pgtype.Hstore
	JSON     = json.RawMessage
)
