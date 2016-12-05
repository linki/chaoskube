# chaoskube
Randomly kills pods in your Kubernetes cluster

## Why

Test how your system behaves under random pod failures.

## Example

Running it will kill a random pod in any namespace every 10 minutes by default.

```
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
