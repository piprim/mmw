package dump

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"
)

// table is a dumpable table: an ordinary table that no extension owns.
type table struct {
	Schema  string
	Name    string
	Columns []string // in ordinal order
	PK      []string // in key order; empty when the table has no primary key
}

func (t *table) key() string {
	return t.Schema + "." + t.Name
}

// foreignKey is a FK constraint from Child.ChildCols to Parent.ParentCols.
type foreignKey struct {
	Child      string // schema.table
	Parent     string // schema.table
	ChildCols  []string
	ParentCols []string
}

// sequenceUse is a column whose default or identity draws from a sequence.
type sequenceUse struct {
	Table  string // schema.table
	Column string
}

// sequence is a sequence with every dumpable column that draws from it.
type sequence struct {
	Name string // schema-qualified, identifiers quoted where needed
	Uses []sequenceUse
}

type catalog struct {
	Tables    map[string]*table
	Keys      []string // table keys, sorted
	FKs       []foreignKey
	Sequences []sequence
}

// children returns the FKs that reference parent.
func (c *catalog) children(parent string) []foreignKey {
	var out []foreignKey
	for _, fk := range c.FKs {
		if fk.Parent == parent {
			out = append(out, fk)
		}
	}

	return out
}

// parents returns the FKs declared on child.
func (c *catalog) parents(child string) []foreignKey {
	var out []foreignKey
	for _, fk := range c.FKs {
		if fk.Child == child {
			out = append(out, fk)
		}
	}

	return out
}

const tablesQuery = `
SELECT n.nspname, c.relname,
       array(SELECT a.attname::text
             FROM pg_attribute a
             WHERE a.attrelid = c.oid AND a.attnum > 0 AND NOT a.attisdropped
             ORDER BY a.attnum),
       array(SELECT a.attname::text
             FROM pg_constraint k
             CROSS JOIN unnest(k.conkey) WITH ORDINALITY AS u(attnum, ord)
             JOIN pg_attribute a ON a.attrelid = k.conrelid AND a.attnum = u.attnum
             WHERE k.conrelid = c.oid AND k.contype = 'p'
             ORDER BY u.ord)
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE c.relkind = 'r'
  AND n.nspname <> 'information_schema'
  AND n.nspname NOT LIKE 'pg\_%'
  AND NOT EXISTS (SELECT 1 FROM pg_depend d
                  WHERE d.classid = 'pg_class'::regclass AND d.objid = c.oid AND d.deptype = 'e')`

const foreignKeysQuery = `
SELECT cn.nspname, cc.relname, pn.nspname, pc.relname,
       array(SELECT a.attname::text
             FROM unnest(k.conkey) WITH ORDINALITY AS u(attnum, ord)
             JOIN pg_attribute a ON a.attrelid = k.conrelid AND a.attnum = u.attnum
             ORDER BY u.ord),
       array(SELECT a.attname::text
             FROM unnest(k.confkey) WITH ORDINALITY AS u(attnum, ord)
             JOIN pg_attribute a ON a.attrelid = k.confrelid AND a.attnum = u.attnum
             ORDER BY u.ord)
FROM pg_constraint k
JOIN pg_class cc ON cc.oid = k.conrelid
JOIN pg_namespace cn ON cn.oid = cc.relnamespace
JOIN pg_class pc ON pc.oid = k.confrelid
JOIN pg_namespace pn ON pn.oid = pc.relnamespace
WHERE k.contype = 'f'`

