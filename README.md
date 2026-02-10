# BOSH LXD CPI

> Would love PRs if people are still playing with BOSH in 2024!

This is a [BOSH CPI](https://bosh.io/) implementation to support [LXD](https://canonical.com/lxd) and [Incus](https://linuxcontainers.org/incus/introduction/). Both require BIOS boot capability, so this generally is more recent releases of LXD and Incus.

## Requirements

Recent verison of LXD and Incus both support BIOS and UEFI boots. "Jammy" or earlier stemcells will require a BIOS boot while "Noble" or later stemcells can use the UEFI boot.

LXD 5.21, LXD supports BIOS boots for VMs, which all the BOSH Stemcells use. Without this feature, they must be UEFI boot devices and VMs are not an option. Incus support is in place for the 6.0 LTS release, and likely some of the pre-releases.

Note that the (initial) BOSH deployment can be run from Linux, a Mac, and presumably from the Linux on WSL. Once a BOSH director is running, the BOSH CLI can be used from [pretty much anywhere](https://bosh.io/docs/cli-v2-install/).

The current development environments are:

* LXD (`5.21/stable`) has been installed via Snap and [this guide](https://documentation.ubuntu.com/lxd/en/latest/tutorial/first_steps/) was generally followed
* LXD (`6/stable`) has been installed via Snap
* Incus (`6.0.0-1ubuntu0.1`) has been installed via `apt` and [the documentation](https://linuxcontainers.org/incus/docs/main/installing/#installing) was followed for Ubuntu
* [IncusOS](https://linuxcontainers.org/incus-os/) has also been reported as working

## Documentation

* [LXD Setup](docs/LXD-SETUP.md) contains some notes from setting up the LXD development environment.
* [Incus Setup](docs/INCUS-SETUP.md) contains some notes regarding Incus.
* [Authentication](docs/AUTHENTICATION.md) shows an example for how to setup the certificate authentication for LXD.
* [Developing](docs/DEVELOPING.md) covers how the tools in this repository are used.
* [Deploying a BOSH director](docs/DEPLOYING.md) walks through a sample deployment for a fresh install (of nearly everything).
* LXD/Incus [CPI configuration options](docs/CONFIGURATION.md).
* Supplied [option files](ops/README.md).
* [Working with Genesis](docs/GENESIS.md) shows how to deploy to Genesis with a non-standard CPI.

## Current State

All CPI calls have been implemented, including snapshots and IaaS-native disk resizing.

### LXD adjustments

* There is a nightly scheduled process to look for unused configuration disks (`vol-c-<uuid>` format). There are events (at least with ZFS) where the configuration disk sometimes doesn't get detached from the VM. The nightly process simply scans the list of detached configuration disks, tries to delete them, and reports success or error in the log. See [cleanup](src/cmd/cleanup/main.go).
* Throttling. Not so much LXD, but more the single host conundrum. There is a service that runs and maintains a hash map of "transaction" reservations. Once the CPI level transaction completes, the transaction is also released. Additionally, these reservations will time out after a certain amount time. _By default, this is disabled._ (See Tuning for more details. Code is at [throttle](src/cmd/cleanup/main.go).)

### Tuning

> Note that the original reason this was required was that for _every VM_ that LXD launched, the QCOW2 format source image gets converted to a RAW format image. This seems to have been resolved. The solution was simply that the `root` disk was specifying the disk size -- which, apparently, triggers LXD to copy the contents of the source image instead of doing overlay type magic. It is unlikely that you will need this.

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
