#!/bin/sh
# Not a migration: roles are cluster-level, so the ETL role cannot create
# itself (AD-6, AD-12). Runs once, on an empty data directory.
set -eu

for var in PORYGON_ETL_PASSWORD PORYGON_APP_PASSWORD PORYGON_BATCH_PASSWORD PORYGON_ANALYTICS_PASSWORD; do
	eval "value=\${$var:-}"
	if [ -z "$value" ]; then
		echo "initdb: $var is unset; set it in deploy/compose/.env" >&2
		exit 1
	fi
done

# :'name' quotes the value as a literal; never interpolate a password into SQL.
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" \
	-v db="$POSTGRES_DB" \
	-v etl_password="$PORYGON_ETL_PASSWORD" \
	-v app_password="$PORYGON_APP_PASSWORD" \
	-v batch_password="$PORYGON_BATCH_PASSWORD" \
	-v analytics_password="$PORYGON_ANALYTICS_PASSWORD" <<'SQL'
CREATE ROLE porygon_etl       LOGIN PASSWORD :'etl_password';
CREATE ROLE porygon_app       LOGIN PASSWORD :'app_password';
CREATE ROLE porygon_batch     LOGIN PASSWORD :'batch_password';
CREATE ROLE porygon_analytics LOGIN PASSWORD :'analytics_password';

REVOKE ALL ON DATABASE :"db" FROM PUBLIC;
GRANT CONNECT ON DATABASE :"db"
    TO porygon_etl, porygon_app, porygon_batch, porygon_analytics;

-- Redundant on PG15+; holds if this is ever restored onto an older cluster.
REVOKE CREATE ON SCHEMA public FROM PUBLIC;
GRANT USAGE, CREATE ON SCHEMA public TO porygon_etl;
GRANT USAGE ON SCHEMA public TO porygon_app, porygon_batch, porygon_analytics;
SQL
