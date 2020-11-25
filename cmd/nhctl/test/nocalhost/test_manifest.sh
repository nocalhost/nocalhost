#!/bin/bash
kubectl delete ns test-manifest-01 >> /dev/null
kubectl create ns test-manifest-01
nhctl uninstall test-manifest-01 >> /dev/null
nhctl install test-manifest-01 -u https://e.coding.net/codingcorp/nocalhost/mini-bookinfo-manifest.git --debug -n test-manifest-01
if [ "$?" != 0 ]; then
    echo "fail"
    exit 1
fi
nhctl uninstall test-manifest-01
if [ "$?" != 0 ]; then
    echo "fail"
    exit 1
fi

echo "succeed"
