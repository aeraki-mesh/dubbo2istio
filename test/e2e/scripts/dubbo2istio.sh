#!/usr/bin/env bash

set -ex

export BUILD_TAG=`git log --format="%H" -n 1`

BASEDIR=$(dirname "$0")

envsubst < $BASEDIR/../common/dubbo2istio.yaml > dubbo2istio.yaml
kubectl create ns dubbo
kubectl apply -f dubbo2istio.yaml -n dubbo
