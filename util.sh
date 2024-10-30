#!/bin/bash

function print_varlist() {
  varlist=$(echo "$@" | tr ' ' '\n' | sort)
  for varname in $varlist
  do
    if [ ! -z "${!varname}" ]
    then
      printf -- "- %-25s %s\n" "${varname}:" "${!varname}"
    fi
    shift
  done
}

function do_help() {
  echo "Subcommands:"
  for subcommand in $(set | grep "^do_.* \(\)" | sed 's/^do_\(.*\) ()/\1/g')
  do
    echo "- ${subcommand/_/-}"
  done
  echo
  echo "Notes:"
  echo "* This script will detect if it is sourced in and setup an alias."
  echo "  (^^ does not work from IDE such as VS Code)"
  echo "* Creds are placed into the 'creds/' folder."
  echo
  echo "Useful environment variables to export..."
  echo "- BOSH_LOG_LEVEL (set to 'debug' to capture all bosh activity including request/response)"
  echo "- BOSH_JUMPBOX_ENABLE (set to any value enable jumpbox user)"
  echo "- BOSH_SNAPSHOTS_ENABLE (set to any value to enable snapshots)"
  echo "- SERVER_URL (set to HTTPS url of server - not localhost)"
  echo "- SERVER_INSECURE (default: false)"
  echo "- SERVER_CLIENT_CERT (set to path of TLS client certificate)"
  echo "- SERVER_CLIENT_KEY (set to path of TLS client key)"
  echo "- SERVER_ENABLE_AGENT (set to 'lxd' or 'incus' to enable that agent)"
  echo "- BOSH_DEPLOYMENT_DIR (default: \${HOME}/Documents/Source/bosh-deployment)"
  echo "- BOSH_PACKAGE_GOLANG_DIR (default ../bosh-package-golang-release)"
  echo "- CONCOURSE_DIR when deploying Concourse"
  echo "- POSTGRES_DIR when deploying Postgres"
  echo "- CPI_DIR (location of bosh-lxd-cpi-release, default '.')"
  echo "- CPI_CONFIG_DIR (location of vars files, default to 'CPI_DIR/manifests')"
  echo
  echo "Configuration values..."
  print_varlist server_project_name server_profile_name server_network_name server_storage_pool_name internal_ip
  echo
  echo "Currently set environment variables..."
  print_varlist BOSH_LOG_LEVEL SERVER_URL SERVER_INSECURE SERVER_CLIENT_CERT SERVER_CLIENT_KEY \
                BOSH_DEPLOYMENT_DIR BOSH_PACKAGE_GOLANG_DIR \
                CONCOURSE_DIR ZOOKEEPER_DIR POSTGRES_DIR
}

function do_stress_test() {
  set -eu

  delay="${1:-}"

  for cmd in destroy deploy_bosh cloud_config runtime_config upload_releases upload_stemcells deploy_postgres deploy_concourse deploy_cf
  do
    if [ ! -z "${delay}" ]
    then
      echo "... pausing for ${delay} seconds ..."
      sleep ${delay}
    fi
    do_${cmd}
  done
}

