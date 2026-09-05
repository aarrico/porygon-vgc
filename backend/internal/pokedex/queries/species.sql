-- name: SearchSpecies :many
SELECT
    s.id,
    s.identifier,
    s.national_dex,
    s.is_default,
    g.identifier AS generation,
    t1.identifier AS type_1,
    t2.identifier AS type_2,
    s.base_hp,
    s.base_attack,
    s.base_defense,
    s.base_special_attack,
    s.base_special_defense,
    s.base_speed
FROM species s
JOIN generation g ON g.id = s.generation_id
JOIN type t1 ON t1.id = s.type_1_id
LEFT JOIN type t2 ON t2.id = s.type_2_id
WHERE s.identifier ILIKE @pattern::text ESCAPE '\'
ORDER BY s.national_dex, s.identifier;
