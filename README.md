# chaoskube

`chaoskube` periodically kills random pods in your Kubernetes cluster.

## Why

Test how your system behaves under arbitrary pod failures.

## Example

Running it will kill a pod in any namespace every 10 minutes by default.

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

Get `chaoskube` via go get, make sure your current context points to your target cluster and use the `--deploy` flag.

```shell
$ go get -u github.com/linki/chaoskube
$ chaoskube --deploy
INFO[0000] Dry run enabled. I won't kill anything. Use --no-dry-run when you're ready.
INFO[0000] Using current context from kubeconfig at /Users/you/.kube/config.
INFO[0000] Deployed quay.io/linki/chaoskube:v0.2.2
```

By default `chaoskube` will be friendly and not kill anything. When you validated your target cluster you may disable dry-run mode. You can also specify a more aggressive interval and other supported flags for your deployment.

```shell
$ chaoskube --interval=1m --no-dry-run --debug --deploy
INFO[0000] Using current context from kubeconfig at /Users/you/.kube/config.
DEBU[0000] Targeting cluster at https://kube.you.me:6443
DEBU[0000] Deploying quay.io/linki/chaoskube:v0.2.2
INFO[0000] Deployed quay.io/linki/chaoskube:v0.2.2
```

Otherwise use the following manifest file or let it serve as an inspiration.

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
        - --interval=1m
        - --no-dry-run
        - --debug
```

If you're running in a Kubernetes cluster and want to target the same cluster use the `--in-cluster` flag as shown.

If you want to target a different cluster or want to run it locally provide a valid kubeconfig via `--kubeconfig` and drop the `--in-cluster` flag. By default, it uses your standard kubeconfig path in your home. Whatever is the current context in there will be targeted.

If you want to increase or decrease the amount of chaos change the interval between killings with the `--interval` flag. Alternatively, you can increase the number of replicas of your `chaoskube` deployment.

Remember that `chaoskube` kills any pod in all your namespaces, including system pods and itself.