function do_final_release() {
  set -eu

  if [ $# -ne 1 ]
  then
    echo "Please include version number in command line."
    exit 1
  fi

  version="$1"
  bosh create-release --final --version=${version} --tarball=bosh-lxd-cpi-release.tgz
}

function do_fix_blobs() {
  golang_releases=${BOSH_PACKAGE_GOLANG_DIR:-"../bosh-package-golang-release"}
  if [ -d ./blobs ]
  then
    rm -rf ./blobs
  fi
  mkdir ./blobs
  for path in $(find ./packages/ -name "golang*")
  do
    package_name=$(basename $path)
    bosh vendor-package ${package_name} ${golang_releases}
  done
}

function do_deploy_cf() {
  if [ -z "$CF_DEPLOYMENT_DIR" ]
  then
    echo "Please set CF_DEPLOYMENT_DIR to root of cf-deployment"
    exit 1
  fi
  set -eu
  source scripts/bosh-env.sh
  export BOSH_DEPLOYMENT=cf
  bosh deploy $CF_DEPLOYMENT_DIR/cf-deployment.yml \
    -o $CF_DEPLOYMENT_DIR/operations/use-compiled-releases.yml \
    -o $CF_DEPLOYMENT_DIR/operations/use-latest-stemcell.yml \
    -o $CF_DEPLOYMENT_DIR/operations/override-app-domains.yml \
    -o $CF_DEPLOYMENT_DIR/operations/scale-to-one-az.yml \
    -o $CF_DEPLOYMENT_DIR/operations/use-haproxy.yml \
    -l $cpi_config_dir/cloudfoundry-vars.yml
#    -o $CF_DEPLOYMENT_DIR/operations/set-router-static-ips.yml \
}

function do_destroy() {
  echo "Destroying all state..."
  rm -rf .dev_builds/
  rm -rf dev_releases/
  rm -rf creds/
  rm -f cpi
  rm -f state.json

  lxc --project  ${server_project_name} list --format json |
    jq -r '.[] | .name | select(test("vm-[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"))' |
    xargs --verbose --no-run-if-empty --max-args=1 lxc delete -f
  lxc --project  ${server_project_name} image list --format json |
    jq -r '.[] | select(.aliases[0].name // "not present" | test("img-[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}")) | .fingerprint' |
    xargs --verbose --no-run-if-empty --max-args=1 lxc image delete
  lxc --project  ${server_project_name} storage volume list --format json |
    jq -r --arg poolname "${server_storage_pool_name}" '.[] | select(.pool == $poolname) | .name' |
    xargs --verbose --no-run-if-empty --max-args=1  lxc storage volume delete  ${server_storage_pool_name}

  echo "Visual confirmation:"
  lxc --project  ${server_project_name} list
  lxc --project  ${server_project_name} image list
  lxc --project  ${server_project_name} storage list
  lxc --project  ${server_project_name} storage volume list  ${server_storage_pool_name}
}

function do_generate_certs() {
  cat > creds/bosh-cert-manifest.yml <<EOF
variables:
- name: bosh_ca
  type: certificate
  options:
    is_ca: true
    common_name: bosh_ca
    duration: 1095
- name: director_ssl
  type: certificate
  options:
    ca: bosh_ca
    common_name: ${internal_ip}
    alternative_names: [${internal_ip}]
EOF
  bosh interpolate creds/bosh-cert-manifest.yml --vars-store creds/bosh-cert.yml
  bosh interpolate creds/bosh-cert.yml --path /director_ssl/certificate > bosh-client.crt
  bosh interpolate creds/bosh-cert.yml --path /director_ssl/private_key > bosh-client.key
}

function do_deploy_bosh() {
  set -eu
  bosh_deployment="${BOSH_DEPLOYMENT_DIR:-${HOME}/Documents/Source/bosh-deployment}"

  if [ -z "${SERVER_URL:-}" ]
  then
    echo "SERVER_URL must be set."
    exit 1
  fi

  local_release="${SERVER_LOCAL_RELEASE:-}"
  server_url="${SERVER_URL}"
  server_insecure="${SERVER_INSECURE:-false}"
  server_client_cert="${SERVER_CLIENT_CERT:-}"
  server_client_key="${SERVER_CLIENT_KEY:-}"
  server_enable_agent="${SERVER_ENABLE_AGENT:-}"
  jumpbox_enable="${BOSH_JUMPBOX_ENABLE:-}"
  snapshots_enable="${BOSH_SNAPSHOTS_ENABLE:-}"
  resize_disk_enable="${BOSH_RESIZE_DISK_ENABLE:-}"
  internal_dns_enable="$(bosh int ${cpi_config_dir}/bosh-vars.yml --path /internal_dns 2>/dev/null || true)"

  bosh_args=()
  [ ! -z "${jumpbox_enable}"      ] && bosh_args+=(--ops-file=${bosh_deployment}/jumpbox-user.yml)
  [ ! -z "${snapshots_enable}"    ] && bosh_args+=(--ops-file=${cpi_dir}/ops/enable-snapshots.yml)
  [ ! -z "${resize_disk_enable}"  ] && bosh_args+=(--ops-file=${bosh_deployment}/misc/cpi-resize-disk.yml)
  [ ! -z "${internal_dns_enable}" ] && bosh_args+=(--ops-file=${bosh_deployment}/misc/dns.yml)
  [ ! -z "${server_enable_agent}" ] && bosh_args+=(--ops-file=${cpi_dir}/ops/enable-${server_enable_agent}-agent.yml)

  if [ ! -z "${local_release}" ]
  then
    cpi_path=$PWD/cpi
    echo "-----> `date`: Create dev release"
    bosh create-release --force --tarball $cpi_path
    bosh_args+=(--ops-file=${cpi_dir}/ops/local-release.yml --var=cpi_path=${cpi_path})
  fi

  echo "-----> `date`: Create env"
  bosh create-env ${bosh_deployment}/bosh.yml \
    --ops-file=${cpi_dir}/cpi.yml \
    --ops-file=${bosh_deployment}/bbr.yml \
    --ops-file=${bosh_deployment}/uaa.yml \
    --ops-file=${bosh_deployment}/credhub.yml \
    --state=state.json \
    --vars-store=creds/bosh.yml \
    --vars-file=${cpi_config_dir}/bosh-vars.yml \
    --var=server_url=$server_url \
    --var=server_insecure=$server_insecure \
    --var-file=server_client_cert=$server_client_cert \
    --var-file=server_client_key=$server_client_key "${bosh_args[@]}"

  bosh interpolate creds/bosh.yml --path /jumpbox_ssh/private_key > creds/jumpbox.pk
  chmod 600 creds/jumpbox.pk
  ssh-keygen -f ~/.ssh/known_hosts -R ${internal_ip}
}

function do_capture_requests() {
  if [ ! -f log ]
  then
    echo "Expecting to find 'log' file."
    exit 1
  fi
  if [ -e requests ] 
  then
    rm -rf requests
  fi
  mkdir -p requests
  num=0
  grep "STDIN: " log | while read LINE
    do
      (( num=num+1 ))
      echo $LINE | sed -nr "s/STDIN: '(.*)'/\1/p" | json_pp > requests/request-$num.json
    done
  num=0
  grep "STDOUT: " log | while read LINE
    do
      (( num=num+1 ))
      echo $LINE | sed -nr "s/STDOUT: '(.*)'/\1/p" | json_pp > requests/response-$num.json
    done
}

function do_cloud_config() {
  source scripts/bosh-env.sh
  bosh update-cloud-config ${cpi_config_dir}/cloud-config.yml
}

function do_runtime_config() {
  source scripts/bosh-env.sh

  if [ -z "${BOSH_DEPLOYMENT_DIR}" ]
  then
    echo "Warning: BOSH_DEPLOYMENT_DIR is not set, not loading the bosh-dns runtime config! (Needed for CF)"
  else
    bosh update-runtime-config --name bosh-dns ${BOSH_DEPLOYMENT_DIR}/runtime-configs/dns.yml
  fi

  if [ ! -z "${SERVER_ENABLE_AGENT:-}" ]
  then
    bosh update-runtime-config --name ${SERVER_ENABLE_AGENT}-agent ${cpi_dir}/manifests/enable-${SERVER_ENABLE_AGENT}-agent-config.yml
  fi
}

function do_upload_stemcells() {
  source scripts/bosh-env.sh

  NEW_STEMCELLS=""
  if [ "new" == "${1-}" ]
  then
    NEW_STEMCELLS=yes
  fi

  JAMMY=$(bosh stemcells --json | jq -r '[ .Tables[] | .Rows[] | select(.os == "ubuntu-jammy")] | length')
  if [ 0 -ne ${JAMMY} -a -z "${NEW_STEMCELLS}" ]
  then
    echo "ubuntu-jammy stemcell exists"
  else
    if [ -f stemcell/ubuntu-jammy-image -a ! -z "${NEW_STEMCELLS}" ]
    then
      rm stemcell/ubuntu-jammy-image
    fi
    if [ ! -f stemcell/ubuntu-jammy-image ]
    then
      echo "Downloading ubuntu-jammy-image"
      curl --location --output stemcell/ubuntu-jammy-image \
        https://bosh.io/d/stemcells/bosh-openstack-kvm-ubuntu-jammy-go_agent
    fi
    bosh upload-stemcell stemcell/ubuntu-jammy-image
  fi
}

function do_upload_releases() {
  source scripts/bosh-env.sh

  POSTGRES=$(bosh --json releases | jq -r '[ .Tables[] | .Rows[] | select(.name == "postgres-release") ] | length')
  if [ 0 -ne $POSTGRES ]
  then
    echo "postgres release exists"
  else
    if [ ! -f release/postgres-release ]
    then
      echo "Downloading postgres-release"
      curl --location --output release/postgres-release \
        https://bosh.io/d/github.com/cloudfoundry/postgres-release
    fi
    bosh upload-release release/postgres-release
  fi
}

function do_deploy_concourse() {
  if [ -z "$CONCOURSE_DIR" ]
  then
    echo "Please set CONCOURSE_DIR to root of concourse-bosh-deployment"
    exit 1
  fi
  set -eu
  source scripts/bosh-env.sh
  export BOSH_DEPLOYMENT=concourse
  bosh deploy $CONCOURSE_DIR/cluster/concourse.yml \
       -o $CONCOURSE_DIR/cluster/operations/backup-atc.yml \
       -o $CONCOURSE_DIR/cluster/operations/basic-auth.yml \
       -o $CONCOURSE_DIR/cluster/operations/static-web.yml \
       -o $CONCOURSE_DIR/cluster/operations/privileged-http.yml \
       -l $CONCOURSE_DIR/versions.yml \
       --vars-store=creds/concourse.yml \
       -l ${cpi_config_dir}/concourse-vars.yml
}

function do_deploy_postgres() {
  if [ -z "$POSTGRES_DIR" ]
  then
    echo "Please set POSTGRES_DIR to root of postgres-release"
    exit 1
  fi
  set -eu
  rm -f creds/postgres.yml
  source scripts/bosh-env.sh
  export BOSH_DEPLOYMENT=postgres
  bosh deploy manifests/postgres.yml \
       --vars-store=creds/postgres.yml \
       -l manifests/postgres-vars.yml
}

function read_config_file() {
  server_project_name=$(bosh interpolate ${cpi_config_dir}/bosh-vars.yml --path /server_project_name)
  server_profile_name=$(bosh interpolate ${cpi_config_dir}/bosh-vars.yml --path /server_profile_name)
  server_network_name=$(bosh interpolate ${cpi_config_dir}/bosh-vars.yml --path /server_network_name)
  server_storage_pool_name=$(bosh interpolate ${cpi_config_dir}/bosh-vars.yml --path /server_storage_pool_name)
  internal_ip=$(bosh interpolate ${cpi_config_dir}/bosh-vars.yml --path /internal_ip)
}

if [[ "$0" == "bash" ]]
then
  alias util="${BASH_SOURCE}"
  echo "'util' now available."
else
  mkdir -p creds
  mkdir -p stemcell
  mkdir -p release
  export BOSH_NON_INTERACTIVE=true
  cpi_dir=${CPI_DIR:-.}
  cpi_config_dir=${CPI_CONFIG_DIR:-$cpi_dir/manifests}
  read_config_file
  cmd=${1:-help}
  shift
  do_${cmd/-/_} "$@"
fi
