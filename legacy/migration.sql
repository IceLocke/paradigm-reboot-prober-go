-- ============================================================================
-- Paradigm: Reboot Prober — Legacy → V2 Database Migration Script
-- ============================================================================
-- Source: Legacy schema (Python/FastAPI, PostgreSQL)
-- Target: New Go/Gin schema (PostgreSQL 14+)
--
-- Key schema changes:
--   1. songs.wiki_id: nullable → NOT NULL (generate placeholders for NULLs)
--   2. difficulties table → eliminated; difficulty stored inline as varchar(20)
--   3. song_levels → charts (rename table + PK column)
--   4. play_records.song_level_id → chart_id; rating float → bigint;
--      record_time: timestamp → timestamptz
--   5. best_play_records: add username + chart_id columns with composite unique index
--   6. best50_trend: archived (no equivalent in new schema)
--   7. All int PKs/FKs widened from integer → bigint (Go int = int64 on 64-bit)
--   8. All PK columns renamed to `id` (user_id→id, song_id→id, etc.)
--   9. prober_users.qq_number (int) → qq_account (text)
--  10. Every table gets created_at / updated_at / deleted_at (BaseModel)
--      with deleted_at btree indexes (soft-delete). Audit columns are
--      nullable and have no DB-level default: GORM's autoCreateTime /
--      autoUpdateTime tags fill them from Go on every write.
--  11. charts gets override_title / override_artist / override_version /
--      override_cover (SongBaseOverride, all nullable).
--  12. Column types normalized to match GORM's Postgres dialect (text
--      instead of character varying, decimal instead of double precision,
--      bigint instead of integer where applicable).
--  13. Indexes created to match GORM model tags 1:1, so that running
--      AutoMigrate after this script is a no-op:
--        - prober_users_username_key           UNIQUE  (prober_users.username)  [kept from legacy]
--        - idx_prober_users_upload_token       UNIQUE  (prober_users.upload_token)
--        - idx_prober_users_deleted_at         btree   (prober_users.deleted_at)
--        - songs_wiki_id_unique                UNIQUE  (songs.wiki_id)          [kept/renamed from legacy]
--        - idx_songs_b15                       btree   (songs.b15)
--        - idx_songs_deleted_at                btree   (songs.deleted_at)
--        - idx_song_difficulty                 UNIQUE partial (charts.song_id, charts.difficulty) WHERE deleted_at IS NULL
--        - idx_charts_deleted_at               btree   (charts.deleted_at)
--        - idx_play_records_username           btree   (play_records.username)
--        - idx_play_records_rating             btree   (play_records.rating)
--        - idx_pr_user_chart                   btree   (play_records.username, play_records.chart_id)
--        - idx_play_records_deleted_at         btree   (play_records.deleted_at)
--        - idx_best_user_chart                 UNIQUE  (best_play_records.username, best_play_records.chart_id)
--        - idx_best_play_records_play_record_id btree  (best_play_records.play_record_id)
--        - idx_best_play_records_deleted_at    btree   (best_play_records.deleted_at)
--
-- Prerequisites:
--   - PostgreSQL 14+ (partial indexes used in step 12)
--   - Back up your database before running this script!
--   - The script runs in a single transaction; any error will rollback all changes.
--
-- Usage:
--   psql -h <host> -U <user> -d <dbname> -f migration.sql
--   OR via the Go wrapper:  go run cmd/migrate/main.go
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
-- 1. Archive best50_trend (no equivalent in V2 schema)
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
--    (Unique index on (song_id, difficulty) is created later in step 12,
--     after deleted_at is added — it must be a PARTIAL index
--     WHERE deleted_at IS NULL to coexist with soft-delete.)
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

CREATE SEQUENCE charts_id_seq AS bigint START WITH 1 INCREMENT BY 1 NO MINVALUE NO MAXVALUE CACHE 1;
SELECT setval('charts_id_seq', COALESCE((SELECT MAX(id) FROM charts), 0));
ALTER TABLE charts ALTER COLUMN id SET DEFAULT nextval('charts_id_seq');
ALTER SEQUENCE charts_id_seq OWNED BY charts.id;

DO $$ BEGIN RAISE NOTICE '[step 3] charts table created. % rows migrated.', (SELECT COUNT(*) FROM charts); END $$;


