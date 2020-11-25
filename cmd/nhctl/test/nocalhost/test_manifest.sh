#!/bin/bash
APPNAME=test-manifest-bookinfo-nocalhost-config-01
kubectl delete ns $APPNAME >> /dev/null
kubectl create ns $APPNAME
nhctl uninstall $APPNAME >> /dev/null
nhctl install $APPNAME -u https://e.coding.net/codingcorp/nocalhost/mini-bookinfo-manifest.git --debug -n $APPNAME
if [ "$?" != 0 ]; then
    echo "fail"
    exit 1
fi
nhctl dev start $APPNAME -d details
if [ "$?" != 0 ]; then
    echo "fail"
    exit 1
fi
nhctl port-forward $APPNAME -d details &
if [ "$?" != 0 ]; then
    echo "fail"
    exit 1
fi
sleep 3

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

nhctl uninstall $APPNAME
if [ "$?" != 0 ]; then
    echo "fail"
    exit 1
fi
kubectl delete ns $APPNAME
echo "succeed"
