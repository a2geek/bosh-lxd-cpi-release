# Deploying release version of LXD CPI

> Most of these steps are only done one time, so it's not as complicated as it appears! Adjust as required.

1. Get to your work directory. Or create one for experimentation.

2. Install LXD. Current development environments are installed via [Snap](https://snapcraft.io/lxd), version `5.21/stable`.

3. Configure LXD.

    Sample run from my desktop work environment:

    ```shell
    $ lxd init
    Would you like to use LXD clustering? (yes/no) [default=no]: 
    Do you want to configure a new storage pool? (yes/no) [default=yes]: 
    Name of the new storage pool [default=default]: 
    Name of the storage backend to use (ceph, dir, lvm, powerflex, zfs, btrfs) [default=zfs]: 
    Create a new ZFS pool? (yes/no) [default=yes]: 
    Would you like to use an existing empty block device (e.g. a disk or partition)? (yes/no) [default=no]: yes
    Path to the existing block device: /dev/sda
    Would you like to connect to a MAAS server? (yes/no) [default=no]: 
    Would you like to create a new local network bridge? (yes/no) [default=yes]: 
    What should the new bridge be called? [default=lxdbr0]: 
    What IPv4 address should be used? (CIDR subnet notation, “auto” or “none”) [default=auto]: 
    What IPv6 address should be used? (CIDR subnet notation, “auto” or “none”) [default=auto]: none
    Would you like the LXD server to be available over the network? (yes/no) [default=no]: yes
    Address to bind LXD to (not including port) [default=all]: 
    Port to bind LXD to [default=8443]: 
    Would you like stale cached images to be updated automatically? (yes/no) [default=yes]: 
    Would you like a YAML "lxd init" preseed to be printed? (yes/no) [default=no]: 
    ```

    Note 1: `/dev/sda` is a spare SATA HDD that was laying around. (Primary disk is NVME and it is low on space!) `sgdisk --zap-all /dev/sda` was used to wipe all partitions.

    Note 2: IPv6 was disabled since I'm not using IPv6. This appears to be optional as BOSH has some support for IPv6.

4. Install BOSH. See [Installing the CLI](https://bosh.io/docs/cli-v2-install/).

5. Create a certificate for deployment.

    This can be done using BOSH:

    ```shell
    $ cat > certgen.yml << EOF
    variables:
    - name: boshhost_ca
    type: certificate
    options:
        is_ca: true
        common_name: boshhost
        duration: 3650
    - name: boshauth
    type: certificate
    options:
        ca: boshhost_ca
        common_name: ((internal_ip))
        alternative_names: [((internal_ip))]
    EOF
    $ bosh interpolate certgen.yml -v internal_ip=10.245.169.5 --vars-store boshcerts.yml
    variables: []

    Succeeded
    $ bosh interpolate boshcerts.yml --path /boshauth/certificate > bosh-client.crt
    $ bosh interpolate boshcerts.yml --path /boshauth/private_key > bosh-client.key
    ```

    Note: To check the network address range for the bridge, you can use `lxc network list` and then decide what the IP address will be for BOSH itself.

6. Allow BOSH to sign into LXD with the certificates:

    ```shell
    $ lxc config trust add --name "bosh@<shostname>" ./bosh-client.crt 
    $ lxc config trust list
    +--------+-----------------+--------------+--------------+-----------------------------+-----------------------------+
    |  TYPE  |      NAME       | COMMON NAME  | FINGERPRINT  |         ISSUE DATE          |         EXPIRY DATE         |
    +--------+-----------------+--------------+--------------+-----------------------------+-----------------------------+
    | client | bosh@<hostname> | 10.245.169.1 | 330275849625 | Sep 8, 2024 at 5:57pm (UTC) | Sep 8, 2025 at 5:57pm (UTC) |
    +--------+-----------------+--------------+--------------+-----------------------------+-----------------------------+
    ```

7. Checkout a local copy of [bosh-deployment](https://github.com/cloudfoundry/bosh-deployment) and [bosh-lxd-cpi-release](https://github.com/a2geek/bosh-lxd-cpi-release).

    ```shell
    $ git clone https://github.com/a2geek/bosh-lxd-cpi-release.git
    Cloning into 'bosh-lxd-cpi-release'...
    remote: Enumerating objects: 6345, done.
    remote: Counting objects: 100% (291/291), done.
    remote: Compressing objects: 100% (226/226), done.
    remote: Total 6345 (delta 66), reused 146 (delta 39), pack-reused 6054 (from 1)
    Receiving objects: 100% (6345/6345), 10.37 MiB | 15.07 MiB/s, done.
    Resolving deltas: 100% (1859/1859), done.
    $ git clone https://github.com/cloudfoundry/bosh-deployment.git
    Cloning into 'bosh-deployment'...
    remote: Enumerating objects: 21743, done.
    remote: Counting objects: 100% (779/779), done.
    remote: Compressing objects: 100% (300/300), done.
    remote: Total 21743 (delta 503), reused 752 (delta 479), pack-reused 20964 (from 1)
    Receiving objects: 100% (21743/21743), 2.82 MiB | 6.27 MiB/s, done.
    Resolving deltas: 100% (14712/14712), done.
    ```

8. Setup the BOSH configuration file:

    > Note: Be certain that `internal_ip` matches the certificate that was generated earlier.

    ```shell
    $ cat > config.yml << EOF
    lxd_project_name: default
    lxd_profile_name: default
    lxd_network_name: lxdbr0
    lxd_storage_pool_name: default
    lxd_url: https://<server-ip-or-name>:8443
    lxd_insecure: true
    internal_cidr: 10.245.169.1/24
    internal_gw: 10.245.169.1
    internal_ip: 10.245.169.5
    EOF
    ```

    > If you use the server _name_ (instead of IP), be certain to also configure your internal DNS. Otherwise BOSH won't be able to find LXD.

9. Deploy the BOSH director!

    ```shell
    $ bosh create-env bosh-deployment/bosh.yml \
        --ops-file=bosh-lxd-cpi-release/cpi.yml \
        --ops-file=bosh-deployment/bbr.yml \
        --ops-file=bosh-deployment/uaa.yml \
        --ops-file=bosh-deployment/credhub.yml \
        --ops-file=bosh-deployment/jumpbox-user.yml \
        --vars-file=config.yml \
        --var=director_name=lxd \
        --state=state.json \
        --vars-store=creds.yml \
        --var-file=lxd_client_cert=bosh-client.crt \
        --var-file=lxd_client_key=bosh-client.key
    Deployment manifest: '.../work/bosh-deployment/bosh.yml'
    Deployment state: 'state.json'

    <snip>

    Finished deploying (00:07:57)

    Cleaning up rendered CPI jobs... Finished (00:00:00)

    Succeeded
    ```

10. To SSH into the BOSH director, grab the secrets from `creds.yml`...

    ```shell
    $ bosh interpolate creds.yml --path /jumpbox_ssh/private_key > jumpbox.pk
    $ chmod 600 jumpbox.pk
    $ internal_ip=$(bosh interpolate config.yml --path /internal_ip)
    $ ssh-keygen -f ~/.ssh/known_hosts -R ${internal_ip}
    # Host 10.245.169.5 found: line 64
    .../.ssh/known_hosts updated.
    Original contents retained as .../.ssh/known_hosts.old
    $ ssh -i jumpbox.pk jumpbox@${internal_ip}
    <snip>
    bosh/0:~$ sudo -i
    bosh/0:~# monit summary
    The Monit daemon 5.2.5 uptime: 21m 

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
    System 'system_f5c2f518-92d9-4b0b-4b3e-1d5090606b04' running
    ```

11. To operate the BOSH director, the following will configure the CLI. And then just configure and deploy like normal!

    > Note that the LXD CPI uses the OpenStack "full" stemcell.

    ```shell
    $ export BOSH_CLIENT=admin
    $ export BOSH_CLIENT_SECRET=$(bosh int creds.yml --path /admin_password)
    $ export BOSH_CA_CERT=$(bosh int creds.yml --path /director_ssl/ca)
    $ export BOSH_ENVIRONMENT=$(bosh int config.yml --path /internal_ip)
    $ bosh env
    Using environment '10.245.169.5' as client 'admin'

    Name               lxd  
    UUID               516f8ee1-df88-49ac-ac33-a1145cf9253f  
    Version            280.1.5 (00000000)  
    Director Stemcell  -/1.531  
    CPI                lxd_cpi  
    Features           config_server: enabled  
                       local_dns: enabled  
                       snapshots: disabled  
    User               admin  

    Succeeded
    ```
