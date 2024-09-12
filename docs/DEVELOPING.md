# Developing (or Tinkering!)

> I have been using both Linux and Mac OS X to deploy. WSL should be ok as well. Just not native Windows due to the Unix assumptions in path names.

If you wish to tinker with this, here is how I'm working with the tools.

First, create a handy environment script (I call it `env.sh`, and it's in `.gitignore` so no secrets are ever at risk).

```bash
export LXD_URL="https://<server>:8443"
export LXD_INSECURE="true"
export LXD_CLIENT_CERT="$PWD/bosh-client.crt"
export LXD_CLIENT_KEY="$PWD/bosh-client.key"
#export BOSH_LOG_LEVEL="DEBUG"
export BOSH_JUMPBOX_ENABLE="yes, please"
export POSTGRES_DIR="<...>/postgres-release"
export BOSH_DEPLOYMENT_DIR="<...>/bosh-deployment"
export CONCOURSE_DIR="<...>/concourse-bosh-deployment"
export CF_DEPLOYMENT_DIR="<...>/cf-deployment"
```

This script can then be source in with `source ./env.sh`.

Second, the `util.sh` script is setup to be sourced in. Note that it must be an actual Bash shell (and not something from an IDE like VS Code).

```bash
$ source ./util.sh 
'util' now available.
```

> Note: 'util' assumes you are in the root of this project.

Run alone to get a list of commands (not fancy but functional):

```bash
$ util
Subcommands:
- capture-requests
- cloud-config
- deploy-bosh
- deploy-cf
- deploy-concourse
- deploy-postgres
- destroy
- final-release
- fix-blobs
- help
- runtime-config
- stress-test
- upload-releases
- upload-stemcells

Notes:
* This script will detect if it is sourced in and setup an alias.
  (^^ does not work from IDE such as VS Code)
* Creds are placed into the 'creds/' folder.

Useful environment variables to export...
- BOSH_LOG_LEVEL (set to 'debug' to capture all bosh activity including request/response)
- BOSH_JUMPBOX_ENABLE (set to any value enable jumpbox user)
- BOSH_SNAPSHOTS_ENABLE (set to any value to enable snapshots)
- LXD_URL (set to HTTPS url of LXD server - not localhost)
- LXD_INSECURE (default: false)
- LXD_CLIENT_CERT (set to path of LXD TLS client certificate)
- LXD_CLIENT_KEY (set to path of LXD TLS client key)
- BOSH_DEPLOYMENT_DIR (default: ${HOME}/Documents/Source/bosh-deployment)
- BOSH_PACKAGE_GOLANG_DIR (default ../bosh-package-golang-release)
- CONCOURSE_DIR when deploying Concourse
- POSTGRES_DIR when deploying Postgres

Configuration values...
- internal_ip:              10.245.0.11
- lxd_network_name:         boshdevbr0
- lxd_profile_name:         default
- lxd_project_name:         boshdev
- lxd_storage_pool_name:    default

Currently set environment variables...
- BOSH_DEPLOYMENT_DIR:      <...>/bosh-deployment
- CONCOURSE_DIR:            <...>/concourse-bosh-deployment
- LXD_CLIENT_CERT:          <...>/bosh-lxd-cpi-release/bosh-client.crt
- LXD_CLIENT_KEY:           <...>/bosh-lxd-cpi-release/bosh-client.key
- LXD_INSECURE:             true
- LXD_URL:                  https://<server>:8443
- POSTGRES_DIR:             <...>/postgres-release

```

Most likely you want to deploy the bosh director:

```bash
$ util deploy_bosh
<lots of output>
```

... and hopefully this works! This particular deploy operation will create a `creds/jumpbox.pk` so an SSH connection can be established:

```bash
$ ssh -i creds/jumpbox.pk jumpbox@10.245.0.11
Unauthorized use is strictly prohibited. All access and activity
is subject to logging and monitoring.
Welcome to Ubuntu 22.04.4 LTS (GNU/Linux 5.15.0-117-generic x86_64)

 * Documentation:  https://help.ubuntu.com
 * Management:     https://landscape.canonical.com
 * Support:        https://ubuntu.com/pro
Last login: Wed Aug 21 23:52:09 UTC 2024 from 192.168.1.254 on pts/0
Last login: Wed Aug 21 23:52:12 2024 from 192.168.1.254
To run a command as administrator (user "root"), use "sudo <command>".
See "man sudo_root" for details.

bosh/0:~$ sudo -i
bosh/0:~# monit summary
The Monit daemon 5.2.5 uptime: 3h 51m 

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
System 'system_0db11f4b-1cfc-47a8-5dd5-6f13a053d7b7' running
```

... and finally, to operate the BOSH director, there is a handy script that sets up the BOSH environment variables for access:

```bash
$ source scripts/bosh-env.sh 
$ bosh deployments
Using environment '10.245.0.11' as client 'admin'

Name       Release(s)                       Stemcell(s)                                     Team(s)  
cf         binary-buildpack/1.1.11          bosh-openstack-kvm-ubuntu-jammy-go_agent/1.506  -  
           bosh-dns/1.38.2                                                                    
           bosh-dns-aliases/0.0.4                                                             
           bpm/1.2.19                                                                         
           capi/1.181.0                                                                       
           cf-cli/1.63.0                                                                      
           cf-networking/3.46.0                                                               
           cf-smoke-tests/42.0.146                                                            
           cflinuxfs4/1.95.0                                                                  
           credhub/2.12.74                                                                    
           diego/2.99.0                                                                       
           dotnet-core-buildpack/2.4.27                                                       
           garden-runc/1.53.0                                                                 
           go-buildpack/1.10.18                                                               
           java-buildpack/4.69.0                                                              
           log-cache/3.0.11                                                                   
           loggregator/107.0.14                                                               
           loggregator-agent/8.1.1                                                            
           nats/56.19.0                                                                       
           nginx-buildpack/1.2.13                                                             
           nodejs-buildpack/1.8.24                                                            
           php-buildpack/4.6.18                                                               
           pxc/1.0.28                                                                         
           python-buildpack/1.8.23                                                            
           r-buildpack/1.2.11                                                                 
           routing/0.297.0                                                                    
           ruby-buildpack/1.10.13                                                             
           silk/3.46.0                                                                        
           staticfile-buildpack/1.6.12                                                        
           statsd-injector/1.11.40                                                            
           uaa/77.9.0                                                                         
concourse  backup-and-restore-sdk/1.18.119  bosh-openstack-kvm-ubuntu-jammy-go_agent/1.506  -  
           bosh-dns/1.38.2                                                                    
           bpm/1.2.16                                                                         
           concourse/7.11.2                                                                   
           postgres/48                                                                        
postgres   bosh-dns/1.38.2                  bosh-openstack-kvm-ubuntu-jammy-go_agent/1.506  -  
           postgres/52                                                                        

3 deployments

Succeeded
```
