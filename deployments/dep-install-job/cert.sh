#!/bin/bash

set -e

usage() {
    cat <<EOF
Generate certificate suitable for use with an Istio webhook service.
This script uses k8s' CertificateSigningRequest API to a generate a
certificate signed by k8s CA suitable for use with Istio webhook
services. This requires permissions to create and approve CSR. See
https://kubernetes.io/docs/tasks/tls/managing-tls-in-a-cluster for
detailed explantion and additional instructions.
The server key/cert k8s CA cert are stored in a k8s secret.
usage: ${0} [OPTIONS]
The following flags are required.
       --service          Service name of webhook.
       --namespace        Namespace where webhook service and secret reside.
       --secret           Secret name for CA certificate and server certificate/key pair.
EOF
    exit 1
}

while [[ $# -gt 0 ]]; do
    case ${1} in
        --service)
            service="$2"
            shift
            ;;
        --secret)
            secret="$2"
            shift
            ;;
        --namespace)
            namespace="$2"
            shift
            ;;
        *)
            usage
            ;;
    esac
    shift
done

[ -z ${service} ] && service=nocalhost-sidecar-injector-controller
[ -z ${secret} ] && secret=nocalhost-sidecar-injector-certs
[ -z ${namespace} ] && namespace=nocalhost-reserved


[ -n "${DEP_NAMESPACE}" ] && namespace=${DEP_NAMESPACE}

if kubectl get secrets -n ${namespace} ${secret} -o json|grep '"ca-cert.pem":' > /dev/null; then
    echo "secret ${namespace}/${secret} has been created so do not need to create one."
    return
fi

if [ ! -x "$(command -v openssl)" ]; then
    echo "openssl not found"
    exit 1
fi

csrName=${service}.${namespace}
tmpdir=$(mktemp -d)
echo "creating certs in tmpdir ${tmpdir} "

cat <<EOF >> ${tmpdir}/csr.conf
[req]
req_extensions = v3_req
distinguished_name = req_distinguished_name
[req_distinguished_name]
[ v3_req ]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names
[alt_names]
DNS.1 = ${service}
DNS.2 = ${service}.${namespace}
DNS.3 = ${service}.${namespace}.svc
EOF

# Create ca
openssl genrsa -out ${tmpdir}/ca-key.pem 2048
openssl req -x509 -new -nodes -key ${tmpdir}/ca-key.pem -days 36500 -out ${tmpdir}/ca-cert.pem -subj "/CN=${service}.${namespace}.svc"

# Create a server certificate.
openssl genrsa -out ${tmpdir}/server-key.pem 2048
# Note the CN is the DNS name of the service of the webhook.
openssl req -new -key ${tmpdir}/server-key.pem -out ${tmpdir}/server.csr -subj "/CN=${service}.${namespace}.svc" -config ${tmpdir}/csr.conf
openssl x509 -req -in ${tmpdir}/server.csr -CA ${tmpdir}/ca-cert.pem -CAkey ${tmpdir}/ca-key.pem -CAcreateserial -out ${tmpdir}/server-cert.pem -days 36500 -extensions v3_req -extfile ${tmpdir}/csr.conf

# create the secret with CA cert and server cert/key
kubectl create secret generic ${secret} \
        --from-file=ca-cert.pem=${tmpdir}/ca-cert.pem \
        --from-file=key.pem=${tmpdir}/server-key.pem \
        --from-file=cert.pem=${tmpdir}/server-cert.pem \
        --dry-run=client -o yaml |
    kubectl -n ${namespace} apply -f -
