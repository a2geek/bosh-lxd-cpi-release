# Ops Files

These ops files are supplied to help configure the BOSH director.

| Name | Description |
| :--- | :--- |
| [`enable-default-secureboot.yml`](enable-default-secureboot.yml) | Enable secureboot for the "default" settings. This is intended for [IncusOS](https://linuxcontainers.org/incus-os/), which indicates that it is ready for "modern security features like UEFI Secure Boot". |
| [`enable-incus-agent.yml`](enable-incus-agent.yml) | Enable the the [Incus Agent](https://linuxcontainers.org/incus/docs/main/howto/instances_create/#install-the-incus-agent-into-virtual-machine-instances) to be run in the BOSH VM. If wanted for _all_ VMs, also see the [runtime config setting](../manifests/enable-incus-agent-config.yml). |
| [`enable-incus.yml`](enable-incus.yml) | Sets the server type to be `incus`. Required when using Incus; also sets the `bios_path` to `seabios.bin`. |
| [`enable-lxd-agent.yml`](enable-lxd-agent.yml) | Enable the [LXD Agent](https://documentation.ubuntu.com/lxd/en/latest/howto/instances_create/#install-the-lxd-agent-into-virtual-machine-instances) to be run in the BOSH VM. If wanted for _all_ VMs, also see the [runtime config setting](../manifests/enable-lxd-agent-config.yml). Example of using the runtime config is in [`util.sh`](../util.sh). |
| [`enable-lxd.yml`](enable-lxd.yml) | Sets the server type to be `lxd`. Required when using LXD; also sets the 'bios_path` to `bios-256k.bin`. |
| [`enable-snapshots.yml`](enable-snapshots.yml) | Enable [snapshots](https://bosh.io/docs/snapshots/) to be taken by LXD. |
| [`enable-throttle.yml`](enable-throttle.yml) | Enable throtte capabilities. Configuration parameters are `lxd_throttle_limit` allows a certain number of processes while `lxd_throttle_hold` is the maximum time that the hold is in place. (Note that this was written for some disk I/O issues that existed with slower disks. Try the [recommended storage setup](https://documentation.ubuntu.com/lxd/en/latest/reference/storage_drivers/#recommended-setup) instead, using ZFS or BTRFS.) |
| [`local-release.yml`](local-release.yml) | Used for development. Changes the release to come from file specified by `cpi_path`. See [`util.sh`](../util.sh) for an example. |
| [`remove-instance-config.yml`](remove-instnace-config.yml) | Remove all `instance_config` settings. | 
| [`set-director-target.yml`](set-director-target.yml) | Set `director_target` to specify which LXD/Incus host the BOSH Director should be deployed to. |
| [`set-instance-config.yml`](set-instance-config.yml) | Set `instance_config` to alter the default values (see [spec](../jobs/lxd_cpi/spec) file). |