// sequencesQuery links sequences to the columns drawing from them: through a
// column default (serial, or a legacy nextval() without OWNED BY) and through
// an identity column. Only numeric columns count: the max of a column like
// code text DEFAULT 'T' || nextval('s') cannot move the sequence.
const sequencesQuery = `
SELECT format('%I.%I', sn.nspname, s.relname), n.nspname, c.relname, a.attname::text
FROM pg_depend d
JOIN pg_attrdef ad ON d.classid = 'pg_attrdef'::regclass AND d.objid = ad.oid
JOIN pg_class s ON d.refclassid = 'pg_class'::regclass AND d.refobjid = s.oid AND s.relkind = 'S'
JOIN pg_namespace sn ON sn.oid = s.relnamespace
JOIN pg_class c ON c.oid = ad.adrelid
JOIN pg_namespace n ON n.oid = c.relnamespace
JOIN pg_attribute a ON a.attrelid = ad.adrelid AND a.attnum = ad.adnum
WHERE a.atttypid IN ('int2'::regtype, 'int4'::regtype, 'int8'::regtype, 'numeric'::regtype)
UNION
SELECT format('%I.%I', sn.nspname, s.relname), n.nspname, c.relname, a.attname::text
FROM pg_depend d
JOIN pg_class s ON d.classid = 'pg_class'::regclass AND d.objid = s.oid AND s.relkind = 'S'
JOIN pg_namespace sn ON sn.oid = s.relnamespace
JOIN pg_class c ON d.refclassid = 'pg_class'::regclass AND d.refobjid = c.oid
JOIN pg_namespace n ON n.oid = c.relnamespace
JOIN pg_attribute a ON a.attrelid = c.oid AND a.attnum = d.refobjsubid
WHERE d.deptype = 'i'`

// loadCatalog reads the dumpable tables, the FKs between them and the
// sequences their columns draw from. Everything is sorted so a dump is
// deterministic.
func loadCatalog(ctx context.Context, conn *pgx.Conn) (*catalog, error) {
	cat := &catalog{Tables: make(map[string]*table)}

	err := queryRows(ctx, conn, tablesQuery, func(rows pgx.Rows) error {
		t := &table{}
		if err := rows.Scan(&t.Schema, &t.Name, &t.Columns, &t.PK); err != nil {
			return fmt.Errorf("scan table: %w", err)
		}
		cat.Tables[t.key()] = t

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("dump: read tables: %w", err)
	}
	cat.Keys = slices.Sorted(maps.Keys(cat.Tables))

	err = queryRows(ctx, conn, foreignKeysQuery, func(rows pgx.Rows) error {
		var childSchema, childName, parentSchema, parentName string
		var fk foreignKey
		err := rows.Scan(&childSchema, &childName, &parentSchema, &parentName, &fk.ChildCols, &fk.ParentCols)
		if err != nil {
			return fmt.Errorf("scan foreign key: %w", err)
		}
		fk.Child = childSchema + "." + childName
		fk.Parent = parentSchema + "." + parentName
		// A FK to or from an extension-owned table is never followed.
		if cat.Tables[fk.Child] != nil && cat.Tables[fk.Parent] != nil {
			cat.FKs = append(cat.FKs, fk)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("dump: read foreign keys: %w", err)
	}
	slices.SortFunc(cat.FKs, func(a, b foreignKey) int {
		return cmp.Or(
			strings.Compare(a.Child, b.Child),
			strings.Compare(a.Parent, b.Parent),
			strings.Compare(strings.Join(a.ChildCols, ","), strings.Join(b.ChildCols, ",")),
		)
	})

	uses := make(map[string][]sequenceUse)
	err = queryRows(ctx, conn, sequencesQuery, func(rows pgx.Rows) error {
		var name, schema, tableName, column string
		if err := rows.Scan(&name, &schema, &tableName, &column); err != nil {
			return fmt.Errorf("scan sequence: %w", err)
		}
		if key := schema + "." + tableName; cat.Tables[key] != nil {
			uses[name] = append(uses[name], sequenceUse{Table: key, Column: column})
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("dump: read sequences: %w", err)
	}
	for _, name := range slices.Sorted(maps.Keys(uses)) {
		u := uses[name]
		slices.SortFunc(u, func(a, b sequenceUse) int {
			return cmp.Or(strings.Compare(a.Table, b.Table), strings.Compare(a.Column, b.Column))
		})
		cat.Sequences = append(cat.Sequences, sequence{Name: name, Uses: u})
	}

	return cat, nil
}

// queryRows runs query on conn and calls scan for each row.
func queryRows(ctx context.Context, conn *pgx.Conn, query string, scan func(pgx.Rows) error) error {
	rows, err := conn.Query(ctx, query)
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		if err := scan(rows); err != nil {
			return err
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("read rows: %w", err)
	}

	return nil
}
