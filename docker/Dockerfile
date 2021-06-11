FROM alpine:3.13.5

ENV REGISTRY_ADDR=""
ENV REGISTRY_NAME="default"
ENV REGISTRY_TYPE="zookeeper"

COPY dubbo2istio /usr/local/bin/
ENTRYPOINT /usr/local/bin/dubbo2istio -type=$REGISTRY_TYPE -name=$REGISTRY_NAME -addr=$REGISTRY_ADDR