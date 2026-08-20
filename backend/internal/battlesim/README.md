# battlesim

CAP-5 — turn-based battle simulation, resolved as an ordered event stream.

Depends on `damagecalc` (calls it as the per-move resolution step) and `pokedex` (AD-2). Turn resolution is synchronous and in-process; RabbitMQ fan-out happens only after a turn's event log is finished (AD-4). Every `BATTLE_EVENT` carries an explicit monotonic `sequence` — never inferred from UUID order. Governed by AD-1, AD-2, AD-4, AD-6, AD-13 — see [docs/adr/](../../../docs/adr/).
