-- name: FindDataSetByIdentifier :one
SELECT id, identifier, parent_id
FROM data_set
WHERE identifier = @identifier::text;
