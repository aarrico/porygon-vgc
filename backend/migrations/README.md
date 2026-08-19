# migrations

Schema migrations, applied exclusively via `golang-migrate`, run only by the ETL role. Application code never runs a migration itself (AD-12).
