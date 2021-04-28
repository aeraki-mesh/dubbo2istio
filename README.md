# Dubbo2Istio

[![CI Tests](https://github.com/aeraki-framework/dubbo2istio/workflows/ci/badge.svg?branch=master)](https://github.com/aeraki-framework/dubbo2istio/actions?query=branch%3Amaster+event%3Apush+workflow%3A%22ci%22)

Dubbo2istio watches Dubbo ZooKeeper registry and synchronize all the dubbo services to Istio.

Dubbo2istio will create a ServiceEntry resource for each service in the Dubbo ZooKeeper registry.

Like this diagram shows, [Aeraki + Dubbo2istio](https://github.com/aeraki-framework/aeraki) can help you manage dubbo applications in Istio Service Meshes. 
![ dubbo2istio ](doc/dubbo2istio.png)