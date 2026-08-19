# deploy

`compose/` — the Docker Compose stack (nginx, Go core, Postgres, RabbitMQ, OTel Collector, and eventually the Python analytics service), used for both local dev and the single prod VM.

`terraform/` — provisions the one VM this stack runs on. No staging environment (AD-11).
