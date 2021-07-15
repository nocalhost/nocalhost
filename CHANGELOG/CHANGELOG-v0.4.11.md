# NOCALHOST Changelog

## 0.4.11 (2021-06-29)

#### :bug:  &nbsp; Bug Fixed

**nhctl**

**JetBrains Plugin**

- Fixed: error causes by empty server version
- Fixed: complete config read only conditions
- Fixed: filter application config file by parsing file content and checking application name
  
#### :muscle: &npsp; Improvement & Refactor

**JetBrains Plugin**

- Supports rename standalone cluster
- Supports install standalone app
- Supports open multiple consoles for the same container
- Supports Job/CronJob/Pod enter DevMode
- Start dev mode without waiting for pods ready

**VSCode Plugin**

- Supports install standalone app
- Supports Job/CronJob/Pod enter DevMode
- Supports Application Port-Forward management
- Supports VSCode Workspace trust

**nhctl**

- Optimize test case
- Supports annotations configuration
- Supports Helm application install/uninstall in Nocalhost

