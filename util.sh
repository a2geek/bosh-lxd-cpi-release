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
  find * -maxdepth 3 -type d | grep -v '^lxd-bosh-cpi$' | xargs rm -rf

  go get -d -t -v lxd-bosh-cpi/...

  # Remove all .git folders
  find * -type d -name '.git' | xargs rm -rf
}

function do_deploy() {
  bosh_deployment="${BOSH_DEPLOYMENT:-${HOME}/Documents/Source/bosh-deployment}"
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
    -v internal_cidr=10.245.0.0/16 \
    -v internal_gw=10.245.0.1 \
    -v internal_ip=10.245.0.11
}

if [[ "$0" == "bash" ]]
then
  alias util="${BASH_SOURCE}"
  echo "'util' now available."
else
  do_${1:-help}
fi
