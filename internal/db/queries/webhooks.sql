-- name: CreateWebhookEvent :exec
-- Uses the payload hash as ID for idempotent dedup — retried identical payloads are silently ignored.
INSERT OR IGNORE INTO webhook_events (id, item_id, webhook_type, webhook_code, payload)
VALUES (?, ?, ?, ?, ?);
