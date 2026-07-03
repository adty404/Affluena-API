-- 000032: Notification deliveries — the sent-log + in-app store for the
-- notification scheduler (due reminders, weekly summary) and any rule-gated
-- send. Two jobs it serves:
--   1. De-dupe: a UNIQUE (user_id, rule_key, dedupe_key) means the same due item
--      (or the same weekly summary) is recorded at most once per window, so the
--      scheduler can tick frequently without re-notifying. dedupe_key encodes the
--      entity + window, e.g. "subscription:<id>:H-3:2026-07-10" or
--      "weekly-summary:2026-W27".
--   2. In-app feed: each row carries a rendered Indonesian title/message +
--      severity + action_path so an in-app notification exists even when the
--      chosen channel is email-only (channel records what was actually sent).
--
-- Rows are pruned by created_at (indexed) if a retention job is added later.
CREATE TABLE IF NOT EXISTS notification_deliveries (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    rule_key text NOT NULL,
    dedupe_key text NOT NULL,
    channel text NOT NULL,          -- 'email' | 'in-app' | 'both' (what was sent)
    title text NOT NULL,
    message text NOT NULL,
    severity text NOT NULL DEFAULT 'info',
    action_path text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, rule_key, dedupe_key)
);

CREATE INDEX IF NOT EXISTS idx_notification_deliveries_user_created
    ON notification_deliveries(user_id, created_at DESC);

COMMENT ON TABLE notification_deliveries IS 'Sent-log + in-app store for rule-gated notifications (de-dupe via UNIQUE(user_id, rule_key, dedupe_key)).';
COMMENT ON COLUMN notification_deliveries.dedupe_key IS 'Stable key for the entity+window, e.g. subscription:<id>:H-3:<due-date> or weekly-summary:<iso-week>.';
COMMENT ON COLUMN notification_deliveries.channel IS 'Channel actually used for the send: email, in-app, or both.';
