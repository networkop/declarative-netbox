# declarative-netbox
An experimental project to explore the idea of managing the Netbox state via declarative API. This repo contains the following:

1. Inside the [./netbox](https://github.com/networkop/declarative-netbox/tree/main/netbox) directory -- a Go library implementing declarative management of Netbox devices, built upon the [netbox-community/go-netbox](https://github.com/netbox-community/go-netbox) package.
2. Inside the [./cmd](https://github.com/networkop/declarative-netbox/tree/main/cmd) directory -- a command-line application `nbctl` (similar to `kubectl`) that can retrieve, apply and delete Netbox objects (not relying on k8s cluster).
3. Inside the [./controllers](https://github.com/networkop/declarative-netbox/tree/main/controllers) directory -- a Kubernetes controller that automates the management of Netbox objects via Kubernetes CRDs.

All of the above applications use a common API defined in the [./api](https://github.com/networkop/declarative-netbox/tree/main/api/v1) directory.


## Quick Start

### Prerequisites 

1. Prepare the test environment

Build a Kubernetes cluster that will host our controller and the Netbox deployment

```
kind create cluster --config hack/kind.yaml
```

Deploy Netbox inside this cluster

```
make netbox
```

Expose Netbox on localhost:32178

```
kubectl apply -f hack/nodeport.yml
```

2. Pre-seed Netbox with static configuration data

We will define site and a couple of device roles and types


```
kubectl exec -it deploy/k8s-netbox bash
source /opt/netbox/venv/bin/activate
/opt/netbox/netbox/manage.py nbshell

Site(name="CITC", status="active").save()

DeviceRole(name="leaf", slug="leaf").save()
DeviceRole(name="spine", slug="spine").save()

Manufacturer(name="nvidia").save()
m = Manufacturer.objects.get(name="nvidia")


DeviceType(model="SN3420", slug="SN3420", manufacturer=m).save()
DeviceType(model="SN3700", slug="SN3700", manufacturer=m).save()
```

## The `nbcli` UX walkthrough

Build the `nbcli` binary

```
make build-cli
```

Authenticate against a Netbox instance (these details will be stored in ~/.netbox/config)

```
./bin/nbctl login  http://localhost:32178 0123456789abcdef0123456789abcdef01234567
```


Get the current list of devices

```
./bin/nbctl get device
+------+----+------+------+------+
| NAME | ID | TYPE | ROLE | SITE |
+------+----+------+------+------+
+------+----+------+------+------+
```

Apply the two new devices from [./config/samples/device_create.yml](https://github.com/networkop/declarative-netbox/blob/main/config/samples/device_create.yml)

```
 ./bin/nbctl apply -f config/samples/device_create.yml
```

Check the the devices are there (this check can also be done via the [web UI](http://localhost:32178/dcim/device-types/))

```
./bin/nbctl get device
+----------+----+--------+-------+------+
| NAME     | ID | TYPE   | ROLE  | SITE |
+----------+----+--------+-------+------+
| leaf-99  |  1 | SN3420 | leaf  | CITC |
| spine-01 |  2 | SN3700 | spine | CITC |
+----------+----+--------+-------+------+
```

Optionally, you can apply a `-oyaml` flag and output those devices in the original YAML format:

```
./bin/nbctl get device leaf-99 -oyaml
apiVersion: netbox.networkop.co.uk/v1
kind: Device
metadata:
  creationTimestamp: null
  name: leaf-99
spec:
  device_type: SN3420
  role: leaf
  site: CITC
status:
  id: 1
  state: Ready
```

Apply the new change from [./config/samples/device_update.yml](https://github.com/networkop/declarative-netbox/blob/main/config/samples/device_update.yml) (swapped device type)

```
./bin/nbctl apply -f config/samples/device_update.yml
./bin/nbctl get device
+----------+----+--------+-------+------+
| NAME     | ID | TYPE   | ROLE  | SITE |
+----------+----+--------+-------+------+
| leaf-99  |  1 | SN3700 | leaf  | CITC |
| spine-01 |  2 | SN3420 | spine | CITC |
+----------+----+--------+-------+------+
```

Delete all devices

```
./bin/nbctl delete -f config/samples/device_update.yml
./bin/nbctl get device
+------+----+------+------+------+
| NAME | ID | TYPE | ROLE | SITE |
+------+----+------+------+------+
+------+----+------+------+------+
```

## The Kubernetes controller walkthrough

Install CRDs generated from the API

```
make install
```

Deploy the Netbox controller

```
make deploy
```

Wait for the controller to transition to ready state

```
kubectl get deployments.apps -n declarative-netbox-system
NAME                                    READY   UP-TO-DATE   AVAILABLE   AGE
declarative-netbox-controller-manager   1/1     1            1           47s
```

Apply the device configuration (this is the same YAML that we used in CLI tool)

```
kubectl apply -f config/samples/device_create.yml
device.netbox.networkop.co.uk/leaf-99 created
device.netbox.networkop.co.uk/spine-01 created
```

Check the configured devices

```
kubectl get device
NAME       ID    SITE   TYPE     ROLE
leaf-99    3     CITC   SN3420   leaf
spine-01   4     CITC   SN3700   spine
```

Update the device configuration

```
kubectl apply -f config/samples/device_update.yml
device.netbox.networkop.co.uk/leaf-99 configured
device.netbox.networkop.co.uk/spine-01 configured
```

View details of individual devices

```
kubectl get device leaf-99 -oyaml                                                                                                    ⎈ kind-kind
apiVersion: netbox.networkop.co.uk/v1
kind: Device
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"netbox.networkop.co.uk/v1","kind":"Device","metadata":{"annotations":{},"name":"leaf-99","namespace":"default"},"spec":{"device_type":"SN3700","role":"leaf","site":"CITC"}}
  creationTimestamp: "2021-12-24T09:51:46Z"
  finalizers:
  - finalizers.netbox.networkop.co.uk
  generation: 2
  name: leaf-99
  namespace: default
  resourceVersion: "4191"
  uid: dab8c5c4-66c8-4a1c-84fb-73a2bf686dc8
spec:
  device_type: SN3700
  role: leaf
  site: CITC
status:
  id: 3
  observedGeneration: 2
  state: Ready
```

Delete configured devices

```
kubectl delete -f config/samples/device_update.yml
```