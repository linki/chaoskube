  * [#153](https://github.com/linki/chaoskube/pull/153) Don't attempt to terminate `Terminating` pods @pims

## v0.14.0 - 2019-05-20

  Features:
  * [#121](https://github.com/linki/chaoskube/pull/121) Add include and exclude regular expression filters for pod names @dansimone

## v0.13.0 - 2019-01-30

  Features:
  * [#120](https://github.com/linki/chaoskube/pull/120) Adding JSON as additional log format @syedimam0012

## v0.12.1 - 2019-01-20

  Updates:
  * [#119](https://github.com/linki/chaoskube/pull/119) Add logo for chaoskube @linki
  * [#118](https://github.com/linki/chaoskube/pull/118) [#81](https://github.com/linki/chaoskube/pull/81) Add Dockerfile for `arm32v6` and `arm64v8` @toolboc
  * [#117](https://github.com/linki/chaoskube/pull/117) [#104](https://github.com/linki/chaoskube/pull/104) Abstract termination strategy in order to add more means of killing pods @jakewins @linki

## v0.12.0 - 2019-01-08

  Features:
  * [#116](https://github.com/linki/chaoskube/pull/116) Add several useful Prometheus metrics @ruehowl @shaikatz

  Updates:
  * [#115](https://github.com/linki/chaoskube/pull/115) Replace event related code with Kubernetes's `EventRecorder` @linki
  * [#114](https://github.com/linki/chaoskube/pull/114) Document major difference to `kube-monkey` @prageethw
  * [#113](https://github.com/linki/chaoskube/pull/113) Update dependencies to match Kubernetes v1.12 API @linki
  * [#112](https://github.com/linki/chaoskube/pull/112) Update docker image to alpine v3.8 and go v1.11 @linki

## v0.11.0 - 2018-10-09

  Features:
  * [#110](https://github.com/linki/chaoskube/pull/110) Add option to define grace period given to pods @palmerabollo
  * [#105](https://github.com/linki/chaoskube/pull/105) Implement event creation after terminating a pod @djboris9

  Updates:
  * [#107](https://github.com/linki/chaoskube/pull/107) Replace `glog` with a `noop` logger to allow for read-only filesystem @linki

## v0.10.0 - 2018-08-06

  Features:
  * [#97](https://github.com/linki/chaoskube/pull/97) Expose basic metrics via Prometheus @bakins
  * [#94](https://github.com/linki/chaoskube/pull/94) Add a health check endpoint @bakins
  * [#86](https://github.com/linki/chaoskube/pull/86) Add a flag to exclude Pods under a certain age @bakins
  * [#84](https://github.com/linki/chaoskube/pull/84) Exclude Pods that are not in phase `Running` @linki
  * [#60](https://github.com/linki/chaoskube/pull/60) Add a Dockerfile for building images for `ppc64le` @hchenxa

  Updates:
  * [#96](https://github.com/linki/chaoskube/pull/96) Use versioned functions of `client-go` @linki
  * [#95](https://github.com/linki/chaoskube/pull/95) Handle signals to enable more graceful shutdown @linki
  * [#89](https://github.com/linki/chaoskube/pull/89) Run `chaoskube` as `nobody` by default @bavarianbidi
  * [#77](https://github.com/linki/chaoskube/pull/77) Use [roveralls](https://github.com/lawrencewoodman/roveralls) to improve coverage results @linki
