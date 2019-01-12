# BOSH LXD CPI

This is a [BOSH CPI](https://bosh.io/) implementation to support [LXD](https://linuxcontainers.org/lxd/introduction/).

Please note that Go is new so the code is ugly. Working on making it functional before beautiful. ;-)

## Requirements

As this depends on LXD, which is Linux only, this is also Linux only.  See the [LXD Introduction](https://linuxcontainers.org/lxd/introduction/).

The current development environment is Ubuntu 18.04. LXD has been installed via a Snap and [this guide](https://linuxcontainers.org/lxd/getting-started-cli/) was generally followed.

## Current State

What _is_ functional:
* A BOSH Director can be stood up.
* Network is configured and available.
* Disk is provisioned and attached. Does not survive a reboot (important for me because we do lose power from time-to-time).

BOSH release status:

| Release | Status | Notes |
| --- | --- | --- |
| [concourse-bosh-deployment](https://github.com/concourse/concourse-bosh-deployment) | Does not work | Workers fail. |
| [postgres-release](https://github.com/cloudfoundry/postgres-release) | Works! | Suffers from the uptime bug. |
| [zookeeper-release](https://github.com/cppforlife/zookeeper-release) | Works! | Issues resolved with older Trusty stemcell. <br> Suffers from the uptime bug. |

What is _not_ functional:
* Concourse deploys `web` and `db` but `worker` fails.
  - Maybe something related to privileges that Garden RunC requires?
  - If these need special privileges, maybe the cloud config needs to allow custom properties for LXD.
* Only supports Unix socket at this time.
* `bosh vms` and `bosh instances` fails with negative uptime for Postgres:
  ```
  $ bosh -d postgres vms
  Using environment '10.245.0.11' as client 'admin'

  Task 17. Done

  Unmarshaling vm info response: '{"vm_cid":"c-b4b2c88b-3941-4e37-5991-6e3858610410","active":true,"vm_created_at":"2019-01-04T03:58:00Z","cloud_properties":{"ephemeral_disk":2048,"instance_type":"c1-m2"},"disk_cid":"vol-p-2786a14e-6199-479b-7e06-22d260816507","disk_cids":["vol-p-2786a14e-6199-479b-7e06-22d260816507"],"ips":["10.245.0.12"],"dns":[],"agent_id":"7db1a505-76bf-401e-bf02-42e63189b5b4","job_name":"postgres","index":0,"job_state":"running","state":"started","resource_pool":"small","vm_type":"small","vitals":{"cpu":{"sys":"2.3","user":"3.8","wait":"0.0"},"disk":{"ephemeral":{"inode_percent":"0","percent":"2"},"persistent":{"inode_percent":"0","percent":"0"},"system":{"inode_percent":"2","percent":"27"}},"load":["1.35","1.16","0.89"],"mem":{"kb":"28528","percent":"1"},"swap":{"kb":"232448","percent":"1"},"uptime":{"secs":977141}},"processes":[{"name":"postgres","state":"running","uptime":{"secs":-972769},"mem":{"kb":59752,"percent":2.9},"cpu":{"total":0}},{"name":"pg_janitor","state":"running","uptime":{"secs":-972770},"mem":{"kb":23560,"percent":1.1},"cpu":{"total":0}}],"resurrection_paused":false,"az":"z1","id":"248d04db-2a15-44c5-8e97-a9fcbda3e35c","bootstrap":true,"ignore":false}':
    json: cannot unmarshal number -972769 into Go struct field VMInfoVitalsUptime.secs of type uint64

  Exit code 1
  ```
  This appears to likely be related to [this non-issue in Monit](https://bitbucket.org/tildeslash/monit/issues/310/monit-in-lxc-containers)... that still has a useful patch.

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
  raw.lxc: |
    lxc.apparmor.profile = unconfined
  security.privileged: "true"
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
