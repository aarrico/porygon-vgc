REVOKE SELECT ON
    data_set,
    generation,
    type,
    stat,
    nature,
    species,
    move,
    move_meta,
    move_stat_change,
    ability,
    item
FROM porygon_app, porygon_batch, porygon_analytics;
