#!/bin/bash

function do_help() {
  echo "Subcommands:"
  for subcommand in $(set | grep "^do_.* \(\)" | sed 's/^do_\(.*\) ()/\1/g')
  do
    echo "- $subcommand"
  done
  echo
  echo "Note that this script will detect if it is sourced in and setup an alias."
  echo
  echo "Useful environment variables to export..."
  echo "- LXD_SOCKET (default: /var/lib/lxd/unix.socket)"
  echo "- BOSH_DEPLOYMENT (default: \${HOME}/Documents/Source/bosh-deployment)"
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

function do_clean() {
  echo "Cleaning out all state..."
  set -x
  rm -rf .dev_builds/
  rm -rf dev_releases/
  rm -f cpi
  rm -f creds.yml
  rm -f state.json

  lxc --project bosh list --format json |
    jq -r '.[] | .name | select(test("c-[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"))' |
    xargs --verbose --no-run-if-empty --max-args=1 lxc delete -f
  lxc --project bosh image list --format json |
    jq -r '.[] | select(.aliases[0].name // "not present" | test("img-[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}")) | .fingerprint' |
    xargs --verbose --no-run-if-empty --max-args=1 lxc image delete
}

function do_deploy() {
  set -eu
  bosh_deployment="${BOSH_DEPLOYMENT:-${HOME}/Documents/Source/bosh-deployment}"
  lxd_unix_socket="${LXD_SOCKET:-/var/lib/lxd/unix.socket}"
  cpi_path=$PWD/cpi

  rm -f creds.yml

  echo "-----> `date`: Create dev release"
  bosh create-release --force --tarball $cpi_path

  echo "-----> `date`: Create env"
  bosh create-env ${bosh_deployment}/bosh.yml \
    -o ./manifests/cpi.yml \
    -o ${bosh_deployment}/jumpbox-user.yml \
    --state=state.json \
    --vars-store=creds.yml \
    -v cpi_path=$cpi_path \
    -v director_name=lxd \
    -v lxd_unix_socket=$lxd_unix_socket \
    -v lxd_project_name=bosh \
    -v lxd_profile_name=default \
    -v lxd_network_name=boshbr0 \
    -v internal_cidr=10.245.0.0/16 \
    -v internal_gw=10.245.0.1 \
    -v internal_ip=10.245.0.11
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

if [[ "$0" == "bash" ]]
then
  alias util="${BASH_SOURCE}"
  echo "'util' now available."
else
  do_${1:-help}
fi
