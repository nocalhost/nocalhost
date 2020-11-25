#!/bin/bash
kubectl delete ns test-manifest-02 >> /dev/null
kubectl create ns test-manifest-02
nhctl uninstall test-manifest-02 >> /dev/null
nhctl install test-manifest-02 -u https://e.coding.net/codingcorp/nocalhost/mini-bookinfo-noconfig.git --debug -n test-manifest-02 --config config.yaml
if [ "$?" != 0 ]; then
    echo "fail"
    exit 1
fi
nhctl uninstall test-manifest-02
if [ "$?" != 0 ]; then
    echo "fail"
    exit 1
fi
echo "succeed"
