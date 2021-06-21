#!/usr/bin/env bash

set -ex

export BUILD_TAG=`git log --format="%H" -n 1`

BASEDIR=$(dirname "$0")

if [ "$1" == zk ]; then
  envsubst < $BASEDIR/../common/dubbo2istio-zk.yaml > dubbo2istio.yaml
elif [ "$1" == nacos ]; then
  envsubst < $BASEDIR/../common/dubbo2istio-nacos.yaml > dubbo2istio.yaml
  elif [ "$1" == etcd ]; then
  envsubst < $BASEDIR/../common/dubbo2istio-etcd.yaml > dubbo2istio.yaml
fi

kubectl create ns dubbo
kubectl apply -f dubbo2istio.yaml -n dubbo