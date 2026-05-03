-- name: UpsertCardConfig :exec
INSERT INTO card_config (account_id, points_program, reward_type, points_cpp, cpp_overridden)
VALUES (sqlc.arg(account_id), sqlc.arg(points_program), sqlc.arg(reward_type), sqlc.arg(points_cpp), sqlc.arg(cpp_overridden))
ON CONFLICT(account_id) DO UPDATE SET
    points_program = excluded.points_program,
    reward_type    = excluded.reward_type,
    points_cpp     = CASE WHEN excluded.cpp_overridden = 1 THEN excluded.points_cpp ELSE card_config.points_cpp END,
    cpp_overridden = CASE WHEN excluded.cpp_overridden = 1 THEN 1 ELSE card_config.cpp_overridden END;

-- name: GetCardConfig :one
SELECT * FROM card_config WHERE account_id = sqlc.arg(account_id);

-- name: ListCardConfigs :many
SELECT * FROM card_config;
