#!/bin/bash

export BOSH_CLIENT=admin
export BOSH_CLIENT_SECRET=$(bosh int creds.yml --path /admin_password)
export BOSH_ENVIRONMENT=10.245.0.11
export BOSH_CA_CERT=$(bosh int creds.yml --path /director_ssl/ca)
