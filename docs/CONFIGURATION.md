# CPI Configuration Options

> Note that this is manually maintained, so looking at the [spec](../jobs/lxd_cpi/spec) may be of use.

## Static CPI configuration (when deploying BOSH)

### Server options

| Option | Description |
| --- | --- |
| `lxd_cpi.server.type` | Type of server to connect to: `"lxd"` (default) or `"incus"`. |
| `lxd_cpi.server.url` | URL of LXD or Incus server (cannot be `localhost` as this URL is used to both create the BOSH director and by BOSH itself within a VM). Example: `"https://server:8443"` |
| `lxd_cpi.server.insecure_skip_verify` | Indicates if the SSL connection should be validated (set to false, the default, for self-signed certificates). |
| `lxd_cpi.server.tls_client_cert` | Client public certificate to use for TLS connection. |
| `lxd_cpi.server.tls_client_key` | Client private key to use for TLS connection. |
| `lxd_cpi.server.bios_path` | Location and name of BIOS to use (default `"bios-256k.bin"`). |

### LXD/Incus options

| Option | Description |
| --- | --- |
| `lxd_cpi.server.project_name` | Name of project with which to place VMs and stemcells (default `default`). |
| `lxd_cpi.server.profile_name` | Name of profile with which to build VMs (default `default`). |
| `lxd_cpi.server.target` | Name of location/target to deploy VM when using cluster; name as LXD shows the host in a cluster. Example `"lxdhost1"` to target a specific machine, `"@group1"` to target a cluster group. |
| `lxd_cpi.server.managed_network_assignment` | If the LXD network is 'managed' (runs a DHCP server), indicates strategy of IP assignment. Primarily applies to BOSH 'manual' network: `static` (default) or `dhcp`. Note that for a 'dynamic' network, `dhcp` must be used. Also note that if you are using the Ubuntu Fan specifically, `dhcp` is what you want as there is extra routing that the static configuration does not include. |
| `lxd_cpi.server.network_name` | Name of network with which to place VMs. Example `lxdbr0`. |
| `lxd_cpi.server.storage_pool_name` | Name of storage pool with which to place disks. (default `default`). |

### BOSH Agent configuration

> These likely can be left alone since BOSH injects these values.

| Option | Description |
| --- | --- |
| `lxd_cpi.agent.mbus` | Mbus URL used by deployed BOSH agents. BOSH value: `nats://((internal_ip)):4222`; example: `"nats://nats:nats-password@10.254.50.4:4222"`. |
| `lxd_cpi.agent.ntp` | NTP configuration used by deployed BOSH agents. |

> These likely can be left alone as it impacts how the BOSH Agent is configured. If you are experimenting with other stemcells, these are the configuration options.

| Option | Description |
| --- | --- |
| `lxd_cpi.agent_config.type` | Agent configuration type for stemcell: `FAT32` or `CDROM` (default). |
| `lxd_cpi.agent_config.label` | Disk label for configuration device (default `config-2`). |
| `lxd_cpi.agent_config.metadata_path` | Path to the metadata file for Agent configuration (default `ec2/latest/meta-data.json`). |
| `lxd_cpi.agent_config.userdata_path` | Path to the userdata file for Agent configuration (default `ec2/latest/user-data`). |
| `lxd_cpi.agent_config.filestore_path` | Path to where agent data should be stored on the Bosh VM (default `/var/vcap/store/agent-data`). |

### Throttle configuration

> These are unlikely to be required. Left overs from early development of the CPI.

| Option | Description |
| --- | --- |
| `lxd_cpi.throttle_config.enabled` | Throttle master switch: `true` or `false` (default). |
| `lxd_cpi.throttle_config.path` | Unix socket location (default `/var/vcap/sys/run/lxd_cpi/throttle.sock`). |
| `lxd_cpi.throttle_config.limit` | Maximum number of processes to allow concurrently (default `4`). |
| `lxd_cpi.throttle_config.hold` | Maximum length of lock hold (default `2m`). |
