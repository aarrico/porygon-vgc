-- name: SearchMoves :many
SELECT
    m.id,
    m.identifier,
    g.identifier AS generation,
    t.identifier AS type,
    m.damage_class,
    m.power,
    m.accuracy,
    m.pp,
    m.priority,
    m.target
FROM move m
JOIN data_set ds ON ds.id = m.data_set_id
JOIN generation g ON g.id = m.generation_id
JOIN type t ON t.id = m.type_id
WHERE m.identifier ILIKE @pattern::text ESCAPE '\'
  AND ds.identifier = @data_set::text
ORDER BY m.identifier;
