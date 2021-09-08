#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

ns=${DEP_NAMESPACE:-"nocalhost-reserved"}

CA_BUNDLE=$(kubectl -n "${ns}" get secrets -o jsonpath="{.items[?(@.metadata.annotations['kubernetes\.io/service-account\.name']=='nocalhost-admin-service-account')].data.ca\.crt}")

export CA_BUNDLE

if command -v envsubst >/dev/null 2>&1; then
    envsubst
else
    sed -e "s|\${CA_BUNDLE}|${CA_BUNDLE}|g"
fi