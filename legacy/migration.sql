-- ============================================================================
-- Paradigm: Reboot Prober — Legacy → V2 Database Migration Script
-- ============================================================================
-- Source: Legacy schema (Python/FastAPI, PostgreSQL)
-- Target: New Go/Gin schema (PostgreSQL)
--
-- Key changes:
--   1. songs.wiki_id: nullable → NOT NULL (generate placeholders for NULLs)
--   2. difficulties table → eliminated; difficulty stored inline as varchar(20)
--   3. song_levels → charts (rename table + PK column)
--   4. play_records.song_level_id → chart_id; rating float → int (×100)
--   5. best_play_records: add username + chart_id columns with composite unique index
--   6. best50_trend: archived (no equivalent in new schema)
--   7. New indexes: upload_token unique, idx_song_difficulty, idx_best_user_chart
--
-- Prerequisites:
--   - PostgreSQL 14+
--   - Backup your database before running this script!
--   - The script runs in a single transaction; any error will rollback all changes
--
-- Usage:
--   psql -h <host> -U <user> -d <dbname> -f migration.sql
--   OR via the Go wrapper: go run cmd/migrate/main.go
-- ============================================================================

BEGIN;

-- ============================================================================
-- 0. Pre-flight checks
-- ============================================================================
DO $$
BEGIN
    -- Verify legacy tables exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'song_levels') THEN
        RAISE EXCEPTION 'Legacy table "song_levels" not found. Is this the correct database?';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'difficulties') THEN
        RAISE EXCEPTION 'Legacy table "difficulties" not found. Is this the correct database?';
    END IF;

    -- Verify new table does NOT already exist (prevent double-run)
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'charts') THEN
        RAISE EXCEPTION 'Table "charts" already exists. Migration may have already been applied.';
    END IF;

    -- Verify difficulty names are valid
    IF EXISTS (
        SELECT 1 FROM difficulties
        WHERE LOWER(name) NOT IN ('detected', 'invaded', 'massive', 'reboot')
    ) THEN
        RAISE EXCEPTION 'Unknown difficulty names found in "difficulties" table. Expected: detected, invaded, massive, reboot.';
    END IF;

    RAISE NOTICE '[pre-flight] All checks passed.';
END $$;


-- ============================================================================
-- 1. Archive best50_trend (no equivalent in new schema)
-- ============================================================================
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'best50_trend') THEN
        -- Drop the FK constraint first
        IF EXISTS (
            SELECT 1 FROM information_schema.table_constraints
            WHERE constraint_name = 'best50_trend_username_fkey'
              AND table_name = 'best50_trend'
        ) THEN
            ALTER TABLE best50_trend DROP CONSTRAINT best50_trend_username_fkey;
        END IF;

        ALTER TABLE best50_trend RENAME TO _legacy_best50_trend;
        RAISE NOTICE '[step 1] Archived best50_trend → _legacy_best50_trend';
    ELSE
        RAISE NOTICE '[step 1] best50_trend not found, skipping archive.';
    END IF;
END $$;


-- ============================================================================
-- 2. Fix songs.wiki_id NULLs and add NOT NULL constraint
-- ============================================================================

-- Generate placeholder wiki_id for rows with NULL (format: __song_<song_id>)
UPDATE songs
SET wiki_id = '__song_' || song_id
WHERE wiki_id IS NULL;

-- Now enforce NOT NULL
ALTER TABLE songs ALTER COLUMN wiki_id SET NOT NULL;

DO $$ BEGIN RAISE NOTICE '[step 2] songs.wiki_id NULLs fixed, NOT NULL enforced.'; END $$;


-- ============================================================================
-- 3. Create charts table from song_levels + difficulties
-- ============================================================================

-- Create the new charts table with data from legacy tables
CREATE TABLE charts (
    chart_id       integer       NOT NULL,
    song_id        integer       NOT NULL,
    difficulty     varchar(20)   NOT NULL,
    level          double precision NOT NULL,
    fitting_level  double precision,
    level_design   character varying,
    notes          integer       NOT NULL
);

