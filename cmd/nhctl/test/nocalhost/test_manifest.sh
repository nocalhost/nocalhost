#!/bin/bash
echo "1 rawManifest-nocalhostConfig 2 rawManifest-outerConfig 3 helm_outerConfig 4 helmRepo"
read -p "choose the app type you want test: " type


SVCNAME=details

if [ "$type" == 1 ]; then
    APPNAME=test-manifest-bookinfo-nocalhost-config-01
    kubectl delete ns $APPNAME >> /dev/null
    kubectl create ns $APPNAME
    echo "clean: uninstalling"
    nhctl uninstall $APPNAME --force --debug >> /dev/null
    echo "installing"
    nhctl install $APPNAME -u https://github.com/lyzhang1999/bookinfo --debug -n $APPNAME
    if [ "$?" != 0 ]; then
        echo "fail"
        exit 1
    fi
elif [ "$type" == 2 ]; then
    APPNAME=test-manifest-bookinfo-outer-config-01
    kubectl delete ns $APPNAME >> /dev/null
    kubectl create ns $APPNAME
    echo "clean: uninstalling"
    nhctl uninstall $APPNAME --force --debug >> /dev/null
    echo "installing"
    nhctl install $APPNAME -u https://github.com/lyzhang1999/bookinfo --debug -n $APPNAME --config config.yaml
    if [ "$?" != 0 ]; then
        echo "fail"
        exit 1
    fi
elif [ "$type" == 3 ]; then
    APPNAME=test-manifest-bookinfo-outer-config-01
    kubectl delete ns $APPNAME >> /dev/null
    kubectl create ns $APPNAME
    echo "clean: uninstalling"
    nhctl uninstall $APPNAME --force --debug >> /dev/null
    echo "installing"
    nhctl install $APPNAME -u https://e.coding.net/codingcorp/nocalhost/bookinfo-noconfig.git --debug -n $APPNAME --config helm_config.yaml
    if [ "$?" != 0 ]; then
        echo "fail"
        exit 1
    fi
    SVCNAME=$APPNAME-details
else
  exit 0
fi


rm -rf ~/sync
mkdir -p ~/sync
touch ~/sync/hello_nhctl

echo "entering dev model..."
nhctl dev start $APPNAME -d $SVCNAME -s ~/sync --debug
if [ "$?" != 0 ]; then
    echo "fail"
    exit 1
fi

echo "starting sync..."
nhctl sync $APPNAME -d $SVCNAME
if [ "$?" != 0 ]; then
    echo "fail"
    exit 1
fi
sleep 3

echo "executing command..."
EXEC_OUTPUT=$(nhctl exec $APPNAME -d $SVCNAME -c ls)
if [ "$?" != 0 ]; then
    echo "fail"
    exit 1
fi
len=${#EXEC_OUTPUT}
EXEC_OUTPUT=${EXEC_OUTPUT:0:len-1}

if [ "$EXEC_OUTPUT" != "hello_nhctl" ]; then
    echo "fail: error output $EXEC_OUTPUT"
    exit 1
fi

echo "entering terminal..."
nhctl dev terminal $APPNAME -d $SVCNAME

echo "starting port forward..."
nhctl port-forward $APPNAME -d $SVCNAME  -p :9080  --debug
if [ "$?" != 0 ]; then
    echo "fail"
    exit 1
fi
sleep 3

read -p "press any key to end dev..." no
nhctl dev end $APPNAME -d $SVCNAME --debug
if [ "$?" != 0 ]; then
    echo "fail"
    exit 1
fi

echo "uninstalling application..."
nhctl uninstall $APPNAME --debug
if [ "$?" != 0 ]; then
    echo "fail"
    exit 1
fi
kubectl delete ns $APPNAME
echo "succeed"
