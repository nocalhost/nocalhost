#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

ns=${DEP_NAMESPACE:-"nocalhost-reserved"}

CA_BUNDLE=''
for x in $(seq 10); do
    CA_BUNDLE=$(kubectl -n "${ns}" get secrets -o jsonpath='{.data.ca-cert\.pem}')
    if [[ ${CA_BUNDLE} != '' ]]; then
        break
    fi
    sleep 1
done

if [[ ${CA_BUNDLE} != '' ]]; then
    echo "failed to get ca-cert.pem form secret"
    exit 1
fi

export CA_BUNDLE

if command -v envsubst >/dev/null 2>&1; then
    envsubst
else
    sed -e "s|\${CA_BUNDLE}|${CA_BUNDLE}|g"
fi