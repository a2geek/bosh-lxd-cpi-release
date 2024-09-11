# Ops Files

These ops files are supplied to help configure the bosh director.

| Name | Description |
| :--- | :--- |
| [`enable-snapshots.yml`](enable-snapshots.yml) | Enable [snapshots](https://bosh.io/docs/snapshots/) to be taken by LXD. |
| [`enable-throttle.yml`](enable-throttle.yml) | Enable throtte capabilities. Configuration parameters are `lxd_throttle_limit` allows a certain number of processes while `lxd_throttle_hold` is the maximum time that the hold is in place. (Note that this was written for some disk I/O issues that existed with slower disks. Try the [recommended storage setup](https://documentation.ubuntu.com/lxd/en/latest/reference/storage_drivers/#recommended-setup).) |
| [`local-release.yml`](local-release.yml) | Used for development. Changes the release to come from file specified by `cpi_path`. See [`util.sh`](../util.sh) for an example. |
| [`set-bios-path.yml`](set-bios-path.yml) | Set `lxd_bios_path` to reconfigure the source of the boot BIOS if the default of `bios-256k.bin` needs to be changed. Note that the environment variable `LXD_QEMU_FW_PATH` (where FW = firmware) can be setup to point to the BIOS directory. If the LXD snap is used, this appears to be already configured. |
