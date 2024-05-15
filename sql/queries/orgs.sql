-- name: CreateOrg :one
INSERT INTO orgs (
    name
) VALUES ($1) RETURNING * ;

-- name: UpdateOrgOwnerID :exec
UPDATE orgs SET owner_id = $1 WHERE id = $2;
