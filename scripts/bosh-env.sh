#!/bin/bash

pushd $(dirname ${BASH_SOURCE})/.. > /dev/null
  export BOSH_CLIENT=admin
  export BOSH_CLIENT_SECRET=$(bosh int creds/bosh.yml --path /admin_password)
  export BOSH_ENVIRONMENT=$(bosh int manifests/bosh-vars.yml --path /internal_ip)
  export BOSH_CA_CERT=$(bosh int creds/bosh.yml --path /director_ssl/ca)
popd > /dev/null