INSERT INTO charts (chart_id, song_id, difficulty, level, fitting_level, level_design, notes)
SELECT
    sl.song_level_id  AS chart_id,
    sl.song_id,
    LOWER(d.name)     AS difficulty,
    sl.level,
    sl.fitting_level,
    sl.level_design,
    sl.notes
FROM song_levels sl
JOIN difficulties d ON sl.difficulty_id = d.difficulty_id;

-- Primary key
ALTER TABLE charts ADD CONSTRAINT charts_pkey PRIMARY KEY (chart_id);

-- Composite unique index: one chart per difficulty per song
CREATE UNIQUE INDEX idx_song_difficulty ON charts (song_id, difficulty);

-- Sequence for auto-increment (continue from max chart_id)
CREATE SEQUENCE charts_chart_id_seq AS integer START WITH 1 INCREMENT BY 1 NO MINVALUE NO MAXVALUE CACHE 1;
SELECT setval('charts_chart_id_seq', COALESCE((SELECT MAX(chart_id) FROM charts), 0));
ALTER TABLE charts ALTER COLUMN chart_id SET DEFAULT nextval('charts_chart_id_seq');
ALTER SEQUENCE charts_chart_id_seq OWNED BY charts.chart_id;

DO $$ BEGIN RAISE NOTICE '[step 3] charts table created from song_levels + difficulties. % rows migrated.', (SELECT COUNT(*) FROM charts); END $$;


-- ============================================================================
-- 4. Migrate play_records: rename column + convert rating
-- ============================================================================

-- Drop old foreign key constraints
ALTER TABLE play_records DROP CONSTRAINT IF EXISTS play_records_song_level_id_fkey;
ALTER TABLE play_records DROP CONSTRAINT IF EXISTS play_records_username_fkey;

-- Rename song_level_id → chart_id
ALTER TABLE play_records RENAME COLUMN song_level_id TO chart_id;

-- Convert rating from double precision to integer
-- Legacy DB already stores rating as ×100 integer values in a float column
-- (e.g. 15486.0 means rating 154.86), so we just need to cast, not multiply.
-- ROUND handles any floating-point imprecision (e.g. 15485.9999999 → 15486).
ALTER TABLE play_records ADD COLUMN rating_new integer;
UPDATE play_records SET rating_new = CAST(ROUND(rating) AS INTEGER);
ALTER TABLE play_records DROP COLUMN rating;
ALTER TABLE play_records RENAME COLUMN rating_new TO rating;
ALTER TABLE play_records ALTER COLUMN rating SET NOT NULL;

-- Add indexes matching the new schema
CREATE INDEX IF NOT EXISTS idx_play_records_username ON play_records (username);
CREATE INDEX IF NOT EXISTS idx_pr_user_chart ON play_records (chart_id);

DO $$ BEGIN RAISE NOTICE '[step 4] play_records migrated: song_level_id → chart_id, rating float → int.'; END $$;


-- ============================================================================
-- 5. Migrate best_play_records: add username + chart_id columns
-- ============================================================================

-- Drop old FK constraint
ALTER TABLE best_play_records DROP CONSTRAINT IF EXISTS best_play_records_play_record_id_fkey;

-- Add new columns
ALTER TABLE best_play_records ADD COLUMN username character varying;
ALTER TABLE best_play_records ADD COLUMN chart_id integer;

-- Populate from play_records via the existing play_record_id reference
UPDATE best_play_records bpr
SET
    username = pr.username,
    chart_id = pr.chart_id
FROM play_records pr
WHERE bpr.play_record_id = pr.play_record_id;

-- Remove any orphan rows where play_record_id doesn't match (shouldn't happen, but be safe)
DELETE FROM best_play_records WHERE username IS NULL OR chart_id IS NULL;

-- Enforce NOT NULL
ALTER TABLE best_play_records ALTER COLUMN username SET NOT NULL;
ALTER TABLE best_play_records ALTER COLUMN chart_id SET NOT NULL;

-- Add composite unique index
CREATE UNIQUE INDEX idx_best_user_chart ON best_play_records (username, chart_id);

-- Add index on play_record_id
CREATE INDEX IF NOT EXISTS idx_best_play_records_play_record_id ON best_play_records (play_record_id);

DO $$ BEGIN RAISE NOTICE '[step 5] best_play_records migrated: added username + chart_id columns.'; END $$;


