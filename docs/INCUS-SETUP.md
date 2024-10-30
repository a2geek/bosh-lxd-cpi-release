# Incus Setup

The Incus Setup is very like the LXD setup. General instructions:

1. Run `incus init` and be certain to enable network connectivity.

2. Generate a certificate (and private key) for the connection. Add to Incus with a command similar to:

    ```bash
    incus config trust add-certificate --name "bosh@odin" ./bosh-client.crt
    ```

3. Deploy a BOSH director. Be certain to select the [`enable-incus.yml`](ops/enable-incus.yml) ops file.

4. Optionally, enable the Incus agent for BOSH with the [`enable-incus-agent.yml`](ops/enable-incus-agent.yml) ops file. See warning below.

5. Optionally, enable the Incus agent for ALL VMs by adding a BOSH config with the [`enable-incus-agent-config.yml`](manifests/enable-incus-agent-config.yml) config manifest. See warning below.

## Note on enabling the agents

By enabling the Incus agent, any ID with appropriate permissions _to Incus_ can connect to ANY VM as `root`. No passwords. This is great for development or a lab or at home. ___But not production.___

```bash
$ bosh vms
Using environment '10.161.172.4' as client 'admin'

Task 9. Done

Deployment 'concourse'

Instance                                     Process State  AZ  IPs            VM CID                                   VM Type  Active  Stemcell  
db/b6d785c1-b78e-4ff5-a4ef-a209f829905f      running        z1  10.161.172.11  vm-01ae91cd-0385-4f72-658a-b9496717212b  default  true    bosh-openstack-kvm-ubuntu-jammy-go_agent/1.621  
web/9fafe0ab-ed07-4c1a-a600-8d7b2b5f12f7     running        z1  10.161.172.5   vm-d621a271-a1bd-4cf1-628b-33145d446393  default  true    bosh-openstack-kvm-ubuntu-jammy-go_agent/1.621  
worker/d5ff362f-6e45-4e39-97ab-e5fced9f1f13  running        z1  10.161.172.12  vm-93907942-b1da-4bf6-743f-8bc2bd07cfa2  default  true    bosh-openstack-kvm-ubuntu-jammy-go_agent/1.621  

3 vms

Succeeded
$ incus exec vm-01ae91cd-0385-4f72-658a-b9496717212b -- bash
db/b6d785c1-b78e-4ff5-a4ef-a209f829905f:~# monit summary
The Monit daemon 5.2.5 uptime: 13m 

Process 'postgres'                  running
Process 'pg_janitor'                running
Process 'bosh-dns'                  running
Process 'bosh-dns-resolvconf'       running
Process 'bosh-dns-healthcheck'      running
System 'system_f14a1f83-3dc9-4652-bbb4-d655f0b17fc7' running
db/b6d785c1-b78e-4ff5-a4ef-a209f829905f:~# exit
```