#!/bin/bash

set -e

ns=${DEP_NAMESPACE:-"nocalhost-reserved"}
secret="nocalhost-sidecar-injector-certs"

CA_BUNDLE=''
for x in $(seq 10); do
    CA_BUNDLE=$(kubectl -n "${ns}" get secrets ${secret} -o jsonpath='{.data.ca-cert\.pem}')
    echo "failed to get ca-cert.pem form secret, try again..."
    if [[ ${CA_BUNDLE} != '' ]]; then
        break
    fi
    sleep 1
done

if [[ ${CA_BUNDLE} == '' ]]; then
    echo "failed to get ca-cert.pem form secret, exit"
    exit 1
fi

echo "replace CA_BUNDLE"
sed -i "s|\${CA_BUNDLE}|${CA_BUNDLE}|g" ./webhook/mutating-webhook-ca-bundle.yaml
