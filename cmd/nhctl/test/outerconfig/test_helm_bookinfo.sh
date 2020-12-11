#!/bin/bash
APPNAME=test-helm-bookinfo-outer-config-01
kubectl delete ns $APPNAME >> /dev/null
kubectl create ns $APPNAME
nhctl uninstall $APPNAME >> /dev/null
nhctl install $APPNAME -u https://github.com/lyzhang1999/bookinfo --debug -n $APPNAME --config helm_config.yaml
if [ "$?" != 0 ]; then
    echo "fail"
    exit 1
fi
sleep 3


nhctl dev start $APPNAME -d details -s "~/WorkSpaces/coding/nocalhost/cmd/nhctl/test/outerconfig"
if [ "$?" != 0 ]; then
    echo "fail"
    exit 1
fi
sleep 10
#nhctl port-forward $APPNAME -d details &
#if [ "$?" != 0 ]; then
#    echo "fail"
#    exit 1
#fi
#sleep 3

nhctl sync $APPNAME -d details
if [ "$?" != 0 ]; then
    echo "fail"
    exit 1
fi

sleep 30

nhctl dev end $APPNAME -d  details
if [ "$?" != 0 ]; then
    echo "fail"
    exit 1
fi

sleep 10
nhctl uninstall $APPNAME
if [ "$?" != 0 ]; then
    echo "fail"
    exit 1
fi
kubectl delete ns $APPNAME
echo "succeed"
