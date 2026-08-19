# teambuilder

CAP-4 — team-building recommendations and saved teams.

Depends on `pokedex` plus ETL-ingested tournament data — not on `damagecalc` or `battlesim` (AD-2). Reads a precomputed, batch-written scores table; never scores live (AD-7). Governed by AD-1, AD-2, AD-6, AD-7, AD-13 — see [ARCHITECTURE-SPINE.md](../../../_bmad-output/planning-artifacts/architecture/architecture-porygon-vgc-2026-08-19/ARCHITECTURE-SPINE.md).

Note: regulation/format-fit recommendations are out of scope until the format/regulation ruleset layer exists (deferred, see spine).
