package pqconn

const DropAllTablesQuery = /*sql*/ `
DO $$
DECLARE
	r RECORD;
BEGIN
	FOR r IN (SELECT * FROM pg_tables WHERE schemaname NOT IN ('pg_catalog', 'information_schema')) LOOP
		EXECUTE 'DROP TABLE IF EXISTS ' || quote_ident(r.schemaname) || '.' || quote_ident(r.tablename) || ' CASCADE';
	END LOOP;
END $$
`

const DropAllTablesInCurrentSchemaQuery = /*sql*/ `
DO $$
DECLARE
	r RECORD;
BEGIN
	FOR r IN (SELECT * FROM pg_tables WHERE schemaname = current_schema()) LOOP
		EXECUTE 'DROP TABLE IF EXISTS ' || quote_ident(r.tablename) || ' CASCADE';
	END LOOP;
END $$
`

const DropAllTablesInSchemaQuery = /*sql*/ `
DO $$
DECLARE
	r RECORD;
BEGIN
	FOR r IN (SELECT * FROM pg_tables WHERE schemaname = $1) LOOP
		EXECUTE 'DROP TABLE IF EXISTS ' || quote_ident(r.schemaname) || '.' || quote_ident(r.tablename) || ' CASCADE';
	END LOOP;
END $$
`
