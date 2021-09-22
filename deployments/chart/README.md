# Nocalhost
The term Nocalhost originates from No Local, which is a cloud-native development tool based on IDE, provides realtime cloud native application developing experience.  

When developing cloud-based application in Nocalhost, any code changes can immediately take effects in remote side, and there is no need to rebuild a new image. This can shorten the entire development feedback loops and massively improve R&D efficiency.

## Prerequisites
- Kubernetes 1.16+
- Helm 3+

## Get Repo Info

```console
helm repo add nocalhost "https://nocalhost-helm.pkg.coding.net/nocalhost/nocalhost"
helm repo update
```

_See [helm repo](https://helm.sh/docs/helm/helm_repo/) for command documentation._

## Install Chart

```console
# Helm
$ helm install [RELEASE_NAME] nocalhost/nocalhost -n nocalhost --create-namespace
```

_See [helm install](https://helm.sh/docs/helm/helm_install/) for command documentation._

## Dependencies

By default this chart installs additional, dependent charts:

- mariadb

_See [helm dependency](https://helm.sh/docs/helm/helm_dependency/) for command documentation._

## Uninstall Chart

```console
# Helm
$ helm uninstall [RELEASE_NAME] -n nocalhost
```

This removes all the Kubernetes components associated with the chart and deletes the release.

_See [helm uninstall](https://helm.sh/docs/helm/helm_uninstall/) for command documentation._

## Upgrading Chart

```console
# Helm
$ helm upgrade [RELEASE_NAME] [CHART] -n nocalhost --install
```

_See [helm upgrade](https://helm.sh/docs/helm/helm_upgrade/) for command documentation._
