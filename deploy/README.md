# deploy

`compose/` — the Docker Compose stack (nginx, Go core, Postgres, RabbitMQ, OTel Collector, and eventually the Python analytics service), used for both local dev and the single prod VM.

`postgres/initdb/` — bind-mounted into the Postgres container's initdb hook. Creates AD-6's four roles, which are cluster-level objects the ETL role cannot create for itself and so cannot live in a migration. Runs once, on an empty data directory only.

`terraform/` — provisions the one VM this stack runs on. No staging environment (AD-11).
