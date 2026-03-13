#!/bin/bash
set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    CREATE USER reviewboard WITH PASSWORD '${POSTGRES_RB_PASSWORD:-changeme_rb_password}';
    GRANT ALL PRIVILEGES ON DATABASE reviewboard TO reviewboard;
    ALTER DATABASE reviewboard OWNER TO reviewboard;
EOSQL
