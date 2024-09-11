# Deploying via Genesis

[Genesis](https://github.com/genesis-community/genesis) is an alternate BOSH "deployment paradigm".

## General strategy

To deploy with a non-standard BOSH CPI, we first need to install the BOSH Genesis Kit, and pick an existing CPI. Since the LXD CPI uses the OpenStack stemcells, we will fake an install on OpenStack.

The BOSH genesis kit allows ops files to customize the deployment, and that capability can be used to alter the CPI configuration.

## Initial setup

```bash
$ genesis init --kit bosh
$ genesis create <environment-name> --create-env
```

During the create for the first BOSH environment, select OpenStack as the CPI. Fill in the OpenStack details with dummy values (`fake`, for instance). For the rest, choose what is wanted/needed for the BOSH deployment.

## Pull out the OpenStack CPI

Since the LXD CPI is likely to change, the first ops file is a stand alone (and likely something that won't change) to remove the OpenStack CPI:

In `ops/remove-openstack.yml`:

```yaml
---
# Remove openstack details
- type: remove
  path: /instance_groups/name=bosh/properties/openstack

- type: remove
  path: /cloud_provider/properties/openstack

- type: remove
  path: /instance_groups/name=bosh/jobs/name=openstack_cpi

- type: remove
  path: /releases/name=bosh-openstack-cpi
```

Modify the `<environment>.yml` file to include the ops file:

```yaml
...
  features:
    - openstack
    - remove-openstack
    - proto
    - bosh-dns-healthcheck
...
```

## Add the LXD CPI

Using the same technique, we can base the CPI addition off of the LXD [`cpi.yml`](../cpi.yml) file that is in the root of this project.

Copy the file over (maybe rename it as `lxd.yml`). Change any variable that would come from the Genesis environment file _or_ from the secrets stored in Vault. Likely, this is what is needed:

```yaml
    lxd: &lxd_settings
      project_name: (( grab params.lxd_project_name ))
      profile_name: (( grab params.lxd_profile_name ))
      network_name: (( grab params.lxd_network_name ))
      storage_pool_name: (( grab params.lxd_storage_pool_name ))
      url: (( grab params.lxd_url ))
      insecure_skip_verify: (( grab params.lxd_insecure ))
      tls_client_cert: (( vault meta.vault "/lxd/creds:lxd_client_cert" ))
      tls_client_key: (( vault meta.vault "/lxd/creds:lxd_client_key" ))
```

Note that the `grab` clauses come from the environment file while the `vault` entries are stored in vault. The determining factor is configuration versus secrets. Secrets go into vault. Configuration go into the environment file.

Update the `<environment>.yml` file to include the new ops file:

```yaml
  features:
    - openstack
    - remove-openstack
    - lxd
    - proto
    - bosh-dns-healthcheck
```

## Store secrets

Load the LXD client certificate and key into Vault via Safe:

```shell
$ safe set secret/<environment>/bosh/lxd/creds lxd_client_key@bosh-client.key lxd_client_cert@bosh-client.crt
```

## A peek at the config file

Finally, a portion of the environment configuration file:

```yaml
  # BOSH on LXD needs to know where the LXD API lives,
  #
  # Openstack credentials are stored in the Vault at
  #   /secret/<environment>/bosh/openstack/creds
  #
  lxd_url: https://<host-ip>:8443
  lxd_insecure: true
  lxd_project_name: default
  lxd_profile_name: default
  lxd_network_name: lxdbr0
  lxd_storage_pool_name: default

  # Unused
  openstack_auth_url: https://fake
  openstack_az: fake
  openstack_default_security_groups: fake
  openstack_flavor: fake
  openstack_region: fake
  openstack_ssh_key: fake
```
