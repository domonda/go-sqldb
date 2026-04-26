package conntest

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
)

// runInformationTests exercises every method on [sqldb.Information]
// against a live database. Skipped silently when the driver did not
// supply DDL for the Information fixtures.
func runInformationTests(t *testing.T, config Config) {
	if config.DDL.CreateInfoParent == "" {
		t.Skip("Information tests skipped: CreateInfoParent DDL not provided")
	}

	feat := config.Information

	// fold returns the case-folded form of a name when the driver
	// uppercases unquoted identifiers in its catalog (Oracle), or the
	// name as-is otherwise.
	fold := func(s string) string {
		if feat.CaseFoldsToUpper {
			return strings.ToUpper(s)
		}
		return s
	}

	// stripSchema removes a schema prefix when the test does not care
	// which schema a name resolved to.
	stripSchema := func(qualified string) string {
		if i := strings.LastIndexByte(qualified, '.'); i >= 0 {
			return qualified[i+1:]
		}
		return qualified
	}

	t.Run("CurrentSchema", func(t *testing.T) {
		conn := config.NewConn(t)
		ctx := t.Context()
		s, err := conn.CurrentSchema(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, s, "CurrentSchema must return a non-empty string")
	})

	t.Run("Schemas", func(t *testing.T) {
		conn := config.NewConn(t)
		ctx := t.Context()
		schemas, err := conn.Schemas(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, schemas, "Schemas must return at least one schema")

		current, err := conn.CurrentSchema(ctx)
		require.NoError(t, err)
		assert.Contains(t, schemas, current,
			"Schemas should include CurrentSchema (%q): %v", current, schemas)
	})

	t.Run("Tables", func(t *testing.T) {
		conn := config.NewConn(t)
		setupInfoTable(t, conn, config.DDL.CreateInfoParent, "conntest_info_parent")
		if config.DDL.CreateInfoView != "" {
			setupView(t, conn, config.DDL.CreateInfoView, "conntest_info_view")
		}
		ctx := t.Context()

		tables, err := conn.Tables(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, tables)

		// Find by suffix because the schema portion is vendor-specific.
		want := fold("conntest_info_parent")
		found := slices.ContainsFunc(tables, func(s string) bool {
			return stripSchema(s) == want
		})
		assert.True(t, found, "Tables should include conntest_info_parent: %v", tables)

		if config.DDL.CreateInfoView != "" {
			// Negative assertion: views must NOT show up in Tables.
			viewName := fold("conntest_info_view")
			leaked := slices.ContainsFunc(tables, func(s string) bool {
				return stripSchema(s) == viewName
			})
			assert.False(t, leaked,
				"Tables must not include views (conntest_info_view): %v", tables)
		}
	})

	t.Run("TableExists", func(t *testing.T) {
		conn := config.NewConn(t)
		setupInfoTable(t, conn, config.DDL.CreateInfoParent, "conntest_info_parent")
		ctx := t.Context()

		ok, err := conn.TableExists(ctx, fold("conntest_info_parent"))
		require.NoError(t, err)
		assert.True(t, ok, "TableExists(conntest_info_parent)")

		ok, err = conn.TableExists(ctx, fold("conntest_no_such_table"))
		require.NoError(t, err)
		assert.False(t, ok, "TableExists on a non-existent table must be false")
	})

	if config.DDL.CreateInfoView != "" {
		t.Run("Views", func(t *testing.T) {
			conn := config.NewConn(t)
			setupInfoTable(t, conn, config.DDL.CreateInfoParent, "conntest_info_parent")
			setupView(t, conn, config.DDL.CreateInfoView, "conntest_info_view")
			ctx := t.Context()

			views, err := conn.Views(ctx)
			require.NoError(t, err)
			require.NotEmpty(t, views)

			want := fold("conntest_info_view")
			found := slices.ContainsFunc(views, func(s string) bool {
				return stripSchema(s) == want
			})
			assert.True(t, found, "Views should include conntest_info_view: %v", views)

			// Negative assertion: base tables must NOT show up in Views.
			tableName := fold("conntest_info_parent")
			leaked := slices.ContainsFunc(views, func(s string) bool {
				return stripSchema(s) == tableName
			})
			assert.False(t, leaked,
				"Views must not include base tables (conntest_info_parent): %v", views)
		})

		t.Run("ViewExists", func(t *testing.T) {
			conn := config.NewConn(t)
			setupInfoTable(t, conn, config.DDL.CreateInfoParent, "conntest_info_parent")
			setupView(t, conn, config.DDL.CreateInfoView, "conntest_info_view")
			ctx := t.Context()

			ok, err := conn.ViewExists(ctx, fold("conntest_info_view"))
			require.NoError(t, err)
			assert.True(t, ok)

			ok, err = conn.ViewExists(ctx, fold("conntest_no_such_view"))
			require.NoError(t, err)
			assert.False(t, ok)

			// A view should not be reported as a base table, and a
			// table should not be reported as a view.
			ok, err = conn.TableExists(ctx, fold("conntest_info_view"))
			require.NoError(t, err)
			assert.False(t, ok, "TableExists must not match a view")
			ok, err = conn.ViewExists(ctx, fold("conntest_info_parent"))
			require.NoError(t, err)
			assert.False(t, ok, "ViewExists must not match a base table")
		})
	}

	t.Run("Columns", func(t *testing.T) {
		conn := config.NewConn(t)
		setupInfoTable(t, conn, config.DDL.CreateInfoParent, "conntest_info_parent")
		ctx := t.Context()

		cols, err := conn.Columns(ctx, fold("conntest_info_parent"))
		require.NoError(t, err)
		require.Len(t, cols, 2)

		names := make([]string, len(cols))
		for i, c := range cols {
			names[i] = strings.ToLower(c.Name)
		}
		assert.ElementsMatch(t, []string{"id1", "id2"}, names)

		for _, c := range cols {
			assert.True(t, c.PrimaryKey, "both id1 and id2 are PK columns: %+v", c)
			assert.False(t, c.Generated,
				"plain int columns must not be reported as Generated: %+v", c)
			assert.False(t, c.ReadOnly,
				"plain int columns must not be reported as ReadOnly: %+v", c)
		}
	})

	if config.DDL.CreateInfoGenerated != "" {
		t.Run("Columns_Generated", func(t *testing.T) {
			conn := config.NewConn(t)
			setupInfoTable(t, conn, config.DDL.CreateInfoGenerated, "conntest_info_generated")
			ctx := t.Context()

			cols, err := conn.Columns(ctx, fold("conntest_info_generated"))
			require.NoError(t, err)
			require.Len(t, cols, 3)

			byName := make(map[string]sqldb.ColumnInfo, len(cols))
			for _, c := range cols {
				byName[strings.ToLower(c.Name)] = c
			}

			id, ok := byName["id"]
			require.True(t, ok, "missing id column in %+v", cols)
			assert.False(t, id.Generated, "id is not generated: %+v", id)
			assert.False(t, id.ReadOnly, "id is not read-only: %+v", id)
			assert.True(t, id.PrimaryKey, "id is the primary key: %+v", id)

			gen, ok := byName["gen_col"]
			require.True(t, ok, "missing gen_col column in %+v", cols)
			assert.True(t, gen.Generated,
				"gen_col is GENERATED ALWAYS AS (id + 1): %+v", gen)
			assert.True(t, gen.ReadOnly,
				"every Generated column is also ReadOnly: %+v", gen)

			// Regression: a column with a non-literal DEFAULT clause
			// (e.g. CURRENT_TIMESTAMP / now() / SYSTIMESTAMP) must NOT
			// be reported as Generated or ReadOnly. MySQL 8.0.13+
			// writes "DEFAULT_GENERATED" into information_schema's
			// extra column for such defaults, which a loose substring
			// match for "GENERATED" would catch as a false positive.
			ts, ok := byName["created_at"]
			require.True(t, ok, "missing created_at column in %+v", cols)
			assert.False(t, ts.Generated,
				"created_at has DEFAULT <current-timestamp-expr> but is NOT generated: %+v", ts)
			assert.False(t, ts.ReadOnly,
				"created_at is writable, not read-only: %+v", ts)
			assert.True(t, ts.HasDefault,
				"created_at has a DEFAULT clause: %+v", ts)
		})
	}

	t.Run("ColumnExists", func(t *testing.T) {
		conn := config.NewConn(t)
		setupInfoTable(t, conn, config.DDL.CreateInfoParent, "conntest_info_parent")
		ctx := t.Context()

		ok, err := conn.ColumnExists(ctx, fold("conntest_info_parent"), fold("id1"))
		require.NoError(t, err)
		assert.True(t, ok)

		ok, err = conn.ColumnExists(ctx, fold("conntest_info_parent"), fold("no_such_column"))
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("PrimaryKey_PreservesConstraintOrder", func(t *testing.T) {
		conn := config.NewConn(t)
		setupInfoTable(t, conn, config.DDL.CreateInfoParent, "conntest_info_parent")
		ctx := t.Context()

		pk, err := conn.PrimaryKey(ctx, fold("conntest_info_parent"))
		require.NoError(t, err)
		// The DDL declares PRIMARY KEY (id2, id1) — constraint order
		// is (id2, id1) even though columns are declared (id1, id2).
		require.Len(t, pk, 2)
		assert.Equal(t, fold("id2"), pk[0],
			"PK[0] must be id2 (constraint order, not column declaration order)")
		assert.Equal(t, fold("id1"), pk[1],
			"PK[1] must be id1")
	})

	if config.DDL.CreateInfoChild != "" {
		t.Run("ForeignKeys_CompositeOrdering", func(t *testing.T) {
			conn := config.NewConn(t)
			setupInfoTable(t, conn, config.DDL.CreateInfoParent, "conntest_info_parent")
			setupInfoTable(t, conn, config.DDL.CreateInfoChild, "conntest_info_child")
			ctx := t.Context()

			fks, err := conn.ForeignKeys(ctx, fold("conntest_info_child"))
			require.NoError(t, err)
			require.Len(t, fks, 1, "child has exactly one FK")

			fk := fks[0]
			require.Len(t, fk.Columns, 2, "composite FK has 2 columns")
			require.Len(t, fk.ReferencedColumns, 2)

			// FK is declared as (parent_id2, parent_id1) ->
			// parent (id2, id1). The local and remote slices must
			// match positionally.
			lc := lowerSlice(fk.Columns)
			rc := lowerSlice(fk.ReferencedColumns)
			assert.Equal(t, []string{"parent_id2", "parent_id1"}, lc,
				"Columns must reflect FK declaration order")
			assert.Equal(t, []string{"id2", "id1"}, rc,
				"ReferencedColumns must align positionally with Columns")

			refTbl := strings.ToLower(stripSchema(fk.ReferencedTable))
			assert.Equal(t, "conntest_info_parent", refTbl)

			assert.Equal(t, "CASCADE", fk.OnDelete,
				"DDL specifies ON DELETE CASCADE")
		})
	}

	// Missing-relation behaviour: every method that takes a table or
	// view name must return a wrapped sql.ErrNoRows when no relation
	// of the appropriate kind exists.
	t.Run("Columns_NotFound", func(t *testing.T) {
		conn := config.NewConn(t)
		ctx := t.Context()
		_, err := conn.Columns(ctx, fold("conntest_no_such_relation"))
		assert.ErrorIs(t, err, sql.ErrNoRows)
	})

	t.Run("ColumnExists_NotFound", func(t *testing.T) {
		conn := config.NewConn(t)
		ctx := t.Context()
		_, err := conn.ColumnExists(ctx, fold("conntest_no_such_relation"), fold("anything"))
		assert.ErrorIs(t, err, sql.ErrNoRows)
	})

	t.Run("PrimaryKey_NotFound", func(t *testing.T) {
		conn := config.NewConn(t)
		ctx := t.Context()
		_, err := conn.PrimaryKey(ctx, fold("conntest_no_such_table"))
		assert.ErrorIs(t, err, sql.ErrNoRows)
	})

	t.Run("ForeignKeys_NotFound", func(t *testing.T) {
		conn := config.NewConn(t)
		ctx := t.Context()
		_, err := conn.ForeignKeys(ctx, fold("conntest_no_such_table"))
		assert.ErrorIs(t, err, sql.ErrNoRows)
	})

	// View-vs-table separation: PrimaryKey and ForeignKeys must reject
	// a name that resolves to a view; Columns and ColumnExists must
	// accept it.
	if config.DDL.CreateInfoView != "" {
		t.Run("Columns_OnView", func(t *testing.T) {
			conn := config.NewConn(t)
			setupInfoTable(t, conn, config.DDL.CreateInfoParent, "conntest_info_parent")
			setupView(t, conn, config.DDL.CreateInfoView, "conntest_info_view")
			ctx := t.Context()

			cols, err := conn.Columns(ctx, fold("conntest_info_view"))
			require.NoError(t, err)
			require.Len(t, cols, 2, "view should expose 2 columns")
			names := make([]string, len(cols))
			for i, c := range cols {
				names[i] = strings.ToLower(c.Name)
			}
			assert.ElementsMatch(t, []string{"id1", "id2"}, names)
		})

		t.Run("ColumnExists_OnView", func(t *testing.T) {
			conn := config.NewConn(t)
			setupInfoTable(t, conn, config.DDL.CreateInfoParent, "conntest_info_parent")
			setupView(t, conn, config.DDL.CreateInfoView, "conntest_info_view")
			ctx := t.Context()

			ok, err := conn.ColumnExists(ctx, fold("conntest_info_view"), fold("id1"))
			require.NoError(t, err)
			assert.True(t, ok)

			ok, err = conn.ColumnExists(ctx, fold("conntest_info_view"), fold("no_such_column"))
			require.NoError(t, err)
			assert.False(t, ok)
		})

		t.Run("PrimaryKey_OnView", func(t *testing.T) {
			conn := config.NewConn(t)
			setupInfoTable(t, conn, config.DDL.CreateInfoParent, "conntest_info_parent")
			setupView(t, conn, config.DDL.CreateInfoView, "conntest_info_view")
			ctx := t.Context()

			_, err := conn.PrimaryKey(ctx, fold("conntest_info_view"))
			assert.ErrorIs(t, err, sql.ErrNoRows,
				"PrimaryKey targets tables only; a view name must error")
		})

		t.Run("ForeignKeys_OnView", func(t *testing.T) {
			conn := config.NewConn(t)
			setupInfoTable(t, conn, config.DDL.CreateInfoParent, "conntest_info_parent")
			setupView(t, conn, config.DDL.CreateInfoView, "conntest_info_view")
			ctx := t.Context()

			_, err := conn.ForeignKeys(ctx, fold("conntest_info_view"))
			assert.ErrorIs(t, err, sql.ErrNoRows,
				"ForeignKeys targets tables only; a view name must error")
		})
	}

	if feat.SupportsRoutines {
		t.Run("Routines_NoFilter", func(t *testing.T) {
			conn := config.NewConn(t)
			ctx := t.Context()
			// Just check the call works and returns []string.
			// Schemas without routines return empty, which is fine.
			_, err := conn.Routines(ctx)
			require.NoError(t, err)
		})

		t.Run("RoutineExists_NameOnly_NotFound", func(t *testing.T) {
			conn := config.NewConn(t)
			ctx := t.Context()
			ok, err := conn.RoutineExists(ctx, fold("conntest_no_such_routine"))
			require.NoError(t, err)
			assert.False(t, ok)
		})

		t.Run("RoutineExists_Signature_NotFound", func(t *testing.T) {
			conn := config.NewConn(t)
			ctx := t.Context()
			ok, err := conn.RoutineExists(ctx, fold("conntest_no_such_routine")+"()")
			require.NoError(t, err)
			assert.False(t, ok)
		})
	} else {
		t.Run("Routines_Unsupported", func(t *testing.T) {
			conn := config.NewConn(t)
			ctx := t.Context()
			_, err := conn.Routines(ctx)
			assert.ErrorIs(t, err, errors.ErrUnsupported)

			_, err = conn.RoutineExists(ctx, "anything")
			assert.ErrorIs(t, err, errors.ErrUnsupported)
		})
	}
}

// setupInfoTable is the Information-test variant of setupTable. The
// cleanup uses context.WithoutCancel so the DROP still runs after the
// test's own context has been cancelled while still propagating any
// values (deadlines, traces) from the parent.
func setupInfoTable(t *testing.T, conn sqldb.Connection, createDDL, tableName string) {
	t.Helper()
	ctx := t.Context()
	_ = conn.Exec(ctx, /*sql*/ `DROP TABLE IF EXISTS `+tableName)
	err := conn.Exec(ctx, createDDL)
	require.NoError(t, err, "creating table %s", tableName)
	cleanupCtx := context.WithoutCancel(ctx)
	t.Cleanup(func() {
		_ = conn.Exec(cleanupCtx, /*sql*/ `DROP TABLE IF EXISTS `+tableName)
	})
}

// setupView is the view counterpart to setupInfoTable.
func setupView(t *testing.T, conn sqldb.Connection, createDDL, viewName string) {
	t.Helper()
	ctx := t.Context()
	_ = conn.Exec(ctx, /*sql*/ `DROP VIEW IF EXISTS `+viewName)
	err := conn.Exec(ctx, createDDL)
	require.NoError(t, err, "creating view %s", viewName)
	cleanupCtx := context.WithoutCancel(ctx)
	t.Cleanup(func() {
		_ = conn.Exec(cleanupCtx, /*sql*/ `DROP VIEW IF EXISTS `+viewName)
	})
}

func lowerSlice(s []string) []string {
	out := make([]string, len(s))
	for i, v := range s {
		out[i] = strings.ToLower(v)
	}
	return out
}
