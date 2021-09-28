#!/bin/bash

set -e
# if nocalhost-dep exist but create cert again, nocalhost-dep will use old cert and will fail to decode body
#if [[ `kubectl get deployment -n nocalhost-reserved -o jsonpath='{.items[*].metadata.labels.app}' -l app=nocalhost-dep` == "nocalhost-dep" ]]; then
#    echo "nocalhost-dep already install, exit now...."
#    exit
#fi

# enter workdir
cd /nocalhost || exit 1

match_namespace_label='- key: __match_namespace_label_key__\n          operator: In\n          values: [ \"__match_namespace_label_values__\" ]'
match_namespace_name='- name: MATCH_NAMESPACE\n              value: "__match_namespace_values__"'

if [[ ${DEP_MATCH_WITH} == "namespaceName" && -n ${DEP_NAMESPACE} ]]; then
    match_namespace_name=${match_namespace_name//__match_namespace_values__/${MATCH_NAMESPACE_NAME}}
    sed -i "s|#__MATCH_NAMESPACE__#|${match_namespace_name}|" ./webhook/deployment.yaml
else
    MATCH_NAMESPACE_LABEL_KEY=${MATCH_NAMESPACE_LABEL_KEY:-"env"}
    MATCH_NAMESPACE_LABEL_VALUE=${MATCH_NAMESPACE_LABEL_VALUE:-"nocalhost"}
    match_namespace_label=${match_namespace_label//__match_namespace_label_key__/${MATCH_NAMESPACE_LABEL_KEY}}
    match_namespace_label=${match_namespace_label//__match_namespace_label_values__/${MATCH_NAMESPACE_LABEL_VALUE}}

    sed -i "s|#__NAMESPACE_MATCH_KEY__#|${match_namespace_label}|" ./webhook/mutating-webhook.yaml
fi

if [[ ${DEP_MATCH_WITH} == "namespaceLabel" ]]; then
    # set label on MY_POD_NAMESPACE
    kubectl label namespaces "${MY_POD_NAMESPACE}" "${MATCH_NAMESPACE_LABEL_KEY}=${MATCH_NAMESPACE_LABEL_VALUE}" --overwrite
fi

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
cp ./webhook/mutating-webhook.yaml ./webhook/mutating-webhook-ca-bundle.yaml
source ./webhook-patch-ca-bundle.sh

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
sed -i "s|image:.*$|image: nocalhost-docker.pkg.coding.net/nocalhost/public/nocalhost-dep:${DEP_VERSION}|" ./webhook/deployment.yaml

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
