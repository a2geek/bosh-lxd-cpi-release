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
  echo "- LXD_SOCKET (default: /var/lib/lxd/unix.socket or /var/snap/lxd/common/lxd/unix.socket)"
  echo "- BOSH_DEPLOYMENT_DIR (default: \${HOME}/Documents/Source/bosh-deployment)"
  echo "- CONCOURSE_DIR when deploying Concourse"
  echo "- ZOOKEEPER_DIR when deploying ZooKeeper"
  echo "- POSTGRES_DIR when deploying Postgres"
  echo "- LXD_MONIT_PATCH_DIR when deploying the runtime config patch"
  echo
  echo "Currently set environment variables..."
  set | egrep "^(BOSH_LOG_LEVEL|LXD_SOCKET|BOSH_DEPLOYMENT_DIR|CONCOURSE_DIR|ZOOKEEPER_DIR|POSTGRES_DIR|LXD_MONIT_PATCH_DIR)="
}

function do_deps() {
  export GOPATH=$PWD

  cd src

  # Remove all deps
  #find * -maxdepth 3 -type d | grep -v '^lxd-bosh-cpi$' | xargs rm -rf

  go get -d -t -v lxd-bosh-cpi/...

  # Remove all .git folders
  find * -type d -name '.git' | xargs rm -rf
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
    -o $CF_DEPLOYMENT_DIR/operations/enable-privileged-container-support.yml \
    -o $CF_DEPLOYMENT_DIR/operations/set-router-static-ips.yml \
    -o $CF_DEPLOYMENT_DIR/operations/use-compiled-releases.yml \
    -o $CF_DEPLOYMENT_DIR/operations/use-latest-stemcell.yml \
    -l manifests/cloudfoundry-vars.yml
}

function do_destroy() {
  echo "Deleting deployments via BOSH..."
  if [ -f creds/bosh.yml ]
  then
    source scripts/bosh-env.sh
    bosh --json deployments |
      jq -r '.Tables[] | .Rows[] | .name' |
      xargs --verbose --no-run-if-empty --max-args=1 --replace=DEPLOYMENT \
        bosh -d DEPLOYMENT delete-deployment --force --non-interactive
  fi

  echo "Destroying out all state..."
  set -x
  rm -rf .dev_builds/
  rm -rf dev_releases/
  rm -rf creds/
  rm -f cpi
  rm -f state.json

  lxc --project bosh list --format json |
    jq -r '.[] | .name | select(test("c-[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"))' |
    xargs --verbose --no-run-if-empty --max-args=1 lxc delete -f
  lxc --project bosh image list --format json |
    jq -r '.[] | select(.aliases[0].name // "not present" | test("img-[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}")) | .fingerprint' |
    xargs --verbose --no-run-if-empty --max-args=1 lxc image delete
  # NO JSON AVAILABLE?!
  lxc storage volume list default |
    grep custom |
    cut -d"|" -f3 |
    xargs --verbose --no-run-if-empty --max-args=1  lxc storage volume delete default

  echo "Visual confirmation:"
  lxc --project bosh list
  lxc --project bosh image list
  lxc --project bosh storage list
  lxc --project bosh storage volume list default
}

function do_deploy_bosh() {
  set -eu
  bosh_deployment="${BOSH_DEPLOYMENT_DIR:-${HOME}/Documents/Source/bosh-deployment}"
  cpi_path=$PWD/cpi

  if [ -z "${LXD_SOCKET:-}" ]
  then
    if [ -S /var/lib/lxd/unix.socket ]
    then
      LXD_SOCKET="/var/lib/lxd/unix.socket"
    elif [ -S /var/snap/lxd/common/lxd/unix.socket ]
    then
      LXD_SOCKET="/var/snap/lxd/common/lxd/unix.socket"
    fi
  fi
  lxd_unix_socket="${LXD_SOCKET:-/var/lib/lxd/unix.socket}"

  rm -f creds/bosh.yml

  echo "-----> `date`: Create dev release"
  bosh create-release --force --tarball $cpi_path

  extra_ops_files=()
  if [ ! -z "${LXD_MONIT_PATCH_DIR:-}" ]
  then
    echo "-----> `date`: Adding lxd-monit-patch"
    extra_ops_files+=("--ops-file=${LXD_MONIT_PATCH_DIR}/manifests/operations/add-to-director.yml")
  fi

  echo "-----> `date`: Create env"
  bosh create-env ${bosh_deployment}/bosh.yml \
    --ops-file=manifests/cpi.yml \
    --ops-file=${bosh_deployment}/bbr.yml \
    --ops-file=${bosh_deployment}/uaa.yml \
    --ops-file=${bosh_deployment}/credhub.yml \
    "${extra_ops_files[@]}" \
    --state=state.json \
    --vars-store=creds/bosh.yml \
    --vars-file=manifests/bosh-vars.yml \
    --var=cpi_path=$cpi_path \
    --var=lxd_unix_socket=$lxd_unix_socket
}

