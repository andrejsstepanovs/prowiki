-- Remove seeded prompt stages added in 000004
DELETE FROM prompt_registry WHERE stage = 'level_4_edge_case';
DELETE FROM prompt_registry WHERE stage = 'intersection_synthesis';

-- NOTE: SQLite supports DROP COLUMN only from version 3.35.0 (released 2021-03-12).
-- If running SQLite < 3.35, these statements will fail; in that case a table-rebuild
-- approach is required. The Go modernc.org/sqlite and mattn/go-sqlite3 drivers
-- bundled with recent Go modules ship SQLite >= 3.40, so the statements below are safe.
ALTER TABLE projects DROP COLUMN fs_location;
ALTER TABLE job_queue DROP COLUMN payload;
