package dump

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// beginSnapshot opens the transaction every dump query runs in. The settings
// make COPY output independent of the server's defaults, and row_security = off
// makes a row-level security policy that would hide rows an error, as in
// pg_dump, instead of a silently smaller dump.
const beginSnapshot = `BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY;
SET row_security = off;
SET TimeZone = 'UTC';
SET DateStyle = 'ISO, YMD';
SET IntervalStyle = 'postgres';
SET extra_float_digits = 3`

// withSnapshot runs fn on conn inside a read-only repeatable-read
// transaction, so every query sees the same data and ctids stay valid.
func withSnapshot(ctx context.Context, conn *pgx.Conn, fn func() error) error {
	if _, err := conn.Exec(ctx, beginSnapshot); err != nil {
		return fmt.Errorf("dump: begin snapshot: %w", err)
	}
	defer func() { _, _ = conn.Exec(context.Background(), "ROLLBACK") }()

	return fn()
}
