# rbac

this folder contains an example, if you aren't cluster-admin and your chaoskube-deployment should be placed in the namespace `chaoskube`.

the `rbac.yaml` contains the `role` and `rolebinding`, which should applied into the `default` namespace to allow the serviceaccount `chaoskube`
in namespace `chaoskube` the deletion and list of pods.
