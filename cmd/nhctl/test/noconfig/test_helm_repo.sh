#!/bin/bash
kubectl delete ns test-helm-repo-01   >> /dev/null
kubectl create ns test-helm-repo-01
nhctl uninstall test-helm-repo-01   >> /dev/null
nhctl install test-helm-repo-01 --helm-repo-name depscloud --helm-chart-name  depscloud -t helm-repo --debug -n test-helm-repo-01
if [ $? != 0 ]; then
    echo "fail"
    exit 1
fi
nhctl uninstall test-helm-repo-01
if [ $? != 0 ]; then
    echo "fail"
    exit 1
fi
echo "succeed"
