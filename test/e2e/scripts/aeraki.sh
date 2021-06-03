#!/usr/bin/env bash

set -ex
export NAMESPACE="istio-system"
export ISTIOD_ADDR="istiod.istio-system:15010"
export BUILD_TAG=latest

BASEDIR=$(dirname "$0")

envsubst < $BASEDIR/../common/aeraki.yaml > aeraki.yaml
kubectl create ns istio-system
kubectl apply -f $BASEDIR/../../../crd/kubernetes/customresourcedefinitions.gen.yaml
kubectl apply -f aeraki.yaml -n ${NAMESPACE}
