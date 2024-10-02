#!/bin/bash

pushd $(dirname ${BASH_SOURCE})/.. > /dev/null
  uaa_admin_client_secret=$(credhub get -n /lxd/cf/uaa_admin_client_secret -q)
  system_domain=$(bosh int manifests/cloudfoundry-vars.yml --path /system_domain)
  uaa target https://uaa.${system_domain} --skip-ssl-validation
  uaa get-client-credentials-token admin -s ${uaa_admin_client_secret}
popd > /dev/null
