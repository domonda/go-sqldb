package pqconn

// DropAllTablesQuery drops all tables in all user schemas
// (everything except pg_catalog and information_schema).
// Uses CASCADE to also drop dependent objects like
// foreign key constraints, indexes, and triggers.
// Uses IF EXISTS so that tables already dropped via CASCADE
// from a previous iteration are silently skipped.
//
// Order: must be executed BEFORE [DropAllTypesQuery]
// when both are used, because DROP TYPE cannot remove composite types
// that PostgreSQL automatically creates for tables.
// Use [DropAllQuery] instead to drop tables and types
// in the correct order within a single query.
//
// Source: pg_tables is a view over pg_class filtered to relkind = 'r' (ordinary table).
// See: https://www.postgresql.org/docs/current/view-pg-tables.html
const DropAllTablesQuery = /*sql*/ `
DO $$
DECLARE
	r RECORD;
BEGIN
	FOR r IN (
		SELECT schemaname, tablename
		FROM pg_tables
		WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
	) LOOP
		EXECUTE 'DROP TABLE IF EXISTS '
			|| quote_ident(r.schemaname) || '.' || quote_ident(r.tablename)
			|| ' CASCADE';
	END LOOP;
END $$
`

// DropAllTypesQuery drops all user-defined types
// (enums, domains, standalone composite types, range types, etc.)
// in all user schemas (everything except pg_catalog and information_schema).
// Uses CASCADE so that dependent objects like auto-created array types
// are removed automatically.
// Uses IF EXISTS so that types already dropped via CASCADE
// from a previous iteration are silently skipped.
//
// Order: must be executed AFTER [DropAllTablesQuery]
// when both are used, because DROP TYPE cannot remove composite types
// that PostgreSQL automatically creates for tables.
// Use [DropAllQuery] instead to drop tables and types
// in the correct order within a single query.
//
// The query excludes two categories of types from pg_type:
//
//  1. Composite types automatically created for tables, views,
//     materialized views, and partitioned tables.
//     Identified by typrelid referencing a pg_class entry with
//     relkind in ('r','v','m','p'). Standalone composite types
//     from CREATE TYPE have relkind = 'c' and are NOT excluded.
//
//  2. Array types (typcategory = 'A') that PostgreSQL automatically
//     creates for every type (e.g. _myenum for myenum).
//     These are internally managed and dropped automatically
//     when their element type is dropped via CASCADE.
//
// Source: pg_type catalogs data types; pg_class.relkind identifies relation kinds.
// See: https://www.postgresql.org/docs/current/catalog-pg-type.html
// See: https://www.postgresql.org/docs/current/catalog-pg-class.html
const DropAllTypesQuery = /*sql*/ `
DO $$
DECLARE
	r RECORD;
BEGIN
	FOR r IN (
		SELECT n.nspname, t.typname
		FROM pg_type t
		JOIN pg_namespace n ON t.typnamespace = n.oid
		WHERE n.nspname NOT IN ('pg_catalog', 'information_schema')
		AND t.typcategory != 'A'
		AND NOT EXISTS (
			SELECT 1 FROM pg_class c
			WHERE c.oid = t.typrelid
			AND c.relkind IN ('r', 'v', 'm', 'p')
		)
	) LOOP
		EXECUTE 'DROP TYPE IF EXISTS '
			|| quote_ident(r.nspname) || '.' || quote_ident(r.typname)
			|| ' CASCADE';
	END LOOP;
END $$
`

// DropAllQuery drops all tables first, then all user-defined types
// in all user schemas.
// Concatenates [DropAllTablesQuery] and [DropAllTypesQuery]
// in the correct order.
const DropAllQuery = DropAllTablesQuery + ";" + DropAllTypesQuery

// DropAllTablesInCurrentSchemaQuery drops all tables
// in the current schema (usually "public").
// Uses CASCADE and IF EXISTS (see [DropAllTablesQuery]).
//
// Order: must be executed BEFORE [DropAllTypesInCurrentSchemaQuery]
// when both are used. Use [DropAllInCurrentSchemaQuery] instead
// to drop tables and types in the correct order within a single query.
const DropAllTablesInCurrentSchemaQuery = /*sql*/ `
DO $$
DECLARE
	r RECORD;
BEGIN
	FOR r IN (
		SELECT tablename
		FROM pg_tables
		WHERE schemaname = current_schema()
	) LOOP
		EXECUTE 'DROP TABLE IF EXISTS '
			|| quote_ident(r.tablename)
			|| ' CASCADE';
	END LOOP;
END $$
`

// DropAllTypesInCurrentSchemaQuery drops all user-defined types
// (enums, domains, standalone composite types, range types, etc.)
// in the current schema (usually "public").
// Uses CASCADE and IF EXISTS (see [DropAllTypesQuery]).
//
// Order: must be executed AFTER [DropAllTablesInCurrentSchemaQuery]
// when both are used. Use [DropAllInCurrentSchemaQuery] instead
// to drop tables and types in the correct order within a single query.
//
// See [DropAllTypesQuery] for details on excluded type categories.
const DropAllTypesInCurrentSchemaQuery = /*sql*/ `
DO $$
DECLARE
	r RECORD;
BEGIN
	FOR r IN (
		SELECT t.typname
		FROM pg_type t
		JOIN pg_namespace n ON t.typnamespace = n.oid
		WHERE n.nspname = current_schema()
		-- Exclude auto-created array types (e.g. _myenum for myenum).
		-- They are dropped automatically when their element type
		-- is dropped via CASCADE.
		AND t.typcategory != 'A'
		-- Exclude composite types automatically created for
		-- tables, views, materialized views, and partitioned tables.
		-- DROP TYPE cannot remove these; use DROP TABLE instead.
		AND NOT EXISTS (
			SELECT 1 FROM pg_class c
			WHERE c.oid = t.typrelid
			AND c.relkind IN (
				'r', -- ordinary table
				'v', -- view
				'm', -- materialized view
				'p'  -- partitioned table
			)
		)
	) LOOP
		EXECUTE 'DROP TYPE IF EXISTS ' || quote_ident(r.typname) || ' CASCADE';
	END LOOP;
END $$
`

// DropAllInCurrentSchemaQuery drops all tables first, then all
// user-defined types in the current schema.
// Concatenates [DropAllTablesInCurrentSchemaQuery] and
// [DropAllTypesInCurrentSchemaQuery] in the correct order.
const DropAllInCurrentSchemaQuery = DropAllTablesInCurrentSchemaQuery + ";" + DropAllTypesInCurrentSchemaQuery
