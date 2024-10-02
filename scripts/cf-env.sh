#!/bin/bash

pushd $(dirname ${BASH_SOURCE})/.. > /dev/null
  export CF_API=$(bosh int manifests/cloudfoundry-vars.yml --path /system_domain)
  export CF_USERNAME=admin
  export CF_PASSWORD=$(credhub get -n /lxd/cf/cf_admin_password -q)
  cf api --skip-ssl-validation
  cf auth
popd > /dev/null
