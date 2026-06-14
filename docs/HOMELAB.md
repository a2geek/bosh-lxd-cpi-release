# Homelab use

Likely a lot of the use for Incus or LXD will be in the "homelab". This is just a collection of notes, so please add PR's to update!

## Devlopment environments / configuration

| Product | Installation | Instances | Routing |
| :--- | :--- | ---: | :--- |
| LXD 6.x | Snap | 1 | Subnet to host |
| LXD 5.21 LTS | Snap | 3 | Uses the [Ubuntu Fan](https://wiki.ubuntu.com/FanNetworking) for networking. Each dedicated subnet is directed to the proper host. See below. |
| Incus 6.x | apt | 1 | Subnet to host |

## Networking

Be careful of Docker. It mucks with the host routing and you may need to `sudo iptables -A DOCKER-USER -j ACCEPT` which allows IP forwards into and out of the VMs.

For single hosts, networking is literally simple and the router just needs to route to the host. As an example, I setup OpenWrt to route my subnet to the host.

For multiple hosts, using the Ubuntu Fan is a good option. BOSH will need to be configured as "dynamic" to allow LXD/Incus to assign the IP addresses. But routing is pretty straight forward.

Alternatively, with multiple hosts there are products like MicroOVN that appear to share IP address space across hosts.

### Ubuntu Fan

Networking has been two different ways. As of June 2026, the CPI now allows `target` to be added into the cloud config. This allows the use of manual networks where BOSH assigns the IP address. Prior to this, BOSH could assign IP addresses but each VM's placement was random (whatever LXD decided) or manually placed -- in that environment dynamic was better which allowed LXD/Incus to make that determination.

The better option is now using manual with an entry for each host. IP assignment is up to you. See [configuration](CONFIGURATION.md) for a sample (near the bottom of the page).

Otherwise, you can set the cloud config to `dynamic` to allow LXD/Incus to set IP address. Uses [DNS Publisher](https://github.com/a2geek/dns-publisher-release) to publish the IP mappings.

## Disk

For a single host, using local storage in LXD or Incus as the storage medium is fine. Read their notes and note the warnings. Even so, I've had success with BTRFS over ZFS (for whatever reason; my experience with ZFS would sometimes hold onto a volume until a reboot).

For a dual host cluster, you are likely "stuck" with using local storage. The LXD CPI will try to colocate the disks to the proper server. However, if LXD gets super busy, it may struggle to get the storage copied. (Incus had this as a bug that got fixed, I didn't find a related bug for LXD.) However, if the host gets a "cool down" it'll likely be ok. 
> My story: I was checking if a Cloud Foundry upgrade would be a problem, so I deployed the same version into my dev environment and _immediately_ upgraded it. It failed on a storage volume copy. So I tried it clean (deleted the deployment etc) and deployed the starting version but it was late, so that's where I left it. In the morning, I deployed the target version and it worked just fine. The storage volume copied just fine. ¯\(ツ)/¯

If you have more than 3 hosts in the cluster, look into MicroCeph (or even MicroCloud). Also review the disk storage options for LXD or Incus at that time. If you pick a network savvy disk, the entire "copy to host" piece gets skipped.

## Software clustering

Just a note that software clustering in a home lab, especially with a single host can be more trouble than it's worth. Early on, I deployed a normal Cloud Foundry
to a workstation, including the MySQL cluster. 3 hosts kept in sync. However, as soon as a power outage, or an OS upgrade that requires a reboot, it became a challenge
because there was no cluster to join when MySQL restarted. Feel free to give it a whirl, but be cautious!