function do_capture_requests() {
  if [ ! -f log ]
  then
    echo "Expecting to find 'log' file."
    exit 1
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

  if [ -z "${LXD_MONIT_PATCH_DIR}" ]
  then
    echo "Warning: LXD_MONIT_PATCH_DIR is not set, not loading the monit-patch runtime config!"
  else
    bosh update-runtime-config --name monit-patch ${LXD_MONIT_PATCH_DIR}/manifests/runtime-config.yml
  fi

  if [ -z "${BOSH_DEPLOYMENT_DIR}" ]
  then
    echo "Warning: BOSH_DEPLOYMENT_DIR is not set, not loading the bosh-dns runtime config! (Needed for CF)"
  else
    bosh update-runtime-config --name bosh-dns ${BOSH_DEPLOYMENT_DIR}/runtime-configs/dns.yml
  fi
}

function do_upload_stemcells() {
  source scripts/bosh-env.sh

  set -x

  if [ "new" == "${1-}" ]
  then
    NEW_STEMCELLS=yes
  fi

  TRUSTY=$(bosh stemcells --json | jq -r '[ .Tables[] | .Rows[] | select(.os == "ubuntu-trusty")] | length')
  if [ 0 -ne ${TRUSTY} -a -z "${NEW_STEMCELLS}" ]
  then
    echo "ubuntu-trusty stemcell exists"
  else
    if [ -f stemcell/ubuntu-trusty-image -a ! -z "${NEW_STEMCELLS}" ]
    then
      rm stemcell/ubuntu-trusty-image
    fi
    if [ ! -f stemcell/ubuntu-trusty-image ]
    then
      echo "Downloading ubuntu-trusty-image"
      curl --location --output stemcell/ubuntu-trusty-image \
        https://bosh.io/d/stemcells/bosh-warden-boshlite-ubuntu-trusty-go_agent
    fi
    bosh upload-stemcell stemcell/ubuntu-trusty-image
  fi

  XENIAL=$(bosh stemcells --json | jq -r '[ .Tables[] | .Rows[] | select(.os == "ubuntu-xenial")] | length')
  if [ 0 -ne ${XENIAL} -a -z "${NEW_STEMCELLS}" ]
  then
    echo "ubuntu-xenial stemcell exists"
  else
    if [ -f stemcell/ubuntu-xenial-image -a ! -z "${NEW_STEMCELLS}" ]
    then
      rm stemcell/ubuntu-xenial-image
    fi
    if [ ! -f stemcell/ubuntu-xenial-image ]
    then
      echo "Downloading ubuntu-xenial-image"
      curl --location --output stemcell/ubuntu-xenial-image \
        https://bosh.io/d/stemcells/bosh-warden-boshlite-ubuntu-xenial-go_agent
    fi
    bosh upload-stemcell stemcell/ubuntu-xenial-image
  fi
}

function do_upload_releases() {
  source scripts/bosh-env.sh

  # See https://github.com/concourse/concourse-bosh-deployment/issues/77
  BBR=$(bosh --json releases | jq -r '[ .Tables[] | .Rows[] | select(.name == "backup-and-restore-sdk") ] | length')
  if [ 0 -ne $BBR ]
  then
    echo "bbr release exists"
  else
    if [ ! -f release/backup-and-restore-sdk ]
    then
      echo "Downloading backup-and-restore-sdk"
      curl --location --output release/backup-and-restore-sdk \
        https://bosh.io/d/github.com/cloudfoundry-incubator/backup-and-restore-sdk-release?v=1.11.2
    fi
    bosh upload-release release/backup-and-restore-sdk
  fi

  POSTGRES=$(bosh --json releases | jq -r '[ .Tables[] | .Rows[] | select(.name == "postgres-release") ] | length')
  if [ 0 -ne $BBR ]
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
  bosh deploy $POSTGRES_DIR/templates/postgres.yml \
       -o $POSTGRES_DIR/templates/operations/add_static_ips.yml \
       -o $POSTGRES_DIR/templates/operations/set_properties.yml \
       -o $POSTGRES_DIR/templates/operations/use_bbr.yml \
       --vars-store=creds/postgres.yml \
       -l manifests/postgres-vars.yml
}

function do_deploy_zookeeper() {
  if [ -z "$ZOOKEEPER_DIR" ]
  then
    echo "Please set ZOOKEEPER_DIR to root of zookeeper-release"
    exit 1
  fi
  set -eu
  source scripts/bosh-env.sh
  export BOSH_DEPLOYMENT=zookeeper
  bosh deploy $ZOOKEEPER_DIR/manifests/zookeeper.yml \
       --vars-store=creds/zookeeper.yml
  bosh run-errand smoke-tests
  bosh run-errand status
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
