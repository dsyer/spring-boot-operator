
Quickly create a deployment manifest for a simple application on Kubernetes:

```
cat > deployment.yaml << EOF
apiVersion: spring.io/v1
kind: Microservice
metadata:
  name: demo
spec:
  image: springguides/demo
EOF
```{{execute}}

Then deploy to Kubernetes

`kubectl apply -f deployment.yaml`{{execute}}

and check the app is running 

`kubectl get all`{{execute}}

Sample output:

```
NAME                        READY   STATUS    RESTARTS   AGE
pod/demo-7b4cfc5767-24qgd   0/1     Running   0          6s

NAME                 TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)    AGE
service/demo         ClusterIP   10.97.182.253   <none>        8080/TCP   7s
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

That it! You have an application running in Kubernetes.

`echo "Send Ctrl+C to kill the container"`{{execute T1 interrupt}}
