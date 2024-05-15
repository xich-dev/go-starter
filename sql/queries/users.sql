-- name: IsUsernameExist :one
SELECT EXISTS (SELECT 1 FROM users WHERE name = $1) AS exist;

-- name: IsPhoneExist :one
SELECT EXISTS (SELECT 1 FROM users WHERE phone = $1) AS exist;

-- name: CreateUser :one
INSERT INTO users (
    org_id,
    name,
    phone,
    password_hash,
    password_salt
) VALUES ($1, $2, $3, $4, $5) RETURNING * ;

-- name: GetUserAccessRuleNames :many
SELECT
    access_rules.name 
FROM access_rules
JOIN user_access_rules ON user_access_rules.rule_id = access_rules.id
WHERE user_access_rules.user_id = $1;

-- name: UpsertPhoneCode :one
INSERT INTO phone_code (
    phone,
    typ,
    code,
    expired_at,
    updated_at
) VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP) 
ON CONFLICT (phone, typ) DO UPDATE SET code = $3, expired_at = $4, used = FALSE, updated_at = CURRENT_TIMESTAMP
RETURNING * ;

-- name: GetPhoneCode :one
SELECT * FROM phone_code WHERE phone = $1 AND typ = $2;

-- name: MarkPhoneCodeUsed :exec
UPDATE phone_code SET used = TRUE WHERE phone = $1 AND typ = $2;

-- name: GetUser :one
SELECT * FROM users WHERE phone = $1 OR name = $1;

-- name: UpdateUserPasswordByPhone :exec
UPDATE users SET password_hash = $2, password_salt = $3 WHERE phone = $1;

-- name: GetOrgInfoByOrgId :one
SELECT * FROM orgs WHERE id = $1;
