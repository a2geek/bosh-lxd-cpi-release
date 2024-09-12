# BOSH LXD CPI

> Would love PRs if people are still playing with BOSH in 2024!

This is a [BOSH CPI](https://bosh.io/) implementation to support [LXD](https://canonical.com/lxd) ... and likely [Incus](https://linuxcontainers.org/incus/introduction/).  It appears that both LXD and Incus are trying to keep comparable/compatible, so LXD comments (and API calls) hopefully apply to Incus as well.

## Requirements

LXD 5.21, LXD supports BIOS boots for VMs, which all the BOSH Stemcells use. Without this feature, they must be UEFI boot devices and VMs are not an option.

As this depends on LXD, which is Linux only, this is also Linux only. Note that the BOSH deployment can be run from a Mac, and presumably from the Linux on WSL.

The current development environment is Ubuntu 22.04. LXD (currently `5.21/stable`) has been installed via a Snap and [this guide](https://documentation.ubuntu.com/lxd/en/latest/tutorial/first_steps/) was generally followed.

## Documentation

* [LXD Setup](docs/LXD-SETUP.md) contains some notes from setting up the development environment.
* [Authentication](docs/AUTHENTICATION.md) shows an example for how to setup the certificate authentication for LXD.
* [Develiping](docs/DEVELOPING.md) covers how the tools in this repository are used.
* [Deploying a BOSH director](docs/DEPLOYING.md) walks through a sample deployment for a fresh install (of nearly everything).
* Supplied [configuration options](ops/README.md).
* [Working with Genesis](docs/GENESIS.md) shows how to deploy to Genesis with a non-standard CPI.

## Current State

* A BOSH Director can be stood up.
* Network is configured and available.
* Disk is provisioned and attached.
* Generally, the common CPI calls are all implemented at this time, including the optional CPI methods:
  * snapshots ([`snapshot_disk`](https://bosh.io/docs/cpi-api-v2-method/snapshot-disk/), [`delete_snapshot`](https://bosh.io/docs/cpi-api-v2-method/delete-snapshot/))
  * IaaS-native disk resizing ([`resize_disk`](https://bosh.io/docs/cpi-api-v2-method/resize-disk/))

### LXD adjustments

* There is a nightly scheduled process to look for unused configuration disks (`vol-c-<uuid>` format) since LXD doesn't give up a disk attachment unless a reboot of the VM occurs. (Or the LXD Agent is installed... which isn't at this time). It simply scan the list of detached configuration disks, tries to delete them, and reports success or error in the log. See [cleanup](src/cmd/cleanup/main.go).
* Throttling. Not so much LXD, but more the single host conundrum. There is a server that runs and maintains a hash map of "transaction" reservations. Once the CPI level transaction completes, the transaction is also released. Additionally, these reservations will time out after a certain amount time. _By default, this is disabled._ (See Tuning for more details. Code is at [throttle](src/cmd/cleanup/main.go).)

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

  > Note that the original reason this was required was that for _every VM_ that LXD launched, the QCOW2 format source image gets converted to a RAW format image. This seems to have been resolved. The solution was simply that the `root` disk was specifying the disk size -- which, apprently, triggers LXD to copy the contents of the source image instead of doing overlay type magic.
