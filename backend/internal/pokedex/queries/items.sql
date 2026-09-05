-- name: SearchItems :many
SELECT
    i.id,
    i.identifier,
    i.category,
    i.fling_power
FROM item i
JOIN data_set ds ON ds.id = i.data_set_id
WHERE i.identifier ILIKE @pattern::text ESCAPE '\'
  AND ds.identifier = @data_set::text
ORDER BY i.identifier;
