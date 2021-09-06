#!/bin/bash

# if nocalhost-dep exist but create cert again, nocalhost-dep will use old cert and will fail to decode body
#if [[ `kubectl get deployment -n nocalhost-reserved -o jsonpath='{.items[*].metadata.labels.app}' -l app=nocalhost-dep` == "nocalhost-dep" ]]; then
#    echo "nocalhost-dep already install, exit now...."
#    exit
#fi

# enter workdir
cd /nocalhost || exit 1

# replace namespace
if [[ -n ${DEP_NAMESPACE} ]] ; then
    echo "replace namespace to ${DEP_NAMESPACE}"
    sed -i "s|name: nocalhost-reserved|name: ${DEP_NAMESPACE}|" ./webhook/namespace.yaml
    sed -i "s|namespace: nocalhost-reserved|namespace: ${DEP_NAMESPACE}|" ./webhook/sa.yaml
    sed -i "s|name: nocalhost-reserved-role-binding|name: nocalhost-reserved-role-binding-${DEP_NAMESPACE}|" ./webhook/sa.yaml
fi

## create namespace
kubectl apply -f ./webhook/namespace.yaml

## create sa
kubectl apply -f ./webhook/sa.yaml

# run and create admission webhook cert shell
source ./cert.sh

# replace CA_BUNDLE
cat ./webhook/mutating-webhook.yaml | ./webhook-patch-ca-bundle.sh > ./webhook/mutating-webhook-ca-bundle.yaml

# replace namespace
if [[ -n ${DEP_NAMESPACE} ]] ; then
    echo "replace namespace to ${DEP_NAMESPACE}"
    sed -i "s|namespace: nocalhost-reserved|namespace: ${DEP_NAMESPACE}|" ./webhook/mutating-webhook-ca-bundle.yaml
    sed -i "s|namespace: nocalhost-reserved|namespace: ${DEP_NAMESPACE}|" ./webhook/sidecar-configmap.yaml
    sed -i "s|namespace: nocalhost-reserved|namespace: ${DEP_NAMESPACE}|" ./webhook/service.yaml
    sed -i "s|namespace: nocalhost-reserved|namespace: ${DEP_NAMESPACE}|" ./webhook/deployment.yaml
fi

# apply MutatingWebhookConfiguration
kubectl apply -f ./webhook/mutating-webhook-ca-bundle.yaml

# apply admission webhook
kubectl apply -f ./webhook/sidecar-configmap.yaml
kubectl apply -f ./webhook/service.yaml

# sed dep docker image version
echo "dep version is"${DEP_VERSION}
sed -i "s|image:.*$|image: codingcorp-docker.pkg.coding.net/nocalhost/public/nocalhost-dep:${DEP_VERSION}|" ./webhook/deployment.yaml

if [[ -n ${DEP_IMAGE} ]] ; then
    echo "replace image to: ${DEP_IMAGE}"
    sed -i "s|image:.*$|image: ${DEP_IMAGE}|" ./webhook/deployment.yaml
fi

kubectl apply -f ./webhook/deployment.yaml

# waiting for dep pod ready
ns=${DEP_NAMESPACE:-"nocalhost-reserved"}
echo "waiting for dep pod ready"
bash ./wait-for/wait_for.sh pod -lapp=nocalhost-dep -n "${ns}"

# done
echo "admission webhook install done!"
