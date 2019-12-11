
Now we are going to deploy a [Petclinic](https://github.com/spring-projects/spring-petclinic) with MySQL. 

First create a namespace for the database service:

```
kubectl create namespace services
```

then install the database service into the new namespace:

```
kubectl apply -f <(kustomize build samples/mysql)
```{{execute}}

One `Service` for MySQL is installed into the `services` namespace:

```
kubectl get all -n services
```{{execute}}

```
NAME                         READY   STATUS    RESTARTS   AGE
pod/mysql-6f594dc97f-sgmnd   1/1     Running   1          6d10h

NAME            TYPE        CLUSTER-IP       EXTERNAL-IP   PORT(S)    AGE
service/mysql   ClusterIP   10.109.255.187   <none>        3306/TCP   6d10h

NAME                    READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/mysql   1/1     1            1           6d10h

NAME                               DESIRED   CURRENT   READY   AGE
replicaset.apps/mysql-6f594dc97f   1         1         1       6d10h
```

Finally deploy the Petclinic:

```
kubectl apply -f samples/petclinic.yaml
```{{execute}}

Now we can connect to the application. First create an SSH tunnel:

`kubectl port-forward svc/demo 8080:80`{{execute T1}}

and then you can verify that the app is running:

`curl localhost:8080/actuator/health | jq .`{{execute T2}}

```
{
  "status": "UP",
  "components": {
    "db": {
      "status": "UP"
    },
    "diskSpace": {
      "status": "UP"
    },
    "ping": {
      "status": "UP"
    }
  }
}
```

That it! You have an application running in Kubernetes and the database is connected.

Next we will drill into the Petclinic sample and see how all teh pieces fit together.