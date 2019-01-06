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
  echo "- LXD_SOCKET (default: /var/lib/lxd/unix.socket)"
  echo "- BOSH_DEPLOYMENT_DIR (default: \${HOME}/Documents/Source/bosh-deployment)"
  echo "- CONCOURSE_DIR when deploying Concourse"
  echo "- ZOOKEEPER_DIR when deploying ZooKeeper"
  echo "- POSTGRES_DIR when deploying Postgres"
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

function do_destroy() {
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
  lxd_unix_socket="${LXD_SOCKET:-/var/lib/lxd/unix.socket}"
  cpi_path=$PWD/cpi

  rm -f creds/bosh.yml

  echo "-----> `date`: Create dev release"
  bosh create-release --force --tarball $cpi_path

  echo "-----> `date`: Create env"
  bosh create-env ${bosh_deployment}/bosh.yml \
    --ops-file=manifests/cpi.yml \
    --ops-file=${bosh_deployment}/bbr.yml \
    --ops-file=${bosh_deployment}/uaa.yml \
    --ops-file=${bosh_deployment}/credhub.yml \
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

function do_upload_stemcells() {
  source scripts/bosh-env.sh

  TRUSTY=$(bosh stemcells --json | jq -r '[ .Tables[] | .Rows[] | select(.os == "ubuntu-trusty")] | length')
  if [ 0 -eq $TRUSTY ]
  then
    bosh upload-stemcell --sha1 c51a1ae3faadd470ded7b83f69d09971bdd7de7b \
         https://bosh.io/d/stemcells/bosh-warden-boshlite-ubuntu-trusty-go_agent?v=3586.65
  else
    echo "ubuntu-trusty stemcell exists"
  fi

  XENIAL=$(bosh stemcells --json | jq -r '[ .Tables[] | .Rows[] | select(.os == "ubuntu-xenial")] | length')
  if [ 0 -eq $XENIAL ]
  then
      bosh upload-stemcell --sha1 f67cc28a19d39ca0504f7fbf5a39859cc934c5eb \
           https://bosh.io/d/stemcells/bosh-warden-boshlite-ubuntu-xenial-go_agent?v=170.15
  else
    echo "ubuntu-xenial stemcell exists"
  fi
}

function do_upload_releases() {
  source scripts/bosh-env.sh

  # See https://github.com/concourse/concourse-bosh-deployment/issues/77
  BBR=$(bosh --json releases | jq -r '[ .Tables[] | .Rows[] | select(.name == "backup-and-restore-sdk") ] | length')
  if [ 0 -eq $BBR ]
  then
    bosh upload-release --sha1 8e74035caee59d84ec46e9dc5d84084e8c5ca5c8 \
         https://bosh.io/d/github.com/cloudfoundry-incubator/backup-and-restore-sdk-release?v=1.11.2
  else
    echo "bbr release exists"
  fi

  POSTGRES=$(bosh --json releases | jq -r '[ .Tables[] | .Rows[] | select(.name == "postgres-release") ] | length')
  if [ 0 -eq $BBR ]
  then
    bosh upload-release https://bosh.io/d/github.com/cloudfoundry/postgres-release
  else
    echo "postgres release exists"
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
  export BOSH_NON_INTERACTIVE=true
  do_${1:-help}
fi
