# LXD Setup

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
$ lxc storage show default
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

# 1. Note that the root disk is needed. Prior code incantations (with and without size) caused problems or actual issues.
# 2. Note no network, since that is assigned in code.
$ lxc profile list
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
