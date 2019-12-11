
Now we are going to deploy a [Petclinic](https://github.com/spring-projects/spring-petclinic) with MySQL. 

First create a namespace for the database service:

```
kubectl create namespace services
```{{execute}}

Then we need to create a `PersistentVolume`:

```
sudo mkdir /mnt/data && kubectl apply -f samples/mysql/pv.yaml -n services
```{{execute}}

At this point we can install the database service into the new namespace:

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

Wait for the application to start:

```
kubectl get all
```{{execute}}

```
NAME                             READY   STATUS    RESTARTS   AGE
pod/petclinic-6997fcbb87-747fr   1/1     Running   0          45s

NAME                 TYPE        CLUSTER-IP       EXTERNAL-IP   PORT(S)   AGE
service/kubernetes   ClusterIP   10.96.0.1        <none>        443/TCP   3m20s
service/petclinic    ClusterIP   10.102.255.123   <none>        80/TCP    45s

NAME                        READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/petclinic   1/1     1            1           45s

NAME                                   DESIRED   CURRENT   READY   AGE
replicaset.apps/petclinic-6997fcbb87   1         1         1       45s
```

Now we can connect to the application. First create an SSH tunnel:

`kubectl port-forward svc/petclinic 8080:80 --address=0.0.0.0`{{execute T1}}

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

That's it! You have an application running in Kubernetes and the database is connected. You can open the UI in your browser by clicking on the "App" tab next to the "Terminal".

Next we will drill into the Petclinic sample and see how all teh pieces fit together.