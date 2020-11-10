
The controller is in Dockerhub and the CRDs are defined in Github, so you should be able to deploy it from just the YAML:

```
kubectl apply -f <(kustomize build github.com/dsyer/spring-boot-operator/config/default?ref=main)
```{{execute}}

> TIP: If the command above fails, keep trying: the network environment takes a minute to stabilize sometimes.

One `Service` for the controller is installed into the `spring-system` namespace:

```
kubectl get all -n spring-system
```{{execute}}

```
NAME                                             READY   STATUS    RESTARTS   AGE
pod/spring-controller-manager-79c6c95677-8hf89   2/2     Running   0          3m17s

NAME                                                TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)    AGE
service/spring-controller-manager-metrics-service   ClusterIP   10.111.94.226   <none>        8443/TCP   3m17s

NAME                                        READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/spring-controller-manager   1/1     1            1           3m17s

NAME                                                   DESIRED   CURRENT   READY   AGE
replicaset.apps/spring-controller-manager-79c6c95677   1         1         1       3m17s
```

Keep looking at the `spring-system` namespace until the controller `Pod` is `Running`. It might take a minute to download the container.
