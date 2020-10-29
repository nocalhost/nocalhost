#!/bin/bash

./nhctl install -u git@e.coding.net:codingcorp/bookinfo/bookinfo-manifest.git -d deployment -k ~/.kube/admin-config -t manifest --pre-install items.yaml -n demo4
./nhctl debug  start -d details-v1  -l ruby  -n demo --kubeconfig /Users/xinxinhuang/.kube/admin-config
./nhctl port-forward -d details-v1 -n demo  -l 12345 -r 22 -k ~/.kube/admin-config
./nhctl sync -l share1 -r /opt/microservicess -p 12345
./nhctl debug end -d  details-v1  -n demo --kubeconfig  /Users/xinxinhuang/.kube/admin-config