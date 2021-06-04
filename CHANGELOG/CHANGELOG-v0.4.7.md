# NOCALHOST Changelog

## 0.4.7 (2021-06-04)

#### :bug:  &nbsp; Bug Fixed

**nhctl**

- Supports to display Helm app in IDEA 

**IDEA Plugin**

- Optimize performance
- Fixed: extend sync status update interval
- Fixed: project settings saving

**VSCode Plugin**

- Fixed "Passing undefined parameters when cleaning up all pvc" issue
  
#### :muscle: &npsp; Improvement & Refactor

- Supports read nocalhostConfig by cm has the highest priority
- Supports smooth upgrade of Windows
- Supports dev parameters of `configmap` type
- Supports read configuration file from source code
- Supports read configuration file from `configmap`
- Supports user values file when using helm upgrade
