# chaoskube
[![Build Status](https://travis-ci.org/linki/chaoskube.svg?branch=master)](https://travis-ci.org/linki/chaoskube)
[![Coverage Status](https://coveralls.io/repos/github/linki/chaoskube/badge.svg?branch=master)](https://coveralls.io/github/linki/chaoskube?branch=master)
[![GitHub release](https://img.shields.io/github/release/linki/chaoskube.svg)](https://github.com/linki/chaoskube/releases)
[![Docker Repository on Quay](https://quay.io/repository/linki/chaoskube/status "Docker Repository on Quay")](https://quay.io/repository/linki/chaoskube)
[![go-doc](https://godoc.org/github.com/linki/chaoskube/chaoskube?status.svg)](https://godoc.org/github.com/linki/chaoskube/chaoskube)

`chaoskube` periodically kills random pods in your Kubernetes cluster.

<p align="center"><img src="chaoskube.png" width="40%" align="center" alt="chaoskube"></p>

## Why

Test how your system behaves under arbitrary pod failures.

## Example

Running it will kill a pod in any namespace every 10 minutes by default.

```console
$ chaoskube
INFO[0000] starting up              dryRun=true interval=10m0s version=v0.21.0
INFO[0000] connecting to cluster    master="https://kube.you.me" serverVersion=v1.10.5+coreos.0
INFO[0000] setting pod filter       annotations= labels= minimumAge=0s namespaces=
INFO[0000] setting quiet times      daysOfYear="[]" timesOfDay="[]" weekdays="[]"
INFO[0000] setting timezone         location=UTC name=UTC offset=0
INFO[0001] terminating pod          name=kube-dns-v20-6ikos namespace=kube-system
INFO[0601] terminating pod          name=nginx-701339712-u4fr3 namespace=chaoskube
INFO[1201] terminating pod          name=kube-proxy-gke-earthcoin-pool-3-5ee87f80-n72s namespace=kube-system
INFO[1802] terminating pod          name=nginx-701339712-bfh2y namespace=chaoskube
INFO[2402] terminating pod          name=heapster-v1.2.0-1107848163-bhtcw namespace=kube-system
INFO[3003] terminating pod          name=l7-default-backend-v1.0-o2hc9 namespace=kube-system
INFO[3603] terminating pod          name=heapster-v1.2.0-1107848163-jlfcd namespace=kube-system
INFO[4203] terminating pod          name=nginx-701339712-bfh2y namespace=chaoskube
INFO[4804] terminating pod          name=nginx-701339712-51nt8 namespace=chaoskube
...
```

`chaoskube` allows to filter target pods [by namespaces, labels, annotations and age](#filtering-targets) as well as [exclude certain weekdays, times of day and days of a year](#limit-the-chaos) from chaos.

## How

### Helm

You can install `chaoskube` with [`Helm`](https://github.com/kubernetes/helm). Follow [Helm's Quickstart Guide](https://helm.sh/docs/intro/quickstart/) and then install the `chaoskube` chart.

```console
$ helm install stable/chaoskube
```

Refer to [chaoskube on kubeapps.com](https://kubeapps.com/charts/stable/chaoskube) to learn how to configure it and to find other useful Helm charts.

### Raw manifest

Refer to [example manifest](./examples/). Be sure to give chaoskube appropriate
permissions using provided ClusterRole.

### Configuration

By default `chaoskube` will be friendly and not kill anything. When you validated your target cluster you may disable dry-run mode by passing the flag `--no-dry-run`. You can also specify a more aggressive interval and other supported flags for your deployment.

If you're running in a Kubernetes cluster and want to target the same cluster then this is all you need to do.

If you want to target a different cluster or want to run it locally specify your cluster via the `--master` flag or provide a valid kubeconfig via the `--kubeconfig` flag. By default, it uses your standard kubeconfig path in your home. That means, whatever is the current context in there will be targeted.

If you want to increase or decrease the amount of chaos change the interval between killings with the `--interval` flag. Alternatively, you can increase the number of replicas of your `chaoskube` deployment.

Remember that `chaoskube` by default kills any pod in all your namespaces, including system pods and itself.

`chaoskube` provides a simple HTTP endpoint that can be used to check that it is running. This can be used for [Kubernetes liveness and readiness probes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-probes/). By default, this listens on port 8080. To disable, pass `--metrics-address=""` to `chaoskube`.

## Filtering targets

However, you can limit the search space of `chaoskube` by providing label, annotation, and namespace selectors, pod name include/exclude patterns, as well as a minimum age setting.

```console
$ chaoskube --labels 'app=mate,chaos,stage!=production'
...
INFO[0000] setting pod filter       labels="app=mate,chaos,stage!=production"
```

This selects all pods that have the label `app` set to `mate`, the label `chaos` set to anything and the label `stage` not set to `production` or unset.

You can filter target pods by namespace selector as well.

```console
$ chaoskube --namespaces 'default,testing,staging'
...
INFO[0000] setting pod filter       namespaces="default,staging,testing"
```

This will filter for pods in the three namespaces `default`, `staging` and `testing`.

Namespaces can additionally be filtered by a namespace label selector.

```console
$ chaoskube --namespace-labels='!integration'
...
INFO[0000] setting pod filter       namespaceLabels="!integration"
```

This will exclude all pods from namespaces with the label `integration`.

You can filter target pods by [OwnerReference's](https://godoc.org/k8s.io/apimachinery/pkg/apis/meta/v1#OwnerReference) kind selector.

```console
$ chaoskube --kinds '!DaemonSet,!StatefulSet'
...
INFO[0000] setting pod filter       kinds="!DaemonSet,!StatefulSet"
```

This will exclude any `DaemonSet` and `StatefulSet` pods.

```console
$ chaoskube --kinds 'DaemonSet'
...
INFO[0000] setting pod filter       kinds="DaemonSet"
```

This will only include any `DaemonSet` pods. 

Please note: any `include` filter will automatically exclude all the pods with no OwnerReference defined.

You can filter pods by name:

```console
$ chaoskube --included-pod-names 'foo|bar' --excluded-pod-names 'prod'
...
INFO[0000] setting pod filter       excludedPodNames=prod includedPodNames="foo|bar"
```

This will cause only pods whose name contains 'foo' or 'bar' and does _not_ contain 'prod' to be targeted.

You can also exclude namespaces and mix and match with the label and annotation selectors.

```console
$ chaoskube \
    --labels 'app=mate,chaos,stage!=production' \
    --annotations '!scheduler.alpha.kubernetes.io/critical-pod' \
    --namespaces '!kube-system,!production'
...
INFO[0000] setting pod filter       annotations="!scheduler.alpha.kubernetes.io/critical-pod" labels="app=mate,chaos,stage!=production" namespaces="!kube-system,!production"
```

This further limits the search space of the above label selector by also excluding any pods in the `kube-system` and `production` namespaces as well as ignore all pods that are marked as critical.

The annotation selector can also be used to run `chaoskube` as a cluster addon and allow pods to opt-in to being terminated as you see fit. For example, you could run `chaoskube` like this:

```console
$ chaoskube --annotations 'chaos.alpha.kubernetes.io/enabled=true' --debug
...
INFO[0000] setting pod filter       annotations="chaos.alpha.kubernetes.io/enabled=true"
DEBU[0000] found candidates         count=0
DEBU[0000] no victim found
```

Unless you already use that annotation somewhere, this will initially ignore all of your pods (you can see the number of candidates in debug mode). You could then selectively opt-in individual deployments to chaos mode by annotating their pods with `chaos.alpha.kubernetes.io/enabled=true`.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  replicas: 3
  template:
    metadata:
      annotations:
        chaos.alpha.kubernetes.io/enabled: "true"
    spec:
      ...
```

You can exclude pods that have recently started by using the `--minimum-age` flag.

```console
$ chaoskube --minimum-age 6h
...
INFO[0000] setting pod filter       minimumAge=6h0m0s
```

## Limit the Chaos

You can limit the time when chaos is introduced by weekdays, time periods of a day, day of a year or all of them together.

Add a comma-separated list of abbreviated weekdays via the `--excluded-weekdays` options, a comma-separated list of time periods via the `--excluded-times-of-day` option and/or a comma-separated list of days of a year via the `--excluded-days-of-year` option and specify a `--timezone` by which to interpret them.

```console
$ chaoskube \
    --excluded-weekdays=Sat,Sun \
    --excluded-times-of-day=22:00-08:00,11:00-13:00 \
    --excluded-days-of-year=Apr1,Dec24 \
    --timezone=Europe/Berlin
...
INFO[0000] setting quiet times      daysOfYear="[Apr 1 Dec24]" timesOfDay="[22:00-08:00 11:00-13:00]" weekdays="[Saturday Sunday]"
INFO[0000] setting timezone         location=Europe/Berlin name=CET offset=1
```

Use `UTC`, `Local` or pick a timezone name from the [(IANA) tz database](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones). If you're testing `chaoskube` from your local machine then `Local` makes the most sense. Once you deploy `chaoskube` to your cluster you should deploy it with a specific timezone, e.g. where most of your team members are living, so that both your team and `chaoskube` have a common understanding when a particular weekday begins and ends, for instance. If your team is spread across multiple time zones it's probably best to pick `UTC` which is also the default. Picking the wrong timezone shifts the meaning of a particular weekday by a couple of hours between you and the server.

## Flags

| Option                    | Description                                                          | Default                    |
|---------------------------|----------------------------------------------------------------------|----------------------------|
| `--interval`              | interval between pod terminations                                    | 10m                        |
| `--labels`                | label selector to filter pods by                                     | (matches everything)       |
| `--annotations`           | annotation selector to filter pods by                                | (matches everything)       |
| `--kinds`                 | owner's kind selector to filter pods by                              | (all kinds)                |
| `--namespaces`            | namespace selector to filter pods by                                 | (all namespaces)           |
| `--namespace-labels`      | label selector to filter namespaces and its pods by                  | (all namespaces)           |
| `--included-pod-names`    | regular expression pattern for pod names to include                  | (all included)             |
| `--excluded-pod-names`    | regular expression pattern for pod names to exclude                  | (none excluded)            |
| `--excluded-weekdays`     | weekdays when chaos is to be suspended, e.g. "Sat,Sun"               | (no weekday excluded)      |
| `--excluded-times-of-day` | times of day when chaos is to be suspended, e.g. "22:00-08:00"       | (no times of day excluded) |
| `--excluded-days-of-year` | days of a year when chaos is to be suspended, e.g. "Apr1,Dec24"      | (no days of year excluded) |
| `--timezone`              | timezone from tz database, e.g. "America/New_York", "UTC" or "Local" | (UTC)                      |
| `--max-runtime`           | Maximum runtime before chaoskube exits                               | -1s (infinite time)        |
| `--max-kill`              | Specifies the maximum number of pods to be terminated per interval   | 1                          |
| `--minimum-age`           | Minimum age to filter pods by                                        | 0s (matches every pod)     |
| `--dry-run`               | don't kill pods, only log what would have been done                  | true                       |
| `--log-format`            | specify the format of the log messages. Options are text and json    | text                       |
| `--log-caller`            | include the calling function name and location in the log messages   | false                      |
| `--slack-webhook`         | The address of the slack webhook for notifications                   | disabled                   |

## Related work

There are several other projects that allow you to create some chaos in your Kubernetes cluster.

* [kube-monkey](https://github.com/asobti/kube-monkey) is a sophisticated pod-based chaos monkey for Kubernetes. Each morning it compiles a schedule of pod terminations that should happen throughout the day. It allows to specify a mean time between failures on a per-pod basis, a feature that `chaoskube` [lacks](https://github.com/linki/chaoskube/issues/20). It can also be made aware of groups of pods forming an application so that it can treat them specially, e.g. kill all pods of an application at once. `kube-mokey` allows filtering targets globally via configuration options as well allows pods to opt-in to chaos via annotations,it allows individual apps to opt-in in their own unique way, as an example, app-a can request to kill him each week day one pod, while app-b which more couragues can request to kill 50% of pods. It understands a similar [configuration file](https://github.com/asobti/kube-monkey/blob/069e6fa9dc54ff9c83ac044b2d653f83e9dbdb5a/examples/configmap.yaml) used by Netflix's ChaosMonkey.
* [PowerfulSeal](https://github.com/bloomberg/powerfulseal) is indeed a powerful tool to trouble your Kubernetes setup. Besides killing pods it can also take out your Cloud VMs or kill your Docker daemon. It has a vast number of [configuration options](https://github.com/bloomberg/powerfulseal/blob/1.1.1/tests/policy/example_config.yml) to define what can be killed and when. It also has an interactive mode that allows you to kill pods easily.
* [fabric8's chaos monkey](https://fabric8.io/guide/chaosMonkey.html): A chaos monkey that comes bundled as an app with [fabric8's](https://fabric8.io/) Kubernetes platform. It can be deployed via a UI and reports any actions taken as a chat message and/or desktop notification. It can be configured with an interval and a pod name pattern that possible targets must match.
* [k8aos](https://github.com/AlexsJones/k8aos): An interactive tool that can issue [a series of random pod deletions](https://github.com/AlexsJones/k8aos/blob/0dd0e1876a3d10b558d661bed7a28f79439b489e/core/mischief.go#L41-L51) across an entire Kubernetes cluster or scoped to a namespace.
* [pod-reaper](https://github.com/target/pod-reaper) kills pods based on an interval and a configurable chaos chance. It allows to specify possible target pods via a label selector and namespace. It has the ability successfully shutdown itself after a while and therefore might be suited to work well with Kubernetes Job objects. It can also be configured to kill every pod that has been running for longer than a configurable duration.
* [kubernetes-pod-chaos-monkey](https://github.com/jnewland/kubernetes-pod-chaos-monkey): A very simple random pod killer using `kubectl` written in a [couple lines of bash](https://github.com/jnewland/kubernetes-pod-chaos-monkey/blob/master/chaos.sh). Given a namespace and an interval it kills a random pod in that namespace at each interval. Pretty much like `chaoskube` worked in the beginning.
* [kubeinvaders](https://github.com/lucky-sideburn/KubeInvaders) gamified chaos engineering tool for Kubernetes. It is like Space Invaders but the aliens are pods or worker nodes.

## Acknowledgements

This project wouldn't be where it is with the ideas and help of several awesome contributors:
* Thanks to [@twildeboer](https://github.com/twildeboer) and [@klautcomputing](https://github.com/klautcomputing) who sparked the idea of limiting chaos during certain times, such as [business hours](https://github.com/linki/chaoskube/issues/35) or [holidays](https://github.com/linki/chaoskube/issues/48) as well as the first implementations of this feature in [#54](https://github.com/linki/chaoskube/pull/54) and [#55](https://github.com/linki/chaoskube/pull/55).
* Thanks to [@klautcomputing](https://github.com/klautcomputing) for the first attempt to solve the missing [percentage feature](https://github.com/linki/chaoskube/pull/47) as well as for providing [the RBAC config](https://github.com/linki/chaoskube/pull/30) files.
* Thanks to [@j0sh3rs](https://github.com/j0sh3rs) for bringing [the Helm chart](https://hub.kubeapps.com/charts/stable/chaoskube) to the latest version.
* Thanks to [@klautcomputing](https://github.com/klautcomputing), [@grosser](https://github.com/grosser), [@twz123](https://github.com/twz123), [@hchenxa](https://github.com/hchenxa) and [@bavarianbidi](https://github.com/bavarianbidi) for improvements to the Dockerfile and docs in [#31](https://github.com/linki/chaoskube/pull/31), [#40](https://github.com/linki/chaoskube/pull/40) and [#58](https://github.com/linki/chaoskube/pull/58).
* Thanks to [@bakins](https://github.com/bakins) for adding the minimum age filter in [#86](https://github.com/linki/chaoskube/pull/86).
* Thanks to [@bakins](https://github.com/bakins) for adding a health check and Prometheus metrics in [#94](https://github.com/linki/chaoskube/pull/94) and [#97](https://github.com/linki/chaoskube/pull/97).

## Contributing

Feel free to create issues or submit pull requests.
