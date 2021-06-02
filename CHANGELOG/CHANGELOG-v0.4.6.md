# NOCALHOST Changelog

## 0.4.6 (2021-06-02)

#### :rocket:  &nbsp; New Feature

**VSCode Plugin**

- Support clearing the pvc of DevSpace

#### :bug:  &nbsp; Bug Fixed

**nhctl**

- Fixed the "ignore pattern" issue
- Cluster config supports preview on the web
- Sorted the context of kubeconfig returned by the API

**IDEA Plugin**

- Fixed "mute application not found error while refreshing tree" issue
- Fixed "add controller type to nhctl command while starting run/debug" issue 
- Fixed "make resume and override sync status asynchronously" issue
- Fixed "NPE while nhctl getting resources" issue
- Fixed "lock downloading nhctl file" issue
- Fixed: "frozen after dev start" issue

**VSCode Plugin**

- Fixed "lock downloading nhctl file" issue
- Fixed "statefulSet state display error" issue
- Fixed "keep configuration notes" issue
- Fixed "statefulSet file sync" issue

#### :muscle: &npsp; Improvement & Refactor

**nhctl**

- Optimized the tree display and features

**IDEA Plugin**

- Reactor: Replace kubectl with nhctl

