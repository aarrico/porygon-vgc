-- name: SearchAbilities :many
SELECT
    a.id,
    a.identifier,
    g.identifier AS generation
FROM ability a
JOIN data_set ds ON ds.id = a.data_set_id
JOIN generation g ON g.id = a.generation_id
WHERE a.identifier ILIKE @pattern::text ESCAPE '\'
    AND ds.identifier = @data_set::text
ORDER BY a.identifier;
