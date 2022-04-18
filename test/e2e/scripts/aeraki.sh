#!/usr/bin/env bash

set -ex
export AERAKI_NAMESPACE="istio-system"
export AERAKI_ISTIOD_ADDR="istiod.istio-system:15010"
export AERAKI_IS_MASTER="true"
export AERAKI_TAG="latest"

BASEDIR=$(dirname "$0")

kubectl create ns istio-system
kubectl apply -f $BASEDIR/../common/aeraki-crd.yaml
kubectl apply -f $BASEDIR/../common/aeraki.yaml -n ${AERAKI_NAMESPACE}
