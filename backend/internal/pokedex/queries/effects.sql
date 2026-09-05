-- name: SearchMovesByEffect :many
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
    m.target,
    mm.category,
    mm.ailment,
    mm.ailment_chance,
    mm.stat_chance,
    mm.flinch_chance,
    mm.crit_rate,
    mm.drain,
    mm.healing,
    mm.min_hits,
    mm.max_hits,
    mm.min_turns,
    mm.max_turns
FROM move m
JOIN data_set ds ON ds.id = m.data_set_id
JOIN generation g ON g.id = m.generation_id
JOIN type t ON t.id = m.type_id
LEFT JOIN move_meta mm ON mm.move_id = m.id
WHERE ds.identifier = @data_set::text
  AND (cardinality(@ailments::text[]) = 0 OR mm.ailment = ALL(@ailments::text[]))
  AND (
      SELECT count(*)
      FROM unnest(@stat_identifiers::text[]) WITH ORDINALITY AS si(identifier, n)
      JOIN unnest(@stat_changes::smallint[]) WITH ORDINALITY AS sc(change, n) ON sc.n = si.n
      JOIN stat s ON s.identifier = si.identifier
      JOIN move_stat_change msc
        ON msc.move_id = m.id AND msc.stat_id = s.id AND msc.change = sc.change
  ) = cardinality(@stat_identifiers::text[])
ORDER BY m.identifier;

-- name: ListMoveStatChanges :many
SELECT msc.move_id, s.identifier AS stat, msc.change
FROM move_stat_change msc
JOIN stat s ON s.id = msc.stat_id
WHERE msc.move_id = ANY(@move_ids::bigint[])
ORDER BY msc.move_id, s.id;

-- name: ListUnknownStats :many
SELECT p.identifier::text
FROM unnest(@identifiers::text[]) AS p(identifier)
WHERE NOT EXISTS (SELECT 1 FROM stat s WHERE s.identifier = p.identifier);

-- name: ListUnknownAilments :many
SELECT p.ailment::text
FROM unnest(@ailments::text[]) AS p(ailment)
WHERE NOT EXISTS (SELECT 1 FROM move_meta mm WHERE mm.ailment = p.ailment);