-- ============================================================================
-- 4. Migrate play_records: rename FK column, convert rating, convert record_time
-- ============================================================================
ALTER TABLE play_records DROP CONSTRAINT IF EXISTS play_records_song_level_id_fkey;
ALTER TABLE play_records DROP CONSTRAINT IF EXISTS play_records_username_fkey;

-- Rename song_level_id → chart_id (FK semantics → charts.id, no DB-level FK)
ALTER TABLE play_records RENAME COLUMN song_level_id TO chart_id;

-- Convert rating: legacy stores (rating × 100) as integer-valued double precision
-- (e.g. 16546.0 == rating 165.46). Just round to bigint to get the V2 storage
-- form. No formula re-computation needed.
ALTER TABLE play_records ADD COLUMN rating_new bigint;
UPDATE play_records SET rating_new = CAST(ROUND(rating) AS BIGINT);
ALTER TABLE play_records DROP COLUMN rating;
ALTER TABLE play_records RENAME COLUMN rating_new TO rating;
ALTER TABLE play_records ALTER COLUMN rating SET NOT NULL;

-- Convert record_time: timestamp (naive local time) → timestamptz
-- Legacy stored Asia/Shanghai local time in a tz-naive column.
ALTER TABLE play_records
  ALTER COLUMN record_time TYPE timestamptz
  USING record_time AT TIME ZONE 'Asia/Shanghai';

DO $$ BEGIN RAISE NOTICE '[step 4] play_records migrated: song_level_id → chart_id, rating float → bigint, record_time → timestamptz.'; END $$;


-- ============================================================================
-- 5. Migrate best_play_records: add username + chart_id columns, backfill from play_records
-- ============================================================================
ALTER TABLE best_play_records DROP CONSTRAINT IF EXISTS best_play_records_play_record_id_fkey;

ALTER TABLE best_play_records ADD COLUMN username character varying;
ALTER TABLE best_play_records ADD COLUMN chart_id bigint;

-- Backfill (play_record_id is still the column name at this point).
UPDATE best_play_records bpr
SET username = pr.username, chart_id = pr.chart_id
FROM play_records pr
WHERE bpr.play_record_id = pr.play_record_id;

-- Remove rows we could not backfill (orphaned best records).
DELETE FROM best_play_records WHERE username IS NULL OR chart_id IS NULL;

ALTER TABLE best_play_records ALTER COLUMN username SET NOT NULL;
ALTER TABLE best_play_records ALTER COLUMN chart_id SET NOT NULL;

DO $$ BEGIN RAISE NOTICE '[step 5] best_play_records migrated: added username + chart_id columns.'; END $$;


-- ============================================================================
-- 6. Widen integer PK/FK columns to bigint (Go int = int64 on 64-bit systems)
-- ============================================================================
ALTER TABLE prober_users      ALTER COLUMN user_id        TYPE bigint;
ALTER TABLE songs             ALTER COLUMN song_id        TYPE bigint;
ALTER TABLE play_records      ALTER COLUMN play_record_id TYPE bigint;
ALTER TABLE play_records      ALTER COLUMN chart_id       TYPE bigint;
ALTER TABLE play_records      ALTER COLUMN score          TYPE bigint;
ALTER TABLE best_play_records ALTER COLUMN best_record_id TYPE bigint;
ALTER TABLE best_play_records ALTER COLUMN play_record_id TYPE bigint;

ALTER SEQUENCE prober_users_user_id_seq            AS bigint;
ALTER SEQUENCE songs_song_id_seq                   AS bigint;
ALTER SEQUENCE play_records_play_record_id_seq     AS bigint;
ALTER SEQUENCE best_play_records_best_record_id_seq AS bigint;

DO $$ BEGIN RAISE NOTICE '[step 6] All integer PK/FK columns widened to bigint.'; END $$;


-- ============================================================================
-- 7. prober_users.qq_number (int) → qq_account (text)
-- ============================================================================
ALTER TABLE prober_users ALTER COLUMN qq_number TYPE text USING qq_number::text;
ALTER TABLE prober_users RENAME COLUMN qq_number TO qq_account;

DO $$ BEGIN RAISE NOTICE '[step 7] prober_users.qq_number → qq_account (text).'; END $$;


-- ============================================================================
-- 8. Rename PK columns to `id` (Go convention: ID field)
-- ============================================================================
ALTER TABLE prober_users      RENAME COLUMN user_id        TO id;
ALTER SEQUENCE prober_users_user_id_seq            RENAME TO prober_users_id_seq;

