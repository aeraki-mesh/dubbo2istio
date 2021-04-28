FROM alpine:3.13.5

ENV ZK_ADDR=""

COPY dubbo2istio /usr/local/bin/
ENTRYPOINT /usr/local/bin/dubbo2istio -zkaddr=$ZK_ADDR