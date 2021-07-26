# NOCALHOST Changelog

## 0.4.14 (2021-07-07)

### nhctl

#### :bug:  &nbsp; Bug Fixed

- Fixed statefulset port-forward failure issue.
- Fixed the issue Firewall alert will be triggered when update nhctl or restart Daemon in Windows system
- Fixed the issue where null pointer caused the daemon to exit

#### :muscle: &npsp; Improvement & Refactor

- Disable the execution of cronjob tasks when cronjob enters DevMode
- Execute ``helm repo update`` command before install helm application
- Combine all ``waiting job init_container`` to one
- ``nocalhost config`` supports complex (any form of yaml) helm values
- Supports WEB/Plugin status token refresh
- Add terminal reconnection features, support automatic reconnection after computer sleep and network disconnection
