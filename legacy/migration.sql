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
--   4. play_records.song_level_id → chart_id; rating float → bigint
--   5. best_play_records: add username + chart_id columns with composite unique index
--   6. best50_trend: archived (no equivalent in new schema)
--   7. All int PKs/FKs widened from integer → bigint (Go int = int64 on 64-bit)
--   8. All PK columns renamed to `id` (user_id→id, song_id→id, etc.)
--   9. New indexes: upload_token unique, idx_song_difficulty, idx_best_user_chart
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
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'song_levels') THEN
        RAISE EXCEPTION 'Legacy table "song_levels" not found. Is this the correct database?';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'difficulties') THEN
        RAISE EXCEPTION 'Legacy table "difficulties" not found. Is this the correct database?';
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'charts') THEN
        RAISE EXCEPTION 'Table "charts" already exists. Migration may have already been applied.';
    END IF;
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
UPDATE songs SET wiki_id = '__song_' || song_id WHERE wiki_id IS NULL;
ALTER TABLE songs ALTER COLUMN wiki_id SET NOT NULL;

DO $$ BEGIN RAISE NOTICE '[step 2] songs.wiki_id NULLs fixed, NOT NULL enforced.'; END $$;


-- ============================================================================
-- 3. Create charts table from song_levels + difficulties
--    New table uses `id` as PK column (not `chart_id`)
-- ============================================================================
CREATE TABLE charts (
    id             bigint            NOT NULL,
    song_id        bigint            NOT NULL,
    difficulty     varchar(20)       NOT NULL,
    level          double precision  NOT NULL,
    fitting_level  double precision,
    level_design   character varying,
    notes          bigint            NOT NULL
);

INSERT INTO charts (id, song_id, difficulty, level, fitting_level, level_design, notes)
SELECT
    sl.song_level_id  AS id,
    sl.song_id,
    LOWER(d.name)     AS difficulty,
    sl.level,
    sl.fitting_level,
    sl.level_design,
    sl.notes
FROM song_levels sl
JOIN difficulties d ON sl.difficulty_id = d.difficulty_id;

ALTER TABLE charts ADD CONSTRAINT charts_pkey PRIMARY KEY (id);
CREATE UNIQUE INDEX idx_song_difficulty ON charts (song_id, difficulty);

CREATE SEQUENCE charts_id_seq AS bigint START WITH 1 INCREMENT BY 1 NO MINVALUE NO MAXVALUE CACHE 1;
SELECT setval('charts_id_seq', COALESCE((SELECT MAX(id) FROM charts), 0));
ALTER TABLE charts ALTER COLUMN id SET DEFAULT nextval('charts_id_seq');
ALTER SEQUENCE charts_id_seq OWNED BY charts.id;

DO $$ BEGIN RAISE NOTICE '[step 3] charts table created. % rows migrated.', (SELECT COUNT(*) FROM charts); END $$;


-- ============================================================================
-- 4. Migrate play_records: rename FK column + convert rating
-- ============================================================================
ALTER TABLE play_records DROP CONSTRAINT IF EXISTS play_records_song_level_id_fkey;
ALTER TABLE play_records DROP CONSTRAINT IF EXISTS play_records_username_fkey;

-- Rename song_level_id → chart_id (FK to charts)
ALTER TABLE play_records RENAME COLUMN song_level_id TO chart_id;

-- Convert rating: legacy stores ×100 integer values in a float column
ALTER TABLE play_records ADD COLUMN rating_new bigint;
UPDATE play_records SET rating_new = CAST(ROUND(rating) AS BIGINT);
ALTER TABLE play_records DROP COLUMN rating;
ALTER TABLE play_records RENAME COLUMN rating_new TO rating;
ALTER TABLE play_records ALTER COLUMN rating SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_play_records_username ON play_records (username);
CREATE INDEX IF NOT EXISTS idx_pr_user_chart ON play_records (chart_id);

DO $$ BEGIN RAISE NOTICE '[step 4] play_records migrated: song_level_id → chart_id, rating float → bigint.'; END $$;


-- ============================================================================
-- 5. Migrate best_play_records: add username + chart_id columns
-- ============================================================================
ALTER TABLE best_play_records DROP CONSTRAINT IF EXISTS best_play_records_play_record_id_fkey;

ALTER TABLE best_play_records ADD COLUMN username character varying;
ALTER TABLE best_play_records ADD COLUMN chart_id bigint;

-- Populate from play_records (play_record_id is still PK name at this point)
UPDATE best_play_records bpr
SET username = pr.username, chart_id = pr.chart_id
FROM play_records pr
WHERE bpr.play_record_id = pr.play_record_id;

DELETE FROM best_play_records WHERE username IS NULL OR chart_id IS NULL;

ALTER TABLE best_play_records ALTER COLUMN username SET NOT NULL;
ALTER TABLE best_play_records ALTER COLUMN chart_id SET NOT NULL;

CREATE UNIQUE INDEX idx_best_user_chart ON best_play_records (username, chart_id);
CREATE INDEX IF NOT EXISTS idx_best_play_records_play_record_id ON best_play_records (play_record_id);

DO $$ BEGIN RAISE NOTICE '[step 5] best_play_records migrated: added username + chart_id columns.'; END $$;


