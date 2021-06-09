# NOCALHOST Changelog

## 0.4.9 (2021-06-04)

#### :bug:  &nbsp; Bug Fixed

**IDEA Plugin**

- Fixed: replace snakeyaml with nhctl yaml on tree rendering
- Fixed: fail to start dev mode on windows
- Fixed: check process termination before sending ctrl+c
- Fixed: fail to create kubeconfig on windows

**VSCode Plugin**

- Fixed: the server cluster query log issue
- Fixed: multiple download boxes issue
- Fixed: add sudo while starting port forward for port less than 1024
  
#### :muscle: &npsp; Improvement & Refactor

**VSCode Plugin**

- Remove kubectl dependency

**nhctl**

- `nhctl` supports DaemonSet to enter DevMode

**IDEA Plugin**

- Supports DaemonSet enter DevMode
- Supports to check server version when listing applications
- Supports use application type from server
- Add open project action for workloads in dev mode