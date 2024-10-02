#!/bin/bash

pushd $(dirname ${BASH_SOURCE})/.. > /dev/null
  uaa_admin_client_secret=$(bosh int creds/bosh.yml --path /uaa_admin_client_secret)
  internal_ip=$(bosh int manifests/bosh-vars.yml --path /internal_ip)
  uaa target https://${internal_ip}:8443 --skip-ssl-validation
  uaa get-client-credentials-token uaa_admin -s ${uaa_admin_client_secret}
popd > /dev/null
