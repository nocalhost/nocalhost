#!/bin/bash
APPNAME=test-manifest-02
kubectl delete ns test-manifest-02 >> /dev/null
kubectl create ns test-manifest-02
nhctl uninstall test-manifest-02 >> /dev/null
nhctl install test-manifest-02 -u https://e.coding.net/codingcorp/nocalhost/mini-bookinfo-noconfig.git --debug -n test-manifest-02 --type manifest --resource-path manifest/templates
if [ "$?" != 0 ]; then
    echo "fail"
    exit 1
fi
nhctl dev start test-manifest-02 -d details
if [ "$?" != 0 ]; then
    echo "fail"
    exit 1
fi
nhctl port-forward $APPNAME -d details &
sleep 3

mkdir sync_for_test
nhctl sync $APPNAME -d details
if [ "$?" != 0 ]; then
    echo "fail"
    exit 1
fi

sleep 3

nhctl dev end $APPNAME -d  details
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
