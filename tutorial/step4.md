
Now we are going to look at how the [Petclinic](https://github.com/spring-projects/spring-petclinic) sample works.

Have a look at the deployment manifest at `samples/petclinic.yaml`{{open}}. You will notice that it has a few features that weren't there in the simple `demo` app. Firstly it has to bind to the MySQL service, so the `Microservice` resource has a reference to the database under its `spec.bindings`:

```
kind: Microservice
metadata:
  name: petclinic
spec:
  image: dsyer/petclinic
  bindings:
  - services/mysql
  ...
```

The binding is a namespaced reference to another CRD, the `ServiceBinding`. We can inspect the `ServiceBinding`:

```
kubectl get servicebinding mysql -n services
```{{execute}}

```
NAME    BOUND
mysql   [default/petclinic]
```

We can tell from this summary that there is one `ServiceBinding` and it is currently bound to one `Microservice` (in the "default" namespace). If you want all the details of the manifest youc can look at the YAML `kubectl get servicebinding mysql -n services -o yaml`{{execute}}.

The same YAML manifest is available under `samples/mysql/binding.yaml`{{open}} if you want to open it up and see precisely what was deployed. It has the form of a `PodTemplateSpec`, which is merged with the application's own `PodTemplateSpec` when the `Deployment` is generated. If you look at the YAML for the active `Deployment` you will see the same stuff in the `Pod` spec `kubectl get deployment petclinic -o yaml`{{execute}}.

Most of the `samples/mysql/binding.yaml`{{open}} is taken up with an init container (which runs before the main application) and the configuration it needs to set up an `application.properties` file in a form that Spring Boot will recognise.

You can apply multiple bindings to a single `Microservice` and each of them is merged using the Kubernetes standard JSON merge semantics. The way that various fields behave when patched can be read in the Go docs, e.g. for [`PodSpec`](https://godoc.org/k8s.io/api/core/v1#PodSpec) you will see

```
type PodSpec struct {
...
    Containers []Container `json:"containers" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,2,rep,name=containers"`
...
}
```

Note that the `patchStrategy` for `Containers` is "merge", with a `patchMergeKey` of "name". That means that additional containers can be added cleanly as long as they have unique names. We would have to look at the definition of the `Container` resource to see what happens to a container that is merged with itself (one with the same name). Some properties have an explicit `patchStrategy` and some do not. The default is to overwrite.

The `ServiceBinding` resource has a special feature influenced by the way Spring Boot has configuration properties that are arrays, expressed as a comma-separated value generally. An example would be `SPRING_CONFIG_LOCATION` which is the search path for `application.properties`. The `samples/mysql/binding.yaml`{{open}} specifies this as an actual array in YAML:

```
apiVersion: spring.io/v1
kind: ServiceBinding
metadata:
  name: mysql
spec:
  env:
  - name: SPRING_CONFIG_LOCATION
    values:
    - classpath:/
    - file:///etc/config/
...
```

The `env` entries at the top of the `spec` are special: they allow multiple values to be accumulated over multiple bindings, all of which are joined together using a comma in the final generated `Deployment`. If you look in the `petclinic` deployment you will see that `SPRING_CONFIG_LOCATION` is a comma-separated value.

The rest of the `samples/petclinic.yaml`{{open}} spec concerns environment variables that adapt the vanilla Petclinic to the Kubernetes cluster. For example:

```
kind: Microservice
metadata:
  name: petclinic
spec:
  image: dsyer/petclinic
  template:
    spec:
      containers:
      - name: app
        env:
        - name: MANAGEMENT_ENDPOINTS_WEB_BASEPATH
          value: /actuator
...
```

Here we reset the `/actuator` endpoint path to the default value (the Petclinic exposes them on `/manage` by default). This isn't mandatory to get the application running, but it helps later if we want to use standard liveness and readiness probes for the generated `Container`. There are a handful of other `env` entries that make things work smoothly for the demo.