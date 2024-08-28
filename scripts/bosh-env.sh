#!/bin/bash

pushd $(dirname ${BASH_SOURCE})/.. > /dev/null
  export BOSH_CLIENT=admin
  export BOSH_CLIENT_SECRET=$(bosh int creds/bosh.yml --path /admin_password)
  export BOSH_ENVIRONMENT=10.245.0.11
  export BOSH_CA_CERT=$(bosh int creds/bosh.yml --path /director_ssl/ca)
  ssh-keygen -f ~/.ssh/known_hosts -R 10.245.0.11
popd > /dev/null
