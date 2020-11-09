#!/bin/bash

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

