-- name: GetUserAccessRules :many
SELECT
    access_rules.name,
    access_rules.id
FROM access_rules
JOIN user_access_rules ON user_access_rules.rule_id = access_rules.id
WHERE user_access_rules.user_id = $1;

-- name: GetAccessRule :one
SELECT * FROM access_rules WHERE name = $1;

-- name: AddUserAccessRule :exec
INSERT INTO user_access_rules (user_id, rule_id) VALUES ((
    SELECT id FROM users WHERE name = $1
), $2) ON CONFLICT DO NOTHING;

-- name: RemoveUserAccessRule :exec
DELETE FROM user_access_rules WHERE user_id = $1 AND rule_id = $2;
