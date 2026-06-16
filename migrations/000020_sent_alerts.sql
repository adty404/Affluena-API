-- Track sent budget alerts to prevent duplicate notifications
CREATE TABLE IF NOT EXISTS sent_alerts (
    user_id UUID NOT NULL,
    category_id UUID NOT NULL,
    month_value VARCHAR(7) NOT NULL,  -- Format: YYYY-MM
    alert_type VARCHAR(20) NOT NULL,  -- 'WARNING' or 'EXCEEDED'
    sent_at TIMESTAMP WITH TIME ZONE NOT NULL,
    PRIMARY KEY (user_id, category_id, month_value, alert_type)
);

-- Index for efficient cleanup of old alerts
CREATE INDEX IF NOT EXISTS idx_sent_alerts_sent_at ON sent_alerts(sent_at);

COMMENT ON TABLE sent_alerts IS 'Tracks budget alerts that have been sent to prevent duplicate notifications';
COMMENT ON COLUMN sent_alerts.month_value IS 'Month in YYYY-MM format for which the alert was sent';
COMMENT ON COLUMN sent_alerts.alert_type IS 'Type of alert: WARNING (80%) or EXCEEDED (100%)';
