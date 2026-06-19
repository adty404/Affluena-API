CREATE TABLE IF NOT EXISTS notification_rules (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  rule_key text NOT NULL,
  title text NOT NULL,
  description text NOT NULL,
  enabled boolean NOT NULL DEFAULT true,
  channel text NOT NULL DEFAULT 'both',
  tone text NOT NULL DEFAULT 'gray',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE(user_id, rule_key)
);

CREATE INDEX IF NOT EXISTS idx_notification_rules_user_id ON notification_rules(user_id);
