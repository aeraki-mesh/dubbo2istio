#!/usr/bin/env bash

set -ex
export NAMESPACE="istio-system"
export ISTIOD_ADDR="istiod.istio-system:15010"

BASEDIR=$(dirname "$0")

kubectl create ns istio-system
kubectl apply -f $BASEDIR/../common/aeraki-crd.yaml
kubectl apply -f $BASEDIR/../common/aeraki.yaml -n ${NAMESPACE}
