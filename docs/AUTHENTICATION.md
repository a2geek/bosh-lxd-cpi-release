# Authentication

This CPI uses the LXD certificate authentication using [BOSH generated certificates](https://bosh.io/docs/director-certs/#generate).

Adding the (future) BOSH connection certificate is pretty simple:

```bash
$ lxc config trust add --name "bosh@<hostname>" ./bosh-client.crt
$ lxc config trust list
+--------+-----------------+-------------+--------------+------------------------------+------------------------------+
|  TYPE  |       NAME      | COMMON NAME | FINGERPRINT  |          ISSUE DATE          |         EXPIRY DATE          |
+--------+-----------------+-------------+--------------+------------------------------+------------------------------+
| client | bosh@<hostname> | bosh        | 4e39a923c420 | Jul 20, 2024 at 4:58pm (UTC) | Jul 18, 2034 at 4:58pm (UTC) |
+--------+-----------+-------------+--------------+------------------------------+--------+-----------+-------------+--------------+------------------------------+------------------------------+
```

If you develop on a remote machine, you can setup the LXC CLI to point to the remote server.

1. Use the `lxc remote add...` commands as described [here](https://documentation.ubuntu.com/lxd/en/latest/howto/server_expose/#authenticate-with-the-lxd-server).
2. Switch the default remote with `lxc remote switch <name>`.
