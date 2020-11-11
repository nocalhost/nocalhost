#!/bin/bash

# if nocalhost-dep exist but create cert again, nocalhost-dep will use old cert and will fail to decode body
if [[ `kubectl get deployment -n nocalhost-reserved -o jsonpath='{.items[*].metadata.labels.app}' -l app=nocalhost-dep` == "nocalhost-dep" ]]; then
    echo "nocalhost-dep already install, exit now...."
    exit
fi

# enter workdir
cd /nocalhost
# run and create admission webhook cert shell
source ./cert.sh

# replace CA_BUNDLE
cat ./webhook/mutating-webhook.yaml | ./webhook-patch-ca-bundle.sh > ./webhook/mutating-webhook-ca-bundle.yaml

# apply MutatingWebhookConfiguration
kubectl apply -f ./webhook/mutating-webhook-ca-bundle.yaml

# apply admission webhook
kubectl apply -f ./webhook/sidecar-configmap.yaml
kubectl apply -f ./webhook/service.yaml
kubectl apply -f ./webhook/deployment.yaml

# done
echo "admission webhook install done!"

