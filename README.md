# BOSH LXD CPI

This is a [BOSH CPI](https://bosh.io/) implementation to support [LXD](https://linuxcontainers.org/lxd/introduction/).

Please note that Go is new so the code is ugly. Working on making it functional before beautiful. ;-)

## Current State

What _is_ functional:
* A BOSH Director can be stood up.
* Network is configured and available.
* Disk is provisioned and attached. Does not survive a reboot (important for me because we do lose power from time-to-time).

What is _not_ functional:
* ZooKeeper has some issue mounting disks.
  - Not all API endpoints have an implementation and maybe this is the issue.
* Concourse deploys `web` and `db` but `worker` fails.
  - Maybe something related to privileges that Garden RunC requires?
  - If these need special privileges, maybe the cloud config needs to allow custom properties for LXD.
* Only supports Unix socket at this time.

## LXD Setup

Note that the LXD configuration should be somewhat flexible. Review `manifests/bosh-vars.yml` for current set of configuration options.

```
director_name: lxd
lxd_project_name: bosh
lxd_profile_name: default
lxd_network_name: boshbr0
lxd_storage_pool_name: default
internal_cidr: 10.245.0.0/16
internal_gw: 10.245.0.1
internal_ip: 10.245.0.11
credhub_encryption_password: topsecret
```
This CPI is designed to operate against a LXD Project. This should be able to keep the BOSH stemcells, vms, and disks (sort of) separate from any other activity on the LXD host.

Current setup of my machine (note these commands are for the project `bosh`):

```
$ lxc project show bosh
description: BOSH environment
config:
  features.images: "true"
  features.profiles: "true"
name: bosh

$ lxc --project bosh profile show default
config:
  raw.lxc: lxc.apparmor.profile = unconfined
description: Default LXD profile for project bosh
devices:
  root:
    path: /
    pool: default
    type: disk
name: default

# Project doesn't apply to network(?)
$ lxc network show lxdbr0
config:
  ipv4.address: 10.94.147.1/24
  ipv4.nat: "true"
  ipv6.address: none
description: ""
name: lxdbr0
type: bridge
managed: true
status: Created
locations:
- none

# This is my storage pool. Whatever you setup should be fine.
# Again, a storage pool is available to all projects.
$ lxc storage show default
config:
  size: 150GB
  source: /var/snap/lxd/common/lxd/disks/default.img
  zfs.pool_name: default
description: ""
name: default
driver: zfs
status: Created
locations:
- none
```

## Tinkering

If you wish to tinker with this, here is how I'm using the scripts:

```
# I have the Ubuntu LXD Snap installed, and the
# socket location is different, configure it like this:
$ export LXD_SOCKET=/var/snap/lxd/common/lxd/unix.socket

# This project needs the bosh-deployment project available.
# Set the location with BOSH_DEPLOYMENT.
$ export BOSH_DEPLOYMENT=...

# 'util.sh' has all the pieces currently being used.
# Source it in to setup the 'util' alias.
$ source ./util.sh
'util' now available.

# Note: 'util' assumes you are in the root of this project.

# Run alone to get a list of commands (not fancy but functional):
$ util
Subcommands:
- capture_requests
- clean
- cloud_config
- deploy_bosh
- deploy_concourse
- deploy_zookeeper
- deps
- help
- upload_releases
- upload_stemcells

Note that this script will detect if it is sourced in and setup an alias.

Useful environment variables to export...
- BOSH_LOG_LEVEL (set to 'debug' to capture all bosh activity including request/response)
- LXD_SOCKET (default: /var/lib/lxd/unix.socket)
- BOSH_DEPLOYMENT (default: ${HOME}/Documents/Source/bosh-deployment)
- CONCOURSE_DIR when deploying Concourse
- ZOOKEEPER_DIR when deploying ZooKeeper

# Most likely you want to deploy the bosh director:
$ util deploy_bosh
... hopefully this works!
```

## Cloud configuration

These properties are available in the cloud properties. Note that they are used for the BOSH Director (`manifests/cpi.yml`) as well as the default cloud configuration (`manifests/cloud-config.yml`).

```
cloud_properties:
  instance_type:
  ephemeral_disk:
  devices:
    name_of_device:
      type: device-type
      parameter1: value1
      parameter2: value2
```

Regarding devices, they are direct mappings into the LXD device configuration.  Sample from the BOSH Director:

```
devices:
  lxdsocket:
    type: proxy
    connect: unix:((lxd_unix_socket))
    listen: unix:/warden-cpi-dev/lxd.socket
    bind: container
    uid: "1000"   # vcap
    gid: "1000"   # vcap
    mode: "0660"
```

Additional information:
* [Instance Types](https://github.com/dustinkirkland/instance-type)
* [Devices](https://github.com/lxc/lxd/blob/master/doc/containers.md#devices-configuration)
