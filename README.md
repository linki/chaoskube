# chaoskube
Randomly kills pods in your Kubernetes cluster

## Why

Test how your system behaves under random pod failures.

## Example

Running it will kill a random pod in any namespace every 10 minutes by default.

```shell
Killing pod kube-system/kube-dns-v20-6ikos
Killing pod chaoskube/nginx-701339712-u4fr3
Killing pod kube-system/kube-proxy-gke-earthcoin-pool-3-5ee87f80-n72s
Killing pod chaoskube/nginx-701339712-bfh2y
Killing pod kube-system/heapster-v1.2.0-1107848163-bhtcw
Killing pod kube-system/l7-default-backend-v1.0-o2hc9
Killing pod kube-system/heapster-v1.2.0-1107848163-jlfcd
Killing pod chaoskube/nginx-701339712-bfh2y
Killing pod chaoskube/nginx-701339712-51nt8
...
```

## How

Use the included manifest file or let it serve as an inspiration.

```yaml
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: chaoskube
spec:
  replicas: 1
  template:
    metadata:
      labels:
        app: chaoskube
    spec:
      containers:
      - name: chaoskube
        image: quay.io/linki/chaoskube:v0.2.2
        args:
        - --in-cluster
        - --interval=10m
```

If you're running in a Kubernetes cluster and want to target the same cluster use the `--in-cluster` flag as shown.

If you want to target a different cluster or want to run it locally use `kubectl proxy --port 8001` to forward a local port to your API server and drop the `--in-cluster` flag.

If you want to increase or decrease the amount of chaos change the interval between killings with the `--interval` flag. Alternatively, you can increase the number of replicas of your chaoskube deployment.

Remember that chaoskube kills any pod in all your namespaces, including system pods and itself.

[1]: https://quay.io/repository/coreos/hyperkube?tab=tags
