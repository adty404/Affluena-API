ALTER TABLE api_logs
ADD COLUMN request_payload TEXT,
ADD COLUMN response_payload TEXT;