ALTER TABLE songs             RENAME COLUMN song_id        TO id;
ALTER SEQUENCE songs_song_id_seq                   RENAME TO songs_id_seq;

ALTER TABLE play_records      RENAME COLUMN play_record_id TO id;
ALTER SEQUENCE play_records_play_record_id_seq     RENAME TO play_records_id_seq;

ALTER TABLE best_play_records RENAME COLUMN best_record_id TO id;
ALTER SEQUENCE best_play_records_best_record_id_seq RENAME TO best_play_records_id_seq;

-- charts.id already created as `id` in step 3.

DO $$ BEGIN RAISE NOTICE '[step 8] All PK columns renamed to id.'; END $$;


-- ============================================================================
-- 9. Add BaseModel audit columns (created_at, updated_at, deleted_at).
--    Audit columns are nullable with no DB-level default. GORM's
--    autoCreateTime / autoUpdateTime fills them from Go on every write;
--    we simply backfill historical rows here for sensible values.
-- ============================================================================

-- --- prober_users -----------------------------------------------------------
ALTER TABLE prober_users ADD COLUMN created_at timestamptz;
ALTER TABLE prober_users ADD COLUMN updated_at timestamptz;
ALTER TABLE prober_users ADD COLUMN deleted_at timestamptz;
UPDATE prober_users SET created_at = now(), updated_at = now()
  WHERE created_at IS NULL OR updated_at IS NULL;

-- --- songs ------------------------------------------------------------------
ALTER TABLE songs ADD COLUMN created_at timestamptz;
ALTER TABLE songs ADD COLUMN updated_at timestamptz;
ALTER TABLE songs ADD COLUMN deleted_at timestamptz;
UPDATE songs SET created_at = now(), updated_at = now()
  WHERE created_at IS NULL OR updated_at IS NULL;

-- --- charts -----------------------------------------------------------------
ALTER TABLE charts ADD COLUMN created_at timestamptz;
ALTER TABLE charts ADD COLUMN updated_at timestamptz;
ALTER TABLE charts ADD COLUMN deleted_at timestamptz;
UPDATE charts SET created_at = now(), updated_at = now()
  WHERE created_at IS NULL OR updated_at IS NULL;

-- --- play_records -----------------------------------------------------------
-- For historical play records, use record_time as a more meaningful audit
-- timestamp than now(); this lets tooling still reason about when records
-- were originally produced.
ALTER TABLE play_records ADD COLUMN created_at timestamptz;
ALTER TABLE play_records ADD COLUMN updated_at timestamptz;
ALTER TABLE play_records ADD COLUMN deleted_at timestamptz;
UPDATE play_records SET created_at = record_time, updated_at = record_time
  WHERE created_at IS NULL OR updated_at IS NULL;

-- --- best_play_records ------------------------------------------------------
-- Best records inherit their audit time from the referenced play_record.
ALTER TABLE best_play_records ADD COLUMN created_at timestamptz;
ALTER TABLE best_play_records ADD COLUMN updated_at timestamptz;
ALTER TABLE best_play_records ADD COLUMN deleted_at timestamptz;
UPDATE best_play_records bpr
SET created_at = pr.record_time, updated_at = pr.record_time
FROM play_records pr
WHERE bpr.play_record_id = pr.id
  AND (bpr.created_at IS NULL OR bpr.updated_at IS NULL);

