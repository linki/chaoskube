# Chaoskube Helm Chart

[chaoskube](https://github.com/linki/chaoskube) periodically kills random pods in your Kubernetes cluster.

## TL;DR;

```console
$ helm install stable/chaoskube
```

## Installing the Chart

To install the chart with the release name `my-release`:

```console
$ helm install --name my-release stable/chaoskube
```

The command deploys chaoskube on the Kubernetes cluster in the default configuration. The [configuration](#configuration) section lists the parameters that can be configured during installation.

## Uninstalling the Chart

To uninstall/delete the my-release deployment:

```console
$ helm delete my-release
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

## Configuration

By default `chaoskube` runs in dry-run mode so it doesn't actually kill anything.
If you're sure you want to use it run `helm` with:

```console
$ helm install stable/chaoskube --set dryRun=false
```

| Parameter                 | Description                                         | Default                           |
|---------------------------|-----------------------------------------------------|-----------------------------------|
| `name`                    | container name                                      | chaoskube                         |
| `image`                   | docker image                                        | quay.io/linki/chaoskube           |
| `imageTag`                | docker image tag                                    | v0.4.0                            |
| `replicas`                | number of replicas to run                           | 1                                 |
| `interval`                | interval between pod terminations                   | 10m                               |
| `labels`                  | label selector to filter pods by                    | "" (matches everything)           |
| `annotations`             | annotation selector to filter pods by               | "" (matches everything)           |
| `namespaces`              | namespace selector to filter pods by                | "" (all namespaces)               |
| `dryRun`                  | don't kill pods, only log what would have been done | true                              |
| `limitChaos`              | limit chaos according to specified times/days       | false                             |
| `location`                | timezone from tz database, e.g "America/New_York"   | (none)                            |
| `offDays`                 | days when chaos is to be suspended. (Or "none")     | "Saturday,Sunday"                 |
| `chaosHrs.start`          | start time for introducing chaos (24hr time)        | 9:30                              |
| `chaosHrs.end`            | end time for introducing chaos (24hr time)          | 14:30                             |
| `holidays`                | comma-separated, "YYYY-MM-DD" days to skip chaos    | "" (none)                         |
| `resources.cpu`           | cpu resource requests and limits                    | 10m                               |
| `resources.memory`        | memory resource requests and limits                 | 16Mi                              |

Setting label and namespaces selectors from the shell can be tricky but is possible (example with zsh):

```console
$ helm install \
  --set labels='app=mate\,stage!=prod',namespaces='!kube-system\,!production' \
  stable/chaoskube --debug --dry-run | grep -A4 args
    args:
    - --in-cluster
    - --interval=10m
    - --labels=app=foo,stage!=prod
    - --namespaces=!kube-system,!production
```
