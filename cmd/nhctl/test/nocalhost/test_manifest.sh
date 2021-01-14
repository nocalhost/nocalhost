#!/bin/bash
echo "1 rawManifest-nocalhostConfig 2 rawManifest-outerConfig 3 helm_outerConfig 4 helmRepo"
read -p "Choose the app type you want test: " type


SVCNAME=details

rm -rf ~/sync
mkdir -p ~/sync/test
touch ~/sync/test/hello_nhctl

if [ "$type" == 1 ]; then
    APPNAME=test-manifest-bookinfo-nocalhost-config-01
    kubectl delete ns $APPNAME >> /dev/null
    kubectl create ns $APPNAME
    echo "Clean: uninstalling"
    nhctl uninstall $APPNAME --force --debug >> /dev/null

    echo "Installing"
    nhctl install $APPNAME -u https://github.com/nocalhost/bookinfo.git --debug -n $APPNAME --config config

    if [ "$?" != 0 ]; then
        echo "Fail"
        exit 1
    fi
elif [ "$type" == 2 ]; then
    APPNAME=test-manifest-bookinfo-outer-config-01
    kubectl delete ns $APPNAME >> /dev/null
    kubectl create ns $APPNAME
    echo "Clean: uninstalling"
    nhctl uninstall $APPNAME --force --debug >> /dev/null
    echo "Installing"
    nhctl install $APPNAME -u https://github.com/nocalhost/bookinfo.git --debug -n $APPNAME --outer-config config.yaml
    if [ "$?" != 0 ]; then
        echo "Fail"
        exit 1
    fi

    touch ~/sync/nosync.txt
    #touch ~/sync/sync.txt
    touch ~/sync/test/nosync.txt
elif [ "$type" == 3 ]; then
    APPNAME=test-helm-bookinfo-outer-config-01
    kubectl delete ns $APPNAME >> /dev/null
    kubectl create ns $APPNAME
    echo "Clean: uninstalling"
    nhctl uninstall $APPNAME --force --debug >> /dev/null
    echo "Installing"
    nhctl install $APPNAME -u https://github.com/nocalhost/bookinfo.git --debug -n $APPNAME --outer-config helm_config.yaml
    if [ "$?" != 0 ]; then
        echo "Fail"
        exit 1
    fi
#    SVCNAME=$APPNAME-details
else
  exit 0
fi

echo "Entering dev DevMode..."
nhctl dev start $APPNAME -d $SVCNAME -s ~/sync --debug
if [ "$?" != 0 ]; then
    echo "Fail"
    exit 1
fi

echo "Starting sync..."
nhctl sync $APPNAME -d $SVCNAME
if [ "$?" != 0 ]; then
    echo "Fail"
    exit 1
fi
sleep 10

echo "Executing command..."
EXEC_OUTPUT=$(nhctl exec $APPNAME -d $SVCNAME -c ls -c test/)
if [ "$?" != 0 ]; then
    echo "Output fail: $EXEC_OUTPUT"
    exit 1
fi
len=${#EXEC_OUTPUT}
EXEC_OUTPUT=${EXEC_OUTPUT:0:len-1}

if [ "$EXEC_OUTPUT" != "hello_nhctl" ]; then
    echo "Fail: error output $EXEC_OUTPUT"
#    exit 1
fi

echo "Entering terminal..."
nhctl dev terminal $APPNAME -d $SVCNAME

echo "Starting port forward..."
nhctl port-forward $APPNAME -d $SVCNAME  -p :9080  --debug
if [ "$?" != 0 ]; then
    echo "Fail"
    exit 1
fi
sleep 3

read -p "Press any key to end DevMode..." no
nhctl dev end $APPNAME -d $SVCNAME --debug
if [ "$?" != 0 ]; then
    echo "Fail"
    exit 1
fi

echo "Uninstalling application..."
nhctl uninstall $APPNAME --debug
if [ "$?" != 0 ]; then
    echo "Fail"
    exit 1
fi
kubectl delete ns $APPNAME
echo "Succeed"
