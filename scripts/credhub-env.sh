#!/bin/bash

pushd $(dirname ${BASH_SOURCE})/.. > /dev/null
  export CREDHUB_CLIENT=credhub-admin
  export CREDHUB_SECRET=$(bosh int creds/bosh.yml --path /credhub_admin_client_secret)
  export CREDHUB_SERVER=$(bosh int manifests/bosh-vars.yml --path /internal_ip):8844
  file=~/.credhub_certs.pem
  bosh int creds/bosh.yml --path /credhub_ca/ca > ${file}
  bosh int creds/bosh.yml --path /uaa_ssl/ca >> ${file}
  export CREDHUB_CA_CERT=${file}
popd > /dev/null
