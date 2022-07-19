# chaoskube Helm Chart

chaoskube periodically kills random pods in your Kubernetes cluster.

## Installation

```console
$ helm repo add chaoskube https://linki.github.io/chaoskube/
$ helm install chaoskube chaoskube/chaoskube --atomic --namespace=chaoskube --create-namespace
```

## Example Helm values

Basic configuration with `3` replicas and minimum resources assigned that will take out any pod it can find (including the other chaoskube pods):

```yaml
chaoskube:
  args:
    no-dry-run: ""
replicaCount: 3
resources:
  limits:
    cpu: 15m
    memory: 32Mi
  requests:
    cpu: 15m
    memory: 32Mi
```

More advance configuration that limits based on several factors like time, day of the week, and date:

```yaml
chaoskube:
  args:
    # kill a pod every 10 minutes
    interval: "10m"
    # only target pods in the test environment
    labels: "environment=test"
    # only consider pods with this annotation
    annotations: "chaos.alpha.kubernetes.io/enabled=true"
    # exclude all DaemonSet pods
    kinds: "!DaemonSet"
    # exclude all pods in the kube-system namespace
    namespaces: "!kube-system"
    # don't kill anything on weekends
    excluded-weekdays: "Sat,Sun"
    # don't kill anything during the night or at lunchtime
    excluded-times-of-day: "22:00-08:00,11:00-13:00"
    # don't kill anything as a joke or on christmas eve
    excluded-days-of-year: "Apr1,Dec24"
    # let's make sure we all agree on what the above times mean
    timezone: "UTC"
    # exclude all pods that haven't been running for at least one hour
    #minimum-age: "1h"
    # terminate pods for real: this disables dry-run mode which is on by default
    no-dry-run: ""
replicaCount: 3
resources:
  limits:
    cpu: 15m
    memory: 32Mi
  requests:
    cpu: 15m
    memory: 32Mi
```
