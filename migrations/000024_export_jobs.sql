-- 000024: Export jobs audit trail.

CREATE TABLE IF NOT EXISTS export_jobs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    format text NOT NULL DEFAULT 'csv',
    from_at timestamptz,
    to_at timestamptz,
    row_count integer NOT NULL DEFAULT 0,
    status text NOT NULL DEFAULT 'completed',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_export_jobs_user ON export_jobs(user_id);
CREATE INDEX IF NOT EXISTS idx_export_jobs_created ON export_jobs(created_at DESC);
