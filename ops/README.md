# Ops Files

These ops files are supplied to help configure the BOSH director.

| Name | Description |
| :--- | :--- |
| [`enable-incus-agent.yml`](enable-incus-agent.yml) | Enable the the [Incus Agent](https://linuxcontainers.org/incus/docs/main/howto/instances_create/#install-the-incus-agent-into-virtual-machine-instances) to be run in the BOSH VM. If wanted for _all_ VMs, also see the [runtime config setting](../manifests/enable-incus-agent-config.yml). |
| [`enable-incus.yml`](enable-incus.yml) | Sets the server type to be `incus`. Required when using Incus. |
| [`enable-lxd-agent.yml`](enable-lxd-agent.yml) | Enable the [LXD Agent](https://documentation.ubuntu.com/lxd/en/latest/howto/instances_create/#install-the-lxd-agent-into-virtual-machine-instances) to be run in the BOSH VM. If wanted for _all_ VMs, also see the [runtime config setting](../manifests/enable-lxd-agent-config.yml). Example of using the runtime config is in [`util.sh`](../util.sh). |
| [`enable-lxd.yml`](enable-lxd.yml) | Sets the server type to be `lxd`. Optional when using LXD (as it is the default). |
| [`enable-snapshots.yml`](enable-snapshots.yml) | Enable [snapshots](https://bosh.io/docs/snapshots/) to be taken by LXD. |
| [`enable-throttle.yml`](enable-throttle.yml) | Enable throtte capabilities. Configuration parameters are `lxd_throttle_limit` allows a certain number of processes while `lxd_throttle_hold` is the maximum time that the hold is in place. (Note that this was written for some disk I/O issues that existed with slower disks. Try the [recommended storage setup](https://documentation.ubuntu.com/lxd/en/latest/reference/storage_drivers/#recommended-setup) instead, using ZFS or BTRFS.) |
| [`local-release.yml`](local-release.yml) | Used for development. Changes the release to come from file specified by `cpi_path`. See [`util.sh`](../util.sh) for an example. |
| [`set-bios-path.yml`](set-bios-path.yml) | Set `lxd_bios_path` to reconfigure the source of the boot BIOS if the default of `bios-256k.bin` needs to be changed. Note that the environment variable `LXD_QEMU_FW_PATH` (where "FW" means firmware) can be setup to point to the BIOS directory. If the LXD snap is used, this appears to be already configured. |
| [`set-director-target.yml`](set-director-target.yml) | Set `director_target` to specify which LXD/Incus host the BOSH Director should be deployed to. |
