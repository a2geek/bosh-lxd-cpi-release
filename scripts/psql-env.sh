#!/bin/bash

if [[ "$0" != "bash" ]]
then
  echo "Please source this script in like:"
  echo "  $ source $0"
  exit 1
fi

pushd $(dirname ${BASH_SOURCE})/.. > /dev/null
  # See https://www.postgresql.org/docs/11/libpq-envars.html
  export PGHOST=$(bosh int manifests/postgres-vars.yml --path /postgres_host_or_ip)
  export PGPORT=5524
  export PGDATABASE=sandbox
  export PGUSER=pgadmin
  export PGPASSWORD=$(bosh int creds/postgres.yml --path /pgadmin_database_password)
popd > /dev/null
