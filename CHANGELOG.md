  Updates:
  * [#595](https://github.com/linki/chaoskube/pull/595) Update builds to use Go `v1.22` @linki
  * [#568](https://github.com/linki/chaoskube/pull/568) Update builds to use Go `v1.21` @linki
  * [#528](https://github.com/linki/chaoskube/pull/528) Update builds to use Go `v1.20` @linki
  * [#505](https://github.com/linki/chaoskube/pull/505) Update builds to use Go `v1.19` @linki

## v0.22.0 - 2021-11-05

  Features:
  * [#325](https://github.com/linki/chaoskube/pull/325) Add maximum runtime flag @IoannisMatzaris
  * [#295](https://github.com/linki/chaoskube/pull/295) Add a Helm chart to deploy chaoskube @ghouscht

  Updates:
  * [#374](https://github.com/linki/chaoskube/pull/374) Add kubeinvaders as related project in the docs @lucky-sideburn
  * [#341](https://github.com/linki/chaoskube/pull/341) Add namespace metadata to deleted pods metric @linki
  * [#323](https://github.com/linki/chaoskube/pull/323) Add missing max-kill flag to the docs @KeisukeYamashita
  * Update several dependencies, such as Kubernetes to v0.20.x.

## v0.21.0 - 2020-09-28

  Features:
  * [#224](https://github.com/linki/chaoskube/pull/224) Allow to filter by OwnerReference's Kind @qlikcoe

  Updates:
  * [#248](https://github.com/linki/chaoskube/pull/248) Added test & build to Makefile @el10savio

## v0.20.0 - 2020-07-03

  Updates:
  * [#197](https://github.com/linki/chaoskube/pull/197) [#203](https://github.com/linki/chaoskube/pull/203) Fix a bug that caused chaoskube to always kill the same pod of a replicated group of pods @HaveFun83 @linki

## v0.19.0 - 2020-04-02

  Updates:
  * [#192](https://github.com/linki/chaoskube/pull/192) Use `context.Context` to cancel in-flight requests @linki
  * [#191](https://github.com/linki/chaoskube/pull/191) Update client-go to `v0.18.0` @linki
  * [#180](https://github.com/linki/chaoskube/pull/180) Update builds to use Go `v1.14` @linki

## v0.18.0 - 2020-02-03

  Updates:
  * [#170](https://github.com/linki/chaoskube/pull/170) Add slack webhook flag to documentation @Clivern
  * [#169](https://github.com/linki/chaoskube/pull/169) Update client-go to v0.17.0 @linki
  * [#167](https://github.com/linki/chaoskube/pull/167) Add Makefile and prettify test output @linki
  * [#166](https://github.com/linki/chaoskube/pull/166) Update klog to v1.0.0 @linki
  * [#164](https://github.com/linki/chaoskube/pull/164) Update Helm's Quickstart Guide link in README @SergioSV96

## v0.17.0 - 2019-12-09

  Features:
  * [#158](https://github.com/linki/chaoskube/pull/158) Support for sending Slack notifications @GaruGaru

## v0.16.0 - 2019-11-08

  Features:
  * [#154](https://github.com/linki/chaoskube/pull/154) Add support for terminating multiple pods per iteration @pims

  Updates:
  * [#156](https://github.com/linki/chaoskube/pull/156) Remove incomplete snippet from the readme and point to examples @jan-warchol
  * [#153](https://github.com/linki/chaoskube/pull/153) Don't attempt to terminate `Terminating` pods @pims
  * [#148](https://github.com/linki/chaoskube/pull/148) Update builds to use Go `v1.13` @linki
  * [#140](https://github.com/linki/chaoskube/pull/140) Update Docker images to use alpine `3.10` @linki

## v0.15.1 - 2019-08-09

  Updates:
  * [#137](https://github.com/linki/chaoskube/pull/137) [#138](https://github.com/linki/chaoskube/pull/138) Avoid writing logs to the container filesystem @linki

## v0.15.0 - 2019-07-30

  Features:
  * [#130](https://github.com/linki/chaoskube/pull/130) Add `--log-caller` flag that adds file name and line to the log output @linki

  Updates:
  * [#129](https://github.com/linki/chaoskube/pull/129) Update client-go to `v12` for Kubernetes `v1.14` @linki
  * [#126](https://github.com/linki/chaoskube/pull/126) Update builds to use Go `v1.12` and Go Modules @linki

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
