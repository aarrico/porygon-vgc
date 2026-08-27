# migrations

Schema migrations, applied exclusively via `golang-migrate`, run only by the ETL role. Application code never runs a migration itself (AD-12).

Applied by the one-shot `migrate` service in `deploy/compose/docker-compose.yml`; `api` gates on it with `service_completed_successfully`, so the app can never boot against an unmigrated schema. `make migrate` re-runs it without restarting the stack.

| Version | Contents |
| --- | --- |
| `000001_core_reference_schema` | The 11 core reference tables (AD-15). `data_set_id` appears only on `move`, `move_meta`, `move_stat_change`, `ability`, `item`; `species`, `type`, `stat`, `generation` and `nature` are generation-scoped. |
| `000002_core_reference_grants` | `SELECT` for the app, batch and analytics roles (AD-6). The ETL role owns every table here and needs no grant of its own. |

The four AD-6 roles are cluster-level objects the ETL role cannot create, so they live in `deploy/postgres/initdb/10-roles.sh` rather than in a migration. That script runs once, as superuser, on an empty data directory — after changing a role password, `make clean` before `make up` or the change will not take.

Grants are written out per table deliberately: no `ALTER DEFAULT PRIVILEGES`, so a table added by a later migration has no role access until a migration says so (AD-6).

Not modelled yet, and not an oversight: ability slots and learnsets are both `data_set`-versioned relationships onto an unversioned `species`, and land together in a later slice.