-- ============================================================================
-- 6. Widen integer columns to bigint (Go int = int64 on 64-bit systems)
-- ============================================================================
ALTER TABLE prober_users ALTER COLUMN user_id TYPE bigint;
ALTER TABLE songs ALTER COLUMN song_id TYPE bigint;
ALTER TABLE play_records ALTER COLUMN play_record_id TYPE bigint;
ALTER TABLE play_records ALTER COLUMN chart_id TYPE bigint;
ALTER TABLE play_records ALTER COLUMN score TYPE bigint;
ALTER TABLE best_play_records ALTER COLUMN best_record_id TYPE bigint;
ALTER TABLE best_play_records ALTER COLUMN play_record_id TYPE bigint;

ALTER SEQUENCE prober_users_user_id_seq AS bigint;
ALTER SEQUENCE songs_song_id_seq AS bigint;
ALTER SEQUENCE play_records_play_record_id_seq AS bigint;
ALTER SEQUENCE best_play_records_best_record_id_seq AS bigint;

DO $$ BEGIN RAISE NOTICE '[step 6] All integer PK/FK columns widened to bigint.'; END $$;


-- ============================================================================
-- 6b. Convert qq_number from integer to text (supports QQ email format)
-- ============================================================================
ALTER TABLE prober_users ALTER COLUMN qq_number TYPE text USING qq_number::text;
ALTER TABLE prober_users RENAME COLUMN qq_number TO qq_account;

DO $$ BEGIN RAISE NOTICE '[step 6b] prober_users.qq_number → qq_account (text).'; END $$;


-- ============================================================================
-- 7. Rename PK columns to `id` (Go convention: ID field)
-- ============================================================================

-- prober_users: user_id → id
ALTER TABLE prober_users RENAME COLUMN user_id TO id;
ALTER SEQUENCE prober_users_user_id_seq RENAME TO prober_users_id_seq;

-- songs: song_id → id
ALTER TABLE songs RENAME COLUMN song_id TO id;
ALTER SEQUENCE songs_song_id_seq RENAME TO songs_id_seq;

-- play_records: play_record_id → id
ALTER TABLE play_records RENAME COLUMN play_record_id TO id;
ALTER SEQUENCE play_records_play_record_id_seq RENAME TO play_records_id_seq;

-- best_play_records: best_record_id → id
ALTER TABLE best_play_records RENAME COLUMN best_record_id TO id;
ALTER SEQUENCE best_play_records_best_record_id_seq RENAME TO best_play_records_id_seq;

-- charts: PK already created as `id` in step 3 — no rename needed.

DO $$ BEGIN RAISE NOTICE '[step 7] All PK columns renamed to id.'; END $$;


-- ============================================================================
-- 8. Add missing indexes on prober_users
-- ============================================================================
CREATE UNIQUE INDEX IF NOT EXISTS idx_prober_users_upload_token ON prober_users (upload_token);

DO $$ BEGIN RAISE NOTICE '[step 8] prober_users indexes created.'; END $$;


-- ============================================================================
-- 9. Drop legacy tables
-- ============================================================================
DROP TABLE IF EXISTS song_levels CASCADE;
DROP TABLE IF EXISTS difficulties CASCADE;

DO $$ BEGIN RAISE NOTICE '[step 9] Legacy tables (song_levels, difficulties) dropped.'; END $$;


-- ============================================================================
-- 10. Post-migration validation
-- ============================================================================
DO $$
DECLARE
    v_charts_count       bigint;
    v_play_records_count bigint;
    v_best_records_count bigint;
    v_users_count        bigint;
    v_songs_count        bigint;
    v_null_wiki_ids      bigint;
    v_null_usernames     bigint;
    v_orphan_charts      bigint;
    v_orphan_records     bigint;
BEGIN
    SELECT COUNT(*) INTO v_users_count FROM prober_users;
    SELECT COUNT(*) INTO v_songs_count FROM songs;
    SELECT COUNT(*) INTO v_charts_count FROM charts;
    SELECT COUNT(*) INTO v_play_records_count FROM play_records;
    SELECT COUNT(*) INTO v_best_records_count FROM best_play_records;

    SELECT COUNT(*) INTO v_null_wiki_ids FROM songs WHERE wiki_id IS NULL;
    IF v_null_wiki_ids > 0 THEN
        RAISE EXCEPTION 'Validation failed: % songs still have NULL wiki_id', v_null_wiki_ids;
    END IF;

    SELECT COUNT(*) INTO v_null_usernames FROM best_play_records WHERE username IS NULL;
    IF v_null_usernames > 0 THEN
        RAISE EXCEPTION 'Validation failed: % best_play_records have NULL username', v_null_usernames;
    END IF;

    -- Orphan checks use new column names: songs.id, charts.id
    SELECT COUNT(*) INTO v_orphan_charts
    FROM charts c LEFT JOIN songs s ON c.song_id = s.id
    WHERE s.id IS NULL;
    IF v_orphan_charts > 0 THEN
        RAISE WARNING 'Found % charts referencing non-existent songs', v_orphan_charts;
    END IF;

    SELECT COUNT(*) INTO v_orphan_records
    FROM play_records pr LEFT JOIN charts c ON pr.chart_id = c.id
    WHERE c.id IS NULL;
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