-- Guard against any remaining NULLs (shouldn't happen after step 5, but be safe).
UPDATE best_play_records
   SET created_at = COALESCE(created_at, now()),
       updated_at = COALESCE(updated_at, now())
 WHERE created_at IS NULL OR updated_at IS NULL;

DO $$ BEGIN RAISE NOTICE '[step 9] BaseModel columns (created_at/updated_at/deleted_at) added to all tables.'; END $$;


-- ============================================================================
-- 10. Add SongBaseOverride columns to charts (all NULL for legacy data)
-- ============================================================================
ALTER TABLE charts ADD COLUMN override_title   text;
ALTER TABLE charts ADD COLUMN override_artist  text;
ALTER TABLE charts ADD COLUMN override_version text;
ALTER TABLE charts ADD COLUMN override_cover   text;

DO $$ BEGIN RAISE NOTICE '[step 10] SongBaseOverride columns added to charts.'; END $$;


-- ============================================================================
-- 11. Normalize column types to match GORM's Postgres dialect defaults
--     (so that running AutoMigrate after this script is a TRUE no-op).
--
--     GORM on PostgreSQL maps:
--       Go string   → text
--       Go float64  → decimal
--       Go int      → bigint (64-bit builds)
--       Go bool     → boolean
--
--     Legacy schema uses "character varying" and "double precision" — we
--     convert them here so AutoMigrate has nothing left to do.
-- ============================================================================

-- --- prober_users: varchar → text, int → bigint, bool defaults --------------
ALTER TABLE prober_users ALTER COLUMN username         TYPE text USING username::text;
ALTER TABLE prober_users ALTER COLUMN email            TYPE text USING email::text;
ALTER TABLE prober_users ALTER COLUMN nickname         TYPE text USING nickname::text;
ALTER TABLE prober_users ALTER COLUMN account          TYPE text USING account::text;
ALTER TABLE prober_users ALTER COLUMN account_number   TYPE bigint USING account_number::bigint;
ALTER TABLE prober_users ALTER COLUMN uuid             TYPE text USING uuid::text;
ALTER TABLE prober_users ALTER COLUMN upload_token     TYPE text USING upload_token::text;
ALTER TABLE prober_users ALTER COLUMN encoded_password TYPE text USING encoded_password::text;
ALTER TABLE prober_users ALTER COLUMN anonymous_probe  SET DEFAULT false;
ALTER TABLE prober_users ALTER COLUMN is_active        SET DEFAULT true;
ALTER TABLE prober_users ALTER COLUMN is_admin         SET DEFAULT false;

-- --- songs: all varchar text-likes → text -----------------------------------
ALTER TABLE songs ALTER COLUMN wiki_id     TYPE text USING wiki_id::text;
ALTER TABLE songs ALTER COLUMN title       TYPE text USING title::text;
ALTER TABLE songs ALTER COLUMN artist      TYPE text USING artist::text;
ALTER TABLE songs ALTER COLUMN genre       TYPE text USING genre::text;
ALTER TABLE songs ALTER COLUMN cover       TYPE text USING cover::text;
ALTER TABLE songs ALTER COLUMN illustrator TYPE text USING illustrator::text;
ALTER TABLE songs ALTER COLUMN version     TYPE text USING version::text;
ALTER TABLE songs ALTER COLUMN album       TYPE text USING album::text;
ALTER TABLE songs ALTER COLUMN bpm         TYPE text USING bpm::text;
ALTER TABLE songs ALTER COLUMN length      TYPE text USING length::text;

-- --- charts: float → decimal, varchar → text --------------------------------
ALTER TABLE charts ALTER COLUMN level         TYPE decimal USING level::decimal;
ALTER TABLE charts ALTER COLUMN fitting_level TYPE decimal USING fitting_level::decimal;
ALTER TABLE charts ALTER COLUMN level_design  TYPE text USING level_design::text;

-- --- play_records: username varchar → text ----------------------------------
ALTER TABLE play_records ALTER COLUMN username TYPE text USING username::text;

-- --- best_play_records: username varchar → text -----------------------------
ALTER TABLE best_play_records ALTER COLUMN username TYPE text USING username::text;

DO $$ BEGIN RAISE NOTICE '[step 11] Column types normalized to match GORM Postgres dialect.'; END $$;


-- ============================================================================
-- 12. Create all indexes (names MUST match what GORM's AutoMigrate generates
--     so that a post-migration AutoMigrate is a true no-op)
-- ============================================================================

-- --- prober_users -----------------------------------------------------------
CREATE UNIQUE INDEX IF NOT EXISTS idx_prober_users_upload_token ON prober_users (upload_token);
CREATE INDEX IF NOT EXISTS idx_prober_users_deleted_at   ON prober_users (deleted_at);

-- --- songs ------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_songs_b15         ON songs (b15);
CREATE INDEX IF NOT EXISTS idx_songs_deleted_at  ON songs (deleted_at);

-- --- charts -----------------------------------------------------------------
-- PARTIAL unique index: soft-deleted charts (deleted_at IS NOT NULL) do NOT
-- participate in uniqueness, so re-adding a chart with the same difficulty
-- after a soft-delete does not conflict.
CREATE UNIQUE INDEX IF NOT EXISTS idx_song_difficulty
    ON charts (song_id, difficulty)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_charts_deleted_at ON charts (deleted_at);

-- --- play_records -----------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_play_records_username   ON play_records (username);
CREATE INDEX IF NOT EXISTS idx_play_records_rating     ON play_records (rating);
-- Composite index: (username, chart_id). GORM tag declares priority:1 on
-- username and priority:2 on chart_id.
CREATE INDEX IF NOT EXISTS idx_pr_user_chart           ON play_records (username, chart_id);
CREATE INDEX IF NOT EXISTS idx_play_records_deleted_at ON play_records (deleted_at);

-- --- best_play_records ------------------------------------------------------
CREATE UNIQUE INDEX IF NOT EXISTS idx_best_user_chart
    ON best_play_records (username, chart_id);
CREATE INDEX IF NOT EXISTS idx_best_play_records_play_record_id
    ON best_play_records (play_record_id);
CREATE INDEX IF NOT EXISTS idx_best_play_records_deleted_at
    ON best_play_records (deleted_at);

DO $$ BEGIN RAISE NOTICE '[step 12] All indexes created (matching GORM tag names).'; END $$;


-- ============================================================================
-- 13. Drop legacy tables (and their orphaned sequences)
-- ============================================================================
DROP TABLE IF EXISTS song_levels  CASCADE;
DROP TABLE IF EXISTS difficulties CASCADE;

DO $$ BEGIN RAISE NOTICE '[step 13] Legacy tables (song_levels, difficulties) dropped.'; END $$;


-- ============================================================================
-- 14. Post-migration validation
-- ============================================================================
DO $$
DECLARE
    v_users_count        bigint;
    v_songs_count        bigint;
    v_charts_count       bigint;
    v_play_records_count bigint;
    v_best_records_count bigint;
    v_null_wiki_ids      bigint;
    v_null_usernames     bigint;
    v_null_created       bigint;
    v_orphan_charts      bigint;
    v_orphan_records     bigint;
BEGIN
    SELECT COUNT(*) INTO v_users_count        FROM prober_users;
    SELECT COUNT(*) INTO v_songs_count        FROM songs;
    SELECT COUNT(*) INTO v_charts_count       FROM charts;
    SELECT COUNT(*) INTO v_play_records_count FROM play_records;
    SELECT COUNT(*) INTO v_best_records_count FROM best_play_records;

    -- wiki_id integrity
    SELECT COUNT(*) INTO v_null_wiki_ids FROM songs WHERE wiki_id IS NULL;
    IF v_null_wiki_ids > 0 THEN
        RAISE EXCEPTION 'Validation failed: % songs still have NULL wiki_id', v_null_wiki_ids;
    END IF;

    -- best_play_records backfill integrity
    SELECT COUNT(*) INTO v_null_usernames FROM best_play_records WHERE username IS NULL;
    IF v_null_usernames > 0 THEN
        RAISE EXCEPTION 'Validation failed: % best_play_records have NULL username', v_null_usernames;
    END IF;

    -- BaseModel backfill integrity
    SELECT
        (SELECT COUNT(*) FROM prober_users      WHERE created_at IS NULL OR updated_at IS NULL) +
        (SELECT COUNT(*) FROM songs             WHERE created_at IS NULL OR updated_at IS NULL) +
        (SELECT COUNT(*) FROM charts            WHERE created_at IS NULL OR updated_at IS NULL) +
        (SELECT COUNT(*) FROM play_records      WHERE created_at IS NULL OR updated_at IS NULL) +
        (SELECT COUNT(*) FROM best_play_records WHERE created_at IS NULL OR updated_at IS NULL)
      INTO v_null_created;
    IF v_null_created > 0 THEN
        RAISE EXCEPTION 'Validation failed: % rows have NULL created_at or updated_at', v_null_created;
    END IF;

    -- Orphan checks (warnings only — legacy may genuinely have orphans)
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
    RAISE NOTICE '  prober_users:      % rows', v_users_count;
    RAISE NOTICE '  songs:             % rows', v_songs_count;
    RAISE NOTICE '  charts:            % rows (from song_levels)', v_charts_count;
    RAISE NOTICE '  play_records:      % rows', v_play_records_count;
    RAISE NOTICE '  best_play_records: % rows', v_best_records_count;
    RAISE NOTICE '============================================================';
END $$;


COMMIT;
