#!/bin/bash

function do_help() {
  echo "Subcommands:"
  for subcommand in $(set | grep "^do_.* \(\)" | sed 's/^do_\(.*\) ()/\1/g')
  do
    echo "- $subcommand"
  done
  echo "Note that this script will detect if it is sourced in and setup an alias."
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
  bosh_deployment="${BOSH_DEPLOYMENT:-~/Documents/Source/bosh-deployment}"
  cpi_path=$PWD/cpi

  rm -f creds.yml

  echo "-----> `date`: Create dev release"
  bosh create-release --force --dir ./../ --tarball $cpi_path

  echo "-----> `date`: Create env"
  bosh create-env ${bosh_deployment}/bosh.yml \
    -o ~/workspace/bosh-deployment/docker/cpi.yml \
    -o ${bosh_deployment}/jumpbox-user.yml \
    -o ../manifests/dev.yml \
    --state=state.json \
    --vars-store=creds.yml \
    -v docker_cpi_path=$cpi_path \
    -v director_name=docker \
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
