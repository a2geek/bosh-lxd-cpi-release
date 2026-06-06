# Error messages

Some descriptions of error messages and what they indicate.

| Message | Notes
| :--- | :--- |
| not connected with trusted connection; using 'untrusted' | The LXD/Incus connection certificate either does not exist or has expired. |
| Error: Failed to create storage pool "remote": Pool 'lxd' in cluster 'ceph' seems to be in use by another LXD instance | It appears that LXD may have difficulty with remote storage and concurrency. Since bosh creates multiple VMs at the same time, I suspect this prevents remote storage until resolved. [Possible ticket to follow LXD #15315](https://github.com/canonical/lxd/issues/15315) |