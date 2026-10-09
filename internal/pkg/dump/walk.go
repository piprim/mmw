package dump

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"
)

// batchSize bounds the ctid array sent in one query.
const batchSize = 5000

// rowSet holds the selected rows: table key → set of ctids.
type rowSet map[string]map[string]struct{}

// add records ctids for table and returns the ones not seen before.
func (r rowSet) add(table string, ctids []string) []string {
	var added []string
	for _, c := range ctids {
		set := r[table]
		if set == nil {
			set = make(map[string]struct{})
			r[table] = set
		}
		if _, ok := set[c]; !ok {
			set[c] = struct{}{}
			added = append(added, c)
		}
	}

	return added
}

func (r rowSet) sorted(table string) []string {
	return slices.Sorted(maps.Keys(r[table]))
}

// summary writes the row count of every table, largest first.
func (r rowSet) summary(w io.Writer) {
	keys := slices.Collect(maps.Keys(r))
	slices.SortFunc(keys, func(a, b string) int {
		return cmp.Or(cmp.Compare(len(r[b]), len(r[a])), strings.Compare(a, b))
	})
	for _, k := range keys {
		_, _ = fmt.Fprintf(w, "%8d  %s\n", len(r[k]), k)
	}
}

// walk selects the seed rows, expands each seed to its children up to its
// depth, then adds every row the selection references, and the rows attached
// to it by an attach rule, until nothing is new. Only seed rows expand to
// children; a row is expanded at most once, by the first seed that reaches it.
func walk(ctx context.Context, conn *pgx.Conn, cat *catalog, cfg *Config) (rowSet, error) {
	for i := range cfg.Seeds {
		if s := &cfg.Seeds[i]; cat.Tables[s.key()] == nil {
			return nil, fmt.Errorf("dump: seed table %s not found (or owned by an extension)", s.key())
		}
	}
	for _, a := range cfg.Attach {
		child, parent := qualify(a.Table), qualify(a.To)
		if !slices.ContainsFunc(cat.FKs, func(fk foreignKey) bool { return fk.Child == child && fk.Parent == parent }) {
			return nil, fmt.Errorf("dump: attach: no foreign key from %s to %s", child, parent)
		}
		if cfg.excluded(child) {
			return nil, fmt.Errorf("dump: table %s is both attached and excluded", child)
		}
	}

	w := &walker{conn: conn, cat: cat, cfg: cfg, rows: make(rowSet)}
	for i := range cfg.Seeds {
		if err := w.expandSeed(ctx, &cfg.Seeds[i]); err != nil {
			return nil, err
		}
	}
	if err := w.closeSelection(ctx); err != nil {
		return nil, err
	}

	return w.rows, nil
}

// walker holds what every step of a walk reads, and the rows selected so far.
type walker struct {
	conn *pgx.Conn
	cat  *catalog
	cfg  *Config
	rows rowSet
}

func (w *walker) expandSeed(ctx context.Context, s *Seed) error {
	conn, cat, cfg, rows := w.conn, w.cat, w.cfg, w.rows
	key := s.key()
	ctids, err := queryStrings(ctx, conn, seedSQL(cat.Tables[key], s))
	if err != nil {
		return fmt.Errorf("dump: seed %s: %w", key, err)
	}

	frontier := map[string][]string{key: rows.add(key, ctids)}
	for range s.childDepth() {
		next := make(map[string][]string)
		for _, parent := range slices.Sorted(maps.Keys(frontier)) {
			for _, fk := range cat.children(parent) {
				if cfg.excluded(fk.Child) {
					continue
				}
				query := childSQL(cat.Tables[fk.Parent], cat.Tables[fk.Child], &fk, cfg.childLimit(s, fk.Child))
				found, err := queryBatched(ctx, conn, query, frontier[parent])
				if err != nil {
					return fmt.Errorf("dump: children %s of %s: %w", fk.Child, fk.Parent, err)
				}
				next[fk.Child] = append(next[fk.Child], rows.add(fk.Child, found)...)
			}
		}
		frontier = next
	}

	return nil
}

// closeSelection adds, for every selected row, the rows it references and
// all the rows attached to it by an attach rule, until nothing is new. Sets
// only grow and are finite, so FK cycles and self-FKs terminate.
func (w *walker) closeSelection(ctx context.Context) error {
	rows := w.rows
	work := make(map[string][]string)
	for key := range rows {
		work[key] = rows.sorted(key)
	}
	for len(work) > 0 {
		next := make(map[string][]string)
		for _, key := range slices.Sorted(maps.Keys(work)) {
			found, err := w.neighbours(ctx, key, work[key])
			if err != nil {
				return err
			}
			for _, table := range slices.Sorted(maps.Keys(found)) {
				if added := rows.add(table, found[table]); len(added) > 0 {
					next[table] = append(next[table], added...)
				}
			}
		}
		work = next
	}

	return nil
}

// neighbours returns, per table, the rows the rows ctids of the table key
// reference and the rows attached to them by an attach rule.
func (w *walker) neighbours(ctx context.Context, key string, ctids []string) (map[string][]string, error) {
	conn, cat, cfg := w.conn, w.cat, w.cfg
	found := make(map[string][]string)
	for _, fk := range cat.parents(key) {
		query := parentSQL(cat.Tables[fk.Parent], cat.Tables[fk.Child], &fk)
		ids, err := queryBatched(ctx, conn, query, ctids)
		if err != nil {
			return nil, fmt.Errorf("dump: parents %s of %s: %w", fk.Parent, fk.Child, err)
		}
		found[fk.Parent] = append(found[fk.Parent], ids...)
	}
	for _, fk := range cat.children(key) {
		if !cfg.attachedVia(&fk) {
			continue
		}
		query := childSQL(cat.Tables[fk.Parent], cat.Tables[fk.Child], &fk, 0)
		ids, err := queryBatched(ctx, conn, query, ctids)
		if err != nil {
			return nil, fmt.Errorf("dump: attached %s of %s: %w", fk.Child, fk.Parent, err)
		}
		found[fk.Child] = append(found[fk.Child], ids...)
	}

	return found, nil
}

// queryBatched runs query, which takes the ctids as $1 (text[]), over ctids
// in batches.
func queryBatched(ctx context.Context, conn *pgx.Conn, query string, ctids []string) ([]string, error) {
	var out []string
	for chunk := range slices.Chunk(ctids, batchSize) {
		found, err := queryStrings(ctx, conn, query, chunk)
		if err != nil {
			return nil, err
		}
		out = append(out, found...)
	}

	return out, nil
}

// queryStrings returns the first column of every row as text.
func queryStrings(ctx context.Context, conn *pgx.Conn, query string, args ...any) ([]string, error) {
	rows, err := conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	out, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, fmt.Errorf("read rows: %w", err)
	}

	return out, nil
}
