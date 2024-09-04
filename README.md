# BOSH LXD CPI

> Would love PRs if people are still playing with BOSH in 2024!

This is a [BOSH CPI](https://bosh.io/) implementation to support [LXD](https://canonical.com/lxd) ... and likely [Incus](https://linuxcontainers.org/incus/introduction/).  It appears that both LXD and Incus are trying to keep comparable/compatible, so LXD comments hopefully apply to Incus as well.

## Requirements

LXD 5.21, LXD supports BIOS boots for VMs, which all the BOSH Stemcells use. Without this feature, they must be UEFI boot devices and VMs are not an option.

> A note on BIOS. This option is specified in the VM create call, or from the command-line with `--config raw.qemu="-bios bios-256k.bin"`. There is an environment variable the snap install of LXD sets up that points to the directory holding the various BIOS files. At this time, this is not configurable within the CPI itself.

As this depends on LXD, which is Linux only, this is also Linux only. Note that the BOSH deployment can be run from a Mac, and presumably from the Linux on WSL.

The current development environment is Ubuntu 22.04. LXD has been installed via a Snap and [this guide](https://documentation.ubuntu.com/lxd/en/latest/tutorial/first_steps/) was generally followed.

## Current State

Generally, the concept is to utilize LXD _projects_ as much as possible. They only piece I haven't tried to resolve is the networks -- if LXD manages and creates the bridge network, it must be in the default project. If you create the network and just tell LXD about it, I believe that the network can be localized to the BOSH project. The general thought is that other projects (VMs or containers) can be put to a different project and the projects would prevent accidental interactions.

### Completed

> Note that Postgres, Concourse (4 VMs), and Cloud Foundry (10+ VMs) are all capable of being deployed.

* A BOSH Director can be stood up.
* Network is configured and available.
* Disk is provisioned and attached.
* Generally, the common CPI calls are all implemented at this time.

### Incomplete

> Note that none of these are required for general operations!

* The following CPI methods are not implemented yet:
  * snapshots ([`snapshot_disk`](https://bosh.io/docs/cpi-api-v2-method/snapshot-disk/), [`delete_snapshot`](https://bosh.io/docs/cpi-api-v2-method/delete-snapshot/))
  * IaaS-native disk resizing ([`resize_disk`](https://bosh.io/docs/cpi-api-v2-method/resize-disk/))

### LXD adjustments

* There is a nightly scheduled process to look for unused configuration disks (`vol-c-<uuid>` format) since LXD doesn't give up a disk attachment unless a reboot of the VM occurs. (Or the LXD Agent is installed... which isn't at this time). It simply scan the list of detached configuration disks, tries to delete them, and reports success or error in the log. See [cleanup](src/cmd/cleanup/main.go).
* Throttling. Not so much LXD, but more the single host conundrum. There is a server that runs and maintains a hash map of "transaction" reservations. Once the CPI level transaction completes, the transaction is also released. Additionally, these reservations will time out after a certain amount time. (See Tuning for more details. Code is at [throttle](src/cmd/cleanup/main.go).)

### Tuning

Beyond the general tuning of a VM size (# of CPUs, amount of memory, or sizing of disks), the following is available for tuning the CPI. Note that the host development environment is a single server with a Xeon CPU, 128GB+ RAM, and SSDs (no spinning disks). SSDs are likely very important for I/O activities.

* Throttling CPI activity. This is a timed gateway type throttle. See the [spec](jobs/lxd_cpi/spec) for actual definition. Here is a sample of overrides. (Note `path` is the default and can likely just be left off.)

  ```yaml
  # This is for rendering within the VM once stood up
  - type: replace
    path: /instance_groups/name=bosh/properties/lxd_cpi/throttle_config?
    value:
      enabled: true
      path: "/var/vcap/sys/run/lxd_cpi/throttle.sock"
      limit: 4
      hold: "2m"
  ```

  > Note that the reason this is required is that for _every VM_ that LXD launches, the QCOW2 format source image gets converted to a RAW format image. I have not found a way to get around this. What this means is that there is ~5GiB of disk being copied _per VM_. So, launch 10 VMs and copy 50GiB of disk. If there is a resolution to this, please submit a ticket with details or create a PR with the fixes!

## LXD Setup

Note that the LXD configuration should be somewhat flexible. Review [`bosh-vars.yml`](manifests/bosh-vars.yml) for current set of configuration options.

```yaml
director_name: lxd
lxd_project_name: boshdev
lxd_profile_name: default
lxd_network_name: boshdevbr0
lxd_storage_pool_name: default
internal_cidr: 10.245.0.0/24
internal_gw: 10.245.0.1
internal_ip: 10.245.0.11
```

This CPI is designed to operate against a LXD Project. This should be able to keep the BOSH stemcells, VMs, and disks separate from any other activity on the LXD host.

Current setup of my machine (note these commands are for the project `boshdev`):

```bash
# Note that the directory driver is apparently very slow and caused timeouts.
# LXD/Incus documentation suggest ZFS or BTRFS are the best drivers. I found a doc on creating a ZFS pool and set that up.
# Note that the pool is not mounted (`-m none`).
$ zpool create -m none lxd-storage mirror /dev/sdb1 /dev/sdc1

# Now to the LXD Init
$ sudo lxd init
Would you like to use LXD clustering? (yes/no) [default=no]: 
Do you want to configure a new storage pool? (yes/no) [default=yes]: 
Name of the new storage pool [default=default]: 
Name of the storage backend to use (lvm, powerflex, zfs, btrfs, ceph, dir) [default=zfs]: zfs
Create a new ZFS pool? (yes/no) [default=yes]: no
Name of the existing ZFS pool or dataset: lxd-storage
Would you like to connect to a MAAS server? (yes/no) [default=no]: 
Would you like to create a new local network bridge? (yes/no) [default=yes]: 
What should the new bridge be called? [default=lxdbr0]: 
What IPv4 address should be used? (CIDR subnet notation, “auto” or “none”) [default=auto]: 
What IPv6 address should be used? (CIDR subnet notation, “auto” or “none”) [default=auto]: 
Would you like the LXD server to be available over the network? (yes/no) [default=no]: ### MAKE THIS YES!
Would you like stale cached images to be updated automatically? (yes/no) [default=yes]: 
Would you like a YAML "lxd init" preseed to be printed? (yes/no) [default=no]: 
$ lxc storage list
+---------+--------+-------------+-------------+---------+---------+
|  NAME   | DRIVER |   SOURCE    | DESCRIPTION | USED BY |  STATE  |
+---------+--------+-------------+-------------+---------+---------+
| default | zfs    | lxd-storage |             | 1       | CREATED |
+---------+--------+-------------+-------------+---------+---------+
rob@athena:~$ lxc storage show default
name: default
description: ""
driver: zfs
status: Created
config:
  source: lxd-storage
  volatile.initial_source: lxd-storage
  zfs.pool_name: lxd-storage
used_by:
- /1.0/profiles/default
locations:
- none

$ lxc project create boshdev -c features.images=true -c features.networks=true -c features.profiles=true -c features.storage.volumes=true
Project boshdev created
$ lxc project show boshdev
name: boshdev
description: ""
config:
  features.images: "true"
  features.networks: "true"
  features.profiles: "true"
  features.storage.buckets: "true"
  features.storage.volumes: "true"
used_by:
- /1.0/profiles/default?project=boshdev
$ lxc project switch boshdev

# Root disk is not specifically needed, but if you're using the LXC CLI, it helps creating experimental VMs. Note no network, since that is assigned in code.
$ lxc  profile list
+---------+-----------------------------------------+---------+
|  NAME   |               DESCRIPTION               | USED BY |
+---------+-----------------------------------------+---------+
| default | Default LXD profile for project boshdev | 0       |
+---------+-----------------------------------------+---------+
$ lxc profile show default
name: default
description: Default LXD profile for project boshdev
config: {}
devices: {}
used_by: []
$ lxc profile edit default
$ lxc profile show default
name: default
description: Default LXD profile for project boshdev
config: {}
devices:
  root:
    path: /
    pool: default
    type: disk
used_by: []

# Project doesn't apply to network when its a managed bridge
$ lxc --project default network create boshdevbr0 --type=bridge ipv4.address=10.245.0.1/24 ipv4.nat=true ipv6.address=none dns.mode=none
Network boshdevbr0 created
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
```

## Authentication

This CPI uses the LXD certificate authentication using [BOSH generated certificates](https://bosh.io/docs/director-certs/#generate).

Adding the (future) BOSH connection certificate is pretty simple:

```bash
$ lxc config trust add --name "bosh@<hostname>" ./bosh-client.crt
$ lxc config trust list
+--------+-----------------+-------------+--------------+------------------------------+------------------------------+
|  TYPE  |       NAME      | COMMON NAME | FINGERPRINT  |          ISSUE DATE          |         EXPIRY DATE          |
+--------+-----------------+-------------+--------------+------------------------------+------------------------------+
| client | bosh@<hostname> | bosh        | 4e39a923c420 | Jul 20, 2024 at 4:58pm (UTC) | Jul 18, 2034 at 4:58pm (UTC) |
+--------+-----------+-------------+--------------+------------------------------+--------+-----------+-------------+--------------+------------------------------+------------------------------+
```

If you develop on a remote machine, you can setup the LXC CLI to point to the remote server.

1. Use the `lxc remote add...` commands as described [here](https://documentation.ubuntu.com/lxd/en/latest/howto/server_expose/#authenticate-with-the-lxd-server).
2. Switch the default remote with `lxc remote switch <name>`.

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

> I have been using both Linux and Mac OS X to deploy. WSL should be ok as well. Just not native Windows due to the Unix assumptions in path names.

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

... and finally, to operate the BOSH director, there is a handy script that sets up the BOSH environment variables for access:

```bash
$ source scripts/bosh-env.sh 
$ bosh deployments
Using environment '10.245.0.11' as client 'admin'

Name       Release(s)                       Stemcell(s)                                     Team(s)  
cf         binary-buildpack/1.1.11          bosh-openstack-kvm-ubuntu-jammy-go_agent/1.506  -  
           bosh-dns/1.38.2                                                                    
           bosh-dns-aliases/0.0.4                                                             
           bpm/1.2.19                                                                         
           capi/1.181.0                                                                       
           cf-cli/1.63.0                                                                      
           cf-networking/3.46.0                                                               
           cf-smoke-tests/42.0.146                                                            
           cflinuxfs4/1.95.0                                                                  
           credhub/2.12.74                                                                    
           diego/2.99.0                                                                       
           dotnet-core-buildpack/2.4.27                                                       
           garden-runc/1.53.0                                                                 
           go-buildpack/1.10.18                                                               
           java-buildpack/4.69.0                                                              
           log-cache/3.0.11                                                                   
           loggregator/107.0.14                                                               
           loggregator-agent/8.1.1                                                            
           nats/56.19.0                                                                       
           nginx-buildpack/1.2.13                                                             
           nodejs-buildpack/1.8.24                                                            
           php-buildpack/4.6.18                                                               
           pxc/1.0.28                                                                         
           python-buildpack/1.8.23                                                            
           r-buildpack/1.2.11                                                                 
           routing/0.297.0                                                                    
           ruby-buildpack/1.10.13                                                             
           silk/3.46.0                                                                        
           staticfile-buildpack/1.6.12                                                        
           statsd-injector/1.11.40                                                            
           uaa/77.9.0                                                                         
concourse  backup-and-restore-sdk/1.18.119  bosh-openstack-kvm-ubuntu-jammy-go_agent/1.506  -  
           bosh-dns/1.38.2                                                                    
           bpm/1.2.16                                                                         
           concourse/7.11.2                                                                   
           postgres/48                                                                        
postgres   bosh-dns/1.38.2                  bosh-openstack-kvm-ubuntu-jammy-go_agent/1.506  -  
           postgres/52                                                                        

3 deployments

Succeeded
```
