-- AD-6. The ETL role owns these tables, so it needs no grant. No ALTER DEFAULT
-- PRIVILEGES: a later table gets no access until a migration grants it.

GRANT SELECT ON
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
TO porygon_app, porygon_batch, porygon_analytics;
