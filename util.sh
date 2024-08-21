#!/bin/bash

function do_help() {
  echo "Subcommands:"
  for subcommand in $(set | grep "^do_.* \(\)" | sed 's/^do_\(.*\) ()/\1/g')
  do
    echo "- $subcommand"
  done
  echo
  echo "Notes:"
  echo "* This script will detect if it is sourced in and setup an alias."
  echo "* Creds are placed into the 'creds/' folder."
  echo
  echo "Useful environment variables to export..."
  echo "- BOSH_LOG_LEVEL (set to 'debug' to capture all bosh activity including request/response)"
  echo "- BOSH_JUMPBOX_ENABLE (set to any value enable jumpbox user)"
  echo "- LXD_URL (set to HTTPS url of LXD server - not localhost)"
  echo "- LXD_INSECURE (default: false)"
  echo "- LXD_CLIENT_CERT (set to path of LXD TLS client certificate)"
  echo "- LXD_CLIENT_KEY (set to path of LXD TLS client key)"
  echo "- BOSH_DEPLOYMENT_DIR (default: \${HOME}/Documents/Source/bosh-deployment)"
  echo "- CONCOURSE_DIR when deploying Concourse"
  echo "- POSTGRES_DIR when deploying Postgres"
  echo
  echo "Currently set environment variables..."
  set | egrep "^(BOSH_LOG_LEVEL|LXD_URL|LXD_INSECURE|LXD_CLIENT_CERT|LXD_CLIENT_KEY|BOSH_DEPLOYMENT_DIR|CONCOURSE_DIR|ZOOKEEPER_DIR|POSTGRES_DIR)="
}

function do_init_lxd() {
  set -eu

  project=$(lxc project list --format csv | grep "boshdev" | cut -d, -f1)
  if [ -z "${project}" ]
  then
    lxc project create boshdev -c features.images=true -c features.storage.volumes=true
  fi
  lxc project list boshdev

  # note that a bridge is apparently "managed" and can not be set in the project itself
  network=$(lxc network list --format csv | grep "boshdevbr0" | cut -d, -f1)
  if [ -z "${network}" ]
  then
    lxc network create boshdevbr0 --type bridge ipv4.address=10.245.0.1/24 ipv4.nat=true ipv6.address=none dns.mode=none
  fi
  lxc network list

  lxc storage create --project boshdev boshdir dir source=/storage/boshdev-disks
  lxc storage list default
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
    -o $CF_DEPLOYMENT_DIR/operations/scale-to-one-az.yml \
    -o $CF_DEPLOYMENT_DIR/operations/set-router-static-ips.yml \
    -o $CF_DEPLOYMENT_DIR/operations/use-compiled-releases.yml \
    -o $CF_DEPLOYMENT_DIR/operations/use-latest-stemcell.yml \
    -l manifests/cloudfoundry-vars.yml
}

function do_destroy() {
  echo "Destroying all state..."
  rm -rf .dev_builds/
  rm -rf dev_releases/
  rm -rf creds/
  rm -f cpi
  rm -f state.json

  lxc --project boshdev list --format json |
    jq -r '.[] | .name | select(test("vm-[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"))' |
    xargs --verbose --no-run-if-empty --max-args=1 lxc delete -f
  lxc --project boshdev image list --format json |
    jq -r '.[] | select(.aliases[0].name // "not present" | test("img-[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}")) | .fingerprint' |
    xargs --verbose --no-run-if-empty --max-args=1 lxc image delete
  lxc storage volume list --format json |
    jq -r '.[] | select(.pool == "boshpool") | .name' |
    xargs --verbose --no-run-if-empty --max-args=1  lxc storage volume delete boshpool

  echo "Visual confirmation:"
  lxc --project boshdev list
  lxc --project boshdev image list
  lxc --project boshdev storage list
  lxc --project boshdev storage volume list default
}

function do_deploy_bosh() {
  set -eu
  bosh_deployment="${BOSH_DEPLOYMENT_DIR:-${HOME}/Documents/Source/bosh-deployment}"
  cpi_path=$PWD/cpi

  if [ -z "${LXD_URL:-}" ]
  then
    echo "LXD_URL must be set."
    exit 1
  fi
  lxd_url="${LXD_URL}"
  lxd_insecure="${LXD_INSECURE:-false}"
  lxd_client_cert="${LXD_CLIENT_CERT:-}"
  lxd_client_key="${LXD_CLIENT_KEY:-}"
  jumpbox_enable="${BOSH_JUMPBOX_ENABLE:-}"

  bosh_args=()
  if [ ! -z "${jumpbox_enable}" ]
  then
    bosh_args+=(--ops-file=${bosh_deployment}/jumpbox-user.yml)
  fi

  echo "-----> `date`: Create dev release"
  bosh create-release --force --tarball $cpi_path

  echo "-----> `date`: Create env"
  bosh create-env ${bosh_deployment}/bosh.yml \
    --ops-file=manifests/cpi.yml \
    --ops-file=${bosh_deployment}/bbr.yml \
    --ops-file=${bosh_deployment}/uaa.yml \
    --ops-file=${bosh_deployment}/credhub.yml \
    --ops-file=${bosh_deployment}/misc/dns.yml \
    --state=state.json \
    --vars-store=creds/bosh.yml \
    --vars-file=manifests/bosh-vars.yml \
    --var=cpi_path=$cpi_path \
    --var=lxd_url=$lxd_url \
    --var=lxd_insecure=$lxd_insecure \
    --var-file=lxd_client_cert=$lxd_client_cert \
    --var-file=lxd_client_key=$lxd_client_key "${bosh_args[@]}"

  bosh interpolate creds/bosh.yml --path /jumpbox_ssh/private_key > creds/jumpbox.pk
  chmod 600 creds/jumpbox.pk
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
  bosh update-cloud-config manifests/cloud-config.yml
}

function do_runtime_config() {
  source scripts/bosh-env.sh

  if [ -z "${BOSH_DEPLOYMENT_DIR}" ]
  then
    echo "Warning: BOSH_DEPLOYMENT_DIR is not set, not loading the bosh-dns runtime config! (Needed for CF)"
  else
    bosh update-runtime-config --name bosh-dns ${BOSH_DEPLOYMENT_DIR}/runtime-configs/dns.yml
  fi
}

function do_upload_stemcells() {
  source scripts/bosh-env.sh

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

  # # See https://github.com/concourse/concourse-bosh-deployment/issues/77
  # BBR=$(bosh --json releases | jq -r '[ .Tables[] | .Rows[] | select(.name == "backup-and-restore-sdk") ] | length')
  # if [ 0 -ne $BBR ]
  # then
  #   echo "bbr release exists"
  # else
  #   if [ ! -f release/backup-and-restore-sdk ]
  #   then
  #     echo "Downloading backup-and-restore-sdk"
  #     curl --location --output release/backup-and-restore-sdk \
  #       https://bosh.io/d/github.com/cloudfoundry-incubator/backup-and-restore-sdk-release?v=1.11.2
  #   fi
  #   bosh upload-release release/backup-and-restore-sdk
  # fi

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
       -l manifests/concourse-vars.yml
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

if [[ "$0" == "bash" ]]
then
  alias util="${BASH_SOURCE}"
  echo "'util' now available."
else
  mkdir -p creds
  mkdir -p stemcell
  mkdir -p release
  export BOSH_NON_INTERACTIVE=true
  do_${1:-help} ${2:-}
fi
