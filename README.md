# BOSH LXD CPI

> Incomplete. Would love PRs if people are still playing with BOSH in 2024!

This is a [BOSH CPI](https://bosh.io/) implementation to support [LXD](https://canonical.com/lxd) ... and likely [Incus](https://linuxcontainers.org/incus/introduction/).  It appears that both LXD and Incus are trying to keep comparable/compatible, so LXD comments hopefully apploy to Incus as well.

## Requirements

LXD 5.21, LXD supports BIOS boots for VMs, which all the BOSH Stemcells use. Without this feature, they must be UEFI boot devices and VMs are not an option.

> A note on BIOS. This option is specified in the VM create call, or from the command-line with `--config raw.qemu="-bios bios-256k.bin"`. There is an environment variable the snap install of LXD sets up that points to the directory holding the various BIOS files. At this time, this is not configurable.

As this depends on LXD, which is Linux only, this is also Linux only.

The current development environment is Ubuntu 22.04. LXD has been installed via a Snap and [this guide](https://documentation.ubuntu.com/lxd/en/latest/tutorial/first_steps/) was generally followed.

## Current State

What _is_ functional:

* A BOSH Director can be stood up.
* Network is configured and available.
* Disk is provisioned and attached.

Problem areas:

* The agent sometimes fails. (It doesn't communicate back to BOSH, and I can't get into the VM to understand why.) For example, Concourse deploys but Postgres (DB only) does not.
* There are sometimes issues with BOSH DNS. When a lookup against 169.254.0.2:53 is tried, to appears to work.
* CF doesn't get far.

## LXD Setup

Note that the LXD configuration should be somewhat flexible. Review `manifests/bosh-vars.yml` for current set of configuration options.

```yaml
director_name: lxd
lxd_project_name: boshdev
lxd_profile_name: default
lxd_network_name: boshdevbr0
lxd_storage_pool_name: boshpool
internal_cidr: 10.245.0.0/24
internal_gw: 10.245.0.1
internal_ip: 10.245.0.11
```

This CPI is designed to operate against a LXD Project. This should be able to keep the BOSH stemcells, vms, and disks (sort of) separate from any other activity on the LXD host.

Current setup of my machine (note these commands are for the project `boshdev`):

```bash
$ lxc project switch boshdev
$ lxc project show boshdev
name: boshdev
description: ""
config:
  features.images: "true"
  features.networks: "true"
  features.profiles: "true"
  features.storage.buckets: "true"
  features.storage.volumes: "true"

$ lxc profile show default
name: default
description: Default LXD profile for project boshdev
config: {}
devices: {}

# Project doesn't apply to network when its a managed bridge
$ lxc --project default network show boshdevbr0
name: boshdevbr0
description: ""
type: bridge
managed: true
status: Created
config:
  dns.mode: none
  ipv4.address: 10.245.0.1/24
  ipv4.nat: "true"
  ipv6.address: none
used_by: []
locations:
- none

# Note that the directory driver is apparently very slow and caused timeouts.
# LXD/Incus documentation suggest ZFS or BTRFS are the best drivers. I found a doc on creating a ZFS pool and set that up.
$ zpool create -m none lxd-storage mirror /dev/sdb1 /dev/sdc1
$ lxc storage create boshpool zfs source=lxd-storage
$ lxc storage show boshpool
name: boshpool
description: ""
driver: zfs
status: Created
config:
  source: lxd-storage
  volatile.initial_source: lxd-storage
  zfs.pool_name: lxd-storage
```

## Authentication

This CPI uses the LXD certificate authentication [using BOSH](https://bosh.io/docs/director-certs/#generate).

## Partitioning

```bash
# sgdisk --zap-all /dev/sdb
Caution! After loading partitions, the CRC doesn't check out!
Warning! Main partition table CRC mismatch! Loaded backup partition table
instead of main partition table!

Warning! One or more CRCs don't match. You should repair the disk!
Main header: OK
Backup header: OK
Main partition table: ERROR
Backup partition table: OK

****************************************************************************
Caution: Found protective or hybrid MBR and corrupt GPT. Using GPT, but disk
verification and recovery are STRONGLY recommended.
****************************************************************************
GPT data structures destroyed! You may now partition the disk using fdisk or
other utilities.
# sgdisk --zap-all /dev/sdc
Caution! After loading partitions, the CRC doesn't check out!
Warning! Main partition table CRC mismatch! Loaded backup partition table
instead of main partition table!

Warning! One or more CRCs don't match. You should repair the disk!
Main header: OK
Backup header: OK
Main partition table: ERROR
Backup partition table: OK

****************************************************************************
Caution: Found protective or hybrid MBR and corrupt GPT. Using GPT, but disk
verification and recovery are STRONGLY recommended.
****************************************************************************
GPT data structures destroyed! You may now partition the disk using fdisk or
other utilities.
# sgdisk -E /dev/sdb
Creating new GPT entries in memory.
2000409230
# sgdisk -E /dev/sdc
Creating new GPT entries in memory.
1953525134
# sgdisk --new 1:2048:1953525134 /dev/sdc
Creating new GPT entries in memory.
The operation has completed successfully.
# sgdisk --new 1:2048:1953525134 /dev/sdb
Creating new GPT entries in memory.
The operation has completed successfully.
# sgdisk --verify /dev/sdb

Caution: Partition 1 doesn't end on a 2048-sector boundary. This may
result in problems with some disk encryption tools.

No problems found. 46886110 free sectors (22.4 GiB) available in 2
segments, the largest of which is 46884096 (22.4 GiB) in size.
# sgdisk --verify /dev/sdc

Caution: Partition 1 doesn't end on a 2048-sector boundary. This may
result in problems with some disk encryption tools.

No problems found. 2014 free sectors (1007.0 KiB) available in 1
segments, the largest of which is 2014 (1007.0 KiB) in size.
# lsblk
NAME                      MAJ:MIN RM   SIZE RO TYPE MOUNTPOINTS
<snip>
sdb                         8:16   0 953.9G  0 disk 
└─sdb1                      8:17   0 931.5G  0 part 
sdc                         8:32   0 931.5G  0 disk 
└─sdc1                      8:33   0 931.5G  0 part 
<snip>
```

## Tinkering

If you wish to tinker with this, here is how I'm working with the tools.

First, create a handy environment script (I call it `env.sh`, and it's in `.gitignore` so no secrets are ever at risk).

```bash
export LXD_URL="https://<server>:8443"
export LXD_INSECURE="true"
export LXD_CLIENT_CERT="$PWD/bosh-client.crt"
export LXD_CLIENT_KEY="$PWD/bosh-client.key"
#export BOSH_LOG_LEVEL="DEBUG"
export BOSH_JUMPBOX_ENABLE="yes, please"
export POSTGRES_DIR="<...>/postgres-release"
export BOSH_DEPLOYMENT_DIR="<...>/bosh-deployment"
export CONCOURSE_DIR="<...>/concourse-bosh-deployment"
export CF_DEPLOYMENT_DIR="<...>/cf-deployment"
```

This script can then be source in with `source ./env.sh`.

Second, the `util.sh` script is setup to be sourced in. Note that it must be an actual Bash shell (and not something from an IDE like VS Code).

```bash
$ source ./util.sh 
'util' now available.
```

> Note: 'util' assumes you are in the root of this project.

Run alone to get a list of commands (not fancy but functional):

```bash
$  util
Subcommands:
- capture_requests
- cloud_config
- deploy_bosh
- deploy_cf
- deploy_concourse
- deploy_postgres
- destroy
- help
- init_lxd
- runtime_config
- upload_releases
- upload_stemcells

Notes:
* This script will detect if it is sourced in and setup an alias.
* Creds are placed into the 'creds/' folder.

Useful environment variables to export...
- BOSH_LOG_LEVEL (set to 'debug' to capture all bosh activity including request/response)
- BOSH_JUMPBOX_ENABLE (set to any value enable jumpbox user)
- LXD_URL (set to HTTPS url of LXD server - not localhost)
- LXD_INSECURE (default: false)
- LXD_CLIENT_CERT (set to path of LXD TLS client certificate)
- LXD_CLIENT_KEY (set to path of LXD TLS client key)
- BOSH_DEPLOYMENT_DIR (default: ${HOME}/Documents/Source/bosh-deployment)
- CONCOURSE_DIR when deploying Concourse
- POSTGRES_DIR when deploying Postgres

Currently set environment variables...
BOSH_DEPLOYMENT_DIR=<...>/bosh-deployment
CONCOURSE_DIR=<...>/concourse-bosh-deployment
LXD_CLIENT_CERT=<...>/bosh-lxd-cpi-release/bosh-client.crt
LXD_CLIENT_KEY=<...>/bosh-lxd-cpi-release/bosh-client.key
LXD_INSECURE=true
LXD_URL=https://<server>:8443
POSTGRES_DIR=<...>/postgres-release
```

Most likely you want to deploy the bosh director:

```bash
$ util deploy_bosh
<lots of output>
```

... and hopefully this works! This particular deploy operation will create a `creds/jumpbox.pk` so an SSH connection can be established:

```bash
$ ssh -i creds/jumpbox.pk jumpbox@10.245.0.11
Unauthorized use is strictly prohibited. All access and activity
is subject to logging and monitoring.
Welcome to Ubuntu 22.04.4 LTS (GNU/Linux 5.15.0-117-generic x86_64)

 * Documentation:  https://help.ubuntu.com
 * Management:     https://landscape.canonical.com
 * Support:        https://ubuntu.com/pro
Last login: Wed Aug 21 23:52:09 UTC 2024 from 192.168.1.254 on pts/0
Last login: Wed Aug 21 23:52:12 2024 from 192.168.1.254
To run a command as administrator (user "root"), use "sudo <command>".
See "man sudo_root" for details.

bosh/0:~$ sudo -i
bosh/0:~# monit summary
The Monit daemon 5.2.5 uptime: 3h 51m 

Process 'nats'                      running
Process 'bosh_nats_sync'            running
Process 'postgres'                  running
Process 'blobstore_nginx'           running
Process 'director'                  running
Process 'worker_1'                  running
Process 'worker_2'                  running
Process 'worker_3'                  running
Process 'worker_4'                  running
Process 'director_scheduler'        running
Process 'director_sync_dns'         running
Process 'director_nginx'            running
Process 'health_monitor'            running
Process 'lxd_cpi'                   running
Process 'uaa'                       running
Process 'credhub'                   running
System 'system_0db11f4b-1cfc-47a8-5dd5-6f13a053d7b7' running
```
