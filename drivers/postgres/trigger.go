package postgres

import (
	"database/sql"

	"github.com/cgalvisleon/josefina/jdb"
)

func TriggerRecords(db *sql.DB) error {
	sql := jdb.SQLUnQuote(`
	CREATE SCHEMA IF NOT EXISTS core;

	CREATE TABLE IF NOT EXISTS core.tables (
	created_at TIMESTAMP,
	updated_at TIMESTAMP,
	table_schema VARCHAR(250),
	table_name VARCHAR(250),
	total BIGINT,
	CONSTRAINT pk_tables PRIMARY KEY (table_schema, table_name));	

	CREATE OR REPLACE FUNCTION core.after_insert_tables()
	RETURNS TRIGGER AS $$	
	BEGIN		
		INSERT INTO core.tables (created_at, updated_at, table_schema, table_name, total)
		VALUES (now(), now(), TG_TABLE_SCHEMA, TG_TABLE_NAME, 1)
		ON CONFLICT (table_schema, table_name) DO UPDATE
		SET total = core.tables.total + 1,
				updated_at = now();

		RETURN NEW;
	END;
	$$ LANGUAGE plpgsql;

	CREATE OR REPLACE FUNCTION core.after_delete_tables()
	RETURNS TRIGGER AS $$	
	BEGIN
		UPDATE core.tables
		SET total = core.tables.total - 1,
				updated_at = now()
		WHERE table_schema = TG_TABLE_SCHEMA
		AND table_name = TG_TABLE_NAME;

		RETURN OLD;
	END;
	$$ LANGUAGE plpgsql;

	CREATE TABLE IF NOT EXISTS core.records (
	created_at TIMESTAMP,
	updated_at TIMESTAMP,
	table_schema VARCHAR(250),
	table_name VARCHAR(250),
	$1 VARCHAR(80),
	CONSTRAINT pk_records PRIMARY KEY (table_schema, table_name, $1));

	CREATE TABLE IF NOT EXISTS core.recyclings (
	created_at TIMESTAMP,
	table_schema VARCHAR(250),
	table_name VARCHAR(250),
	$1 VARCHAR(80),
	CONSTRAINT pk_recyclings PRIMARY KEY (table_schema, table_name, $1));

	DROP TRIGGER IF EXISTS RECORDS_SET ON core.records CASCADE;
	CREATE TRIGGER RECORDS_SET
	AFTER INSERT ON core.records
	FOR EACH ROW
	EXECUTE FUNCTION core.after_insert_tables();

	DROP TRIGGER IF EXISTS RECORDS_DELETE ON core.records CASCADE;
	CREATE TRIGGER RECORDS_DELETE
	AFTER DELETE ON core.records
	FOR EACH ROW
	EXECUTE FUNCTION core.after_delete_tables();

	CREATE OR REPLACE FUNCTION core.after_records()
	RETURNS TRIGGER AS $$
	DECLARE
		vnew JSONB;
 		vold JSONB;
	BEGIN
		vnew = to_jsonb(NEW);
		vold = to_jsonb(OLD);

		IF TG_OP = 'INSERT' AND (vnew ? '$1') THEN
			INSERT INTO core.records (created_at, updated_at, table_schema, table_name, $1)
			VALUES (now(), now(), TG_TABLE_SCHEMA, TG_TABLE_NAME, NEW.$1);
		END IF;
		
		IF TG_OP = 'UPDATE' AND (vnew ? '$1') THEN
			UPDATE core.records
			SET updated_at = now()
			WHERE table_schema = TG_TABLE_SCHEMA
			AND table_name = TG_TABLE_NAME
			AND $1 = NEW.$1;
		END IF;

		IF TG_OP = 'UPDATE' AND (vnew ? '$1') AND (vnew ? '$2') AND vnew->>'$2' != vold->>'$2' AND vnew->>'$2' = '$3' THEN
			INSERT INTO core.recyclings (created_at, table_schema, table_name, $1)
			VALUES (now(), TG_TABLE_SCHEMA, TG_TABLE_NAME, NEW.$1);
		END IF;
		
		IF TG_OP = 'UPDATE' AND (vnew ? '$1') AND (vnew ? '$2') AND vnew->>'$2' != vold->>'$2' AND vnew->>'$2' != '$3' THEN
			DELETE FROM core.recyclings
			WHERE table_schema = TG_TABLE_SCHEMA
			AND table_name = TG_TABLE_NAME
			AND $1 = NEW.$1;
		END IF;
		
		IF TG_OP = 'DELETE' AND (vold ? '$1') THEN
			DELETE FROM core.records
			WHERE table_schema = TG_TABLE_SCHEMA
			AND table_name = TG_TABLE_NAME
			AND $1 = OLD.$1;

			DELETE FROM core.recyclings
			WHERE table_schema = TG_TABLE_SCHEMA
			AND table_name = TG_TABLE_NAME
			AND $1 = OLD.$1;
		END IF;

		IF TG_OP = 'DELETE' THEN
      RETURN OLD;
    ELSE
      RETURN NEW;
    END IF;
	END;
	$$ LANGUAGE plpgsql;
	`, jdb.INDEX, jdb.SOURCE, jdb.ForDelete)

	_, err := db.Exec(sql)
	if err != nil {
		return err
	}

	return nil
}
