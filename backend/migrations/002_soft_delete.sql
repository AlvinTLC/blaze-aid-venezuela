-- 002_soft_delete.sql — add soft-delete support to the catalog entities.
ALTER TABLE aid_projects    ADD COLUMN IF NOT EXISTS deleted_at timestamptz;
ALTER TABLE resources       ADD COLUMN IF NOT EXISTS deleted_at timestamptz;
ALTER TABLE missing_persons ADD COLUMN IF NOT EXISTS deleted_at timestamptz;
ALTER TABLE volunteers      ADD COLUMN IF NOT EXISTS deleted_at timestamptz;

-- Partial indexes keep "active row" reads fast (the common case).
CREATE INDEX IF NOT EXISTS idx_aid_projects_active    ON aid_projects (updated_at)    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_resources_active       ON resources (updated_at)       WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_missing_persons_active ON missing_persons (updated_at) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_volunteers_active      ON volunteers (updated_at)      WHERE deleted_at IS NULL;
