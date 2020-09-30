## WallClocks
### Deployment
This project deployment has been tested against kind 0.7

You might need following components installed on your dev machine in order to build an image
```
wget https://github.com/kubernetes-sigs/kubebuilder/releases/download/v2.3.1/kubebuilder_2.3.1_darwin_amd64.tar.gz
tar xzvf kubebuilder_2.3.1_darwin_amd64.tar.gz
sudo mv kubebuilder_2.3.1_darwin_amd64 /usr/local/kubebuilder
```
Build artefacts
```
make
```

Deploy rbacs and crds to the cluster
```
make install
```

Build the docker image and deploy it to cluster
```
make docker-build docker-push IMG=alekcander/clocks:latest  
make deploy IMG=alekcander/clocks:latest 
```
you should see something like this as an output
```
/Users/alexander.chernov/go/bin/controller-gen "crd:trivialVersions=true" rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases
cd config/manager && kustomize edit set image controller=alekcander/clocks:latest
kustomize build config/default | kubectl apply -f -
namespace/ziglu-system created
customresourcedefinition.apiextensions.k8s.io/timezones.wallclocks.ziglu.io configured
customresourcedefinition.apiextensions.k8s.io/wallclocks.wallclocks.ziglu.io configured
role.rbac.authorization.k8s.io/ziglu-leader-election-role created
clusterrole.rbac.authorization.k8s.io/ziglu-manager-role created
clusterrole.rbac.authorization.k8s.io/ziglu-proxy-role created
clusterrole.rbac.authorization.k8s.io/ziglu-metrics-reader created
rolebinding.rbac.authorization.k8s.io/ziglu-leader-election-rolebinding created
clusterrolebinding.rbac.authorization.k8s.io/ziglu-manager-rolebinding created
clusterrolebinding.rbac.authorization.k8s.io/ziglu-proxy-rolebinding created
service/ziglu-controller-manager-metrics-service created
deployment.apps/ziglu-controller-manager created

```

Deploy couple of resources
```
apiVersion: wallclocks.ziglu.io/v1beta1
kind: Timezone
metadata:
  name: timezone-sample
spec:
  # Add fields here
  timezones:
    - America/Los_Angeles
    - Europe/London
---
apiVersion: wallclocks.ziglu.io/v1beta1
kind: Timezone
metadata:
  name: italy
spec:
  timezones:
    - Europe/Rome
```
