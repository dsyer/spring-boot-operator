
Quickly inspect the deployment manifest at `samples/demo.yaml`{{open}} for a simple application on Kubernetes. It's only a few lines because with a few opinions baked in, it only really needs to know the image location:

```
apiVersion: spring.io/v1
kind: Microservice
metadata:
  name: demo
spec:
  image: springguides/demo
```

Then deploy to Kubernetes

`kubectl apply -f samples/demo.yaml`{{execute}}

and check the app is running 

`kubectl get all`{{execute}}

The 6 lines of YAML have created for you a `Deployment` and a `Service`, and the `Service` is exposed on port 80 in the cluster. Sample output:

```
NAME                        READY   STATUS    RESTARTS   AGE
pod/demo-7b4cfc5767-24qgd   0/1     Running   0          6s

NAME                 TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)    AGE
service/demo         ClusterIP   10.97.182.253   <none>        80/TCP   7s
service/kubernetes   ClusterIP   10.96.0.1       <none>        443/TCP    9m22s

NAME                   READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/demo   0/1     1            0           7s

NAME                              DESIRED   CURRENT   READY   AGE
replicaset.apps/demo-7b4cfc5767   1         1         0       7s
master
```

Now we can connect to the application. First create an SSH tunnel:

`kubectl port-forward svc/demo 8080:80`{{execute T1}}

and then you can verify that the app is running:

`curl localhost:8080/actuator/health`{{execute T2}}

```
{"status":"UP"}
```

That it! You have an application running in Kubernetes. You can inspect the YAML that was generated in the controller, e.g.

```
kubectl get deployment demo -o yaml
```

Finally, clean up the deployment and tear down the tunnel ready for the next step:

`kubectl delete microservices --all`{{execute T1 interrupt}}
