-- name: ListWebAuthnCredentials :many
SELECT * FROM webauthn_credentials ORDER BY created_at;

-- name: GetWebAuthnCredentialByCredentialID :one
SELECT * FROM webauthn_credentials WHERE credential_id = ?;

-- name: CreateWebAuthnCredential :exec
INSERT INTO webauthn_credentials (id, credential_id, public_key, sign_count, backup_eligible, backup_state)
VALUES (?, ?, ?, ?, ?, ?);

-- name: UpdateWebAuthnCredential :exec
UPDATE webauthn_credentials SET sign_count = ?, backup_eligible = ?, backup_state = ? WHERE credential_id = ?;

-- name: CountWebAuthnCredentials :one
SELECT COUNT(*) FROM webauthn_credentials;