-- ============================================================================
-- 6. Add missing indexes on prober_users
-- ============================================================================

-- upload_token unique index (new schema requires it)
CREATE UNIQUE INDEX IF NOT EXISTS idx_prober_users_upload_token ON prober_users (upload_token);

DO $$ BEGIN RAISE NOTICE '[step 6] prober_users indexes created.'; END $$;


-- ============================================================================
-- 7. Drop legacy tables and sequences
-- ============================================================================

-- Drop song_levels (replaced by charts)
-- CASCADE also drops the owned sequence (song_levels_song_level_id_seq)
DROP TABLE IF EXISTS song_levels CASCADE;

-- Drop difficulties lookup table (no longer needed)
-- CASCADE also drops the owned sequence (difficulties_difficulty_id_seq)
DROP TABLE IF EXISTS difficulties CASCADE;

-- Note: best50_trend_b50_trend_id_seq is kept because the archived
-- _legacy_best50_trend table still depends on it.

DO $$ BEGIN RAISE NOTICE '[step 7] Legacy tables (song_levels, difficulties) dropped.'; END $$;


-- ============================================================================
-- 8. Post-migration validation
-- ============================================================================
DO $$
DECLARE
    v_charts_count       integer;
    v_play_records_count integer;
    v_best_records_count integer;
    v_users_count        integer;
    v_songs_count        integer;
    v_null_wiki_ids      integer;
    v_null_usernames     integer;
    v_orphan_charts      integer;
    v_orphan_records     integer;
BEGIN
    SELECT COUNT(*) INTO v_users_count FROM prober_users;
    SELECT COUNT(*) INTO v_songs_count FROM songs;
    SELECT COUNT(*) INTO v_charts_count FROM charts;
    SELECT COUNT(*) INTO v_play_records_count FROM play_records;
    SELECT COUNT(*) INTO v_best_records_count FROM best_play_records;

    -- Check for NULL wiki_ids (should be 0)
    SELECT COUNT(*) INTO v_null_wiki_ids FROM songs WHERE wiki_id IS NULL;
    IF v_null_wiki_ids > 0 THEN
        RAISE EXCEPTION 'Validation failed: % songs still have NULL wiki_id', v_null_wiki_ids;
    END IF;

    -- Check for NULL usernames in best_play_records (should be 0)
    SELECT COUNT(*) INTO v_null_usernames FROM best_play_records WHERE username IS NULL;
    IF v_null_usernames > 0 THEN
        RAISE EXCEPTION 'Validation failed: % best_play_records have NULL username', v_null_usernames;
    END IF;

    -- Check for orphan charts (chart references non-existent song)
    SELECT COUNT(*) INTO v_orphan_charts
    FROM charts c
    LEFT JOIN songs s ON c.song_id = s.song_id
    WHERE s.song_id IS NULL;
    IF v_orphan_charts > 0 THEN
        RAISE WARNING 'Found % charts referencing non-existent songs', v_orphan_charts;
    END IF;

    -- Check for orphan play_records (record references non-existent chart)
    SELECT COUNT(*) INTO v_orphan_records
    FROM play_records pr
    LEFT JOIN charts c ON pr.chart_id = c.chart_id
    WHERE c.chart_id IS NULL;
    IF v_orphan_records > 0 THEN
        RAISE WARNING 'Found % play_records referencing non-existent charts', v_orphan_records;
    END IF;

    RAISE NOTICE '============================================================';
    RAISE NOTICE 'Migration validation passed!';
    RAISE NOTICE '  prober_users:     % rows', v_users_count;
    RAISE NOTICE '  songs:            % rows', v_songs_count;
    RAISE NOTICE '  charts:           % rows (from song_levels)', v_charts_count;
    RAISE NOTICE '  play_records:     % rows', v_play_records_count;
    RAISE NOTICE '  best_play_records: % rows', v_best_records_count;
    RAISE NOTICE '============================================================';
END $$;


COMMIT;

-- ============================================================================
-- Migration complete!
-- After running this script, start the Go server to let GORM AutoMigrate
-- apply any remaining minor schema adjustments (e.g., column type tweaks).
-- ============================================================================