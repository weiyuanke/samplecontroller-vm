>Read and Learn, Analyze how sample-controller (https://github.com/kubernetes/sample-controller) works.
the sample controller does things:

1. Sample Controller watch for any changes to CRD(Foo), for example, adding Custom Resource,
updating Custom Resource or Deleting Custom Resource.
2. Sample Contoller reads the spec fields(deploymentName/replicas), and create a nginx deployment
with the specified replicas.
3. when custom resource is deleted or updated, the coresponding deployment is also deleted or updated.

for example, create a custom resource of type Foo, 

```
apiVersion: samplecontroller.k8s.io/v1alpha1
kind: Foo
metadata:
  name: wyktest
spec:
  deploymentName: wyktst-deployment-foo
  replicas: 2

yuankes-MacBook-Ali:sample-controller yuankewei$ kubectl apply -f artifacts/examples/example-foo.yaml
foo.samplecontroller.k8s.io/wyktest unchanged
yuankes-MacBook-Ali:sample-controller yuankewei$ kubectl get foo wyktest
NAME      AGE
wyktest   80s
yuankes-MacBook-Ali:sample-controller yuankewei$
```

the deploment with the specified name and replicas is created:

```
yuankes-MacBook-Ali:sample-controller yuankewei$ kubectl get deployments
NAME                    READY   UP-TO-DATE   AVAILABLE   AGE
wyktst-deployment-foo   2/2     2            2           104s
yuankes-MacBook-Ali:sample-controller yuankewei$ kubectl get pods
NAME                                     READY   STATUS    RESTARTS   AGE
wyktst-deployment-foo-5645bdc957-5mtj5   1/1     Running   0          117s
wyktst-deployment-foo-5645bdc957-x8z7t   1/1     Running   0          117s
```

adjust the replicas in custom resource wyktest

```
yuankes-MacBook-Ali:sample-controller yuankewei$ vi artifacts/examples/example-foo.yaml
yuankes-MacBook-Ali:sample-controller yuankewei$ kubectl apply -f artifacts/examples/example-foo.yaml
foo.samplecontroller.k8s.io/wyktest configured
yuankes-MacBook-Ali:sample-controller yuankewei$ cat artifacts/examples/example-foo.yaml
apiVersion: samplecontroller.k8s.io/v1alpha1
kind: Foo
metadata:
  name: wyktest
spec:
  deploymentName: wyktst-deployment-foo
  replicas: 3
yuankes-MacBook-Ali:sample-controller yuankewei$ kubectl get deployment
NAME                    READY   UP-TO-DATE   AVAILABLE   AGE
wyktst-deployment-foo   3/3     3            3           4m7s
yuankes-MacBook-Ali:sample-controller yuankewei$
```

when the custom resource(wyktest) is deleted, the deployment is also deleted.

```
yuankes-MacBook-Ali:sample-controller yuankewei$ kubectl delete foo wyktest
foo.samplecontroller.k8s.io "wyktest" deleted
yuankes-MacBook-Ali:sample-controller yuankewei$ kubectl get deployment
No resources found.
yuankes-MacBook-Ali:sample-controller yuankewei$
```

>1.1. Could you explain each file which is in https://github.com/kubernetes/sample-controller/tree/master/artifacts/examples

crd.yaml:

```
defined a custom resource type called "Foo"
references: https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/
```

example-foo.yaml:

```
create a concrete custom resource of Type "Foo".
when created, we can use kubectl to get a list of resource of type Foo.

yuankes-MacBook-Ali:sample-controller yuankewei$ kubectl get foo wyktest
NAME      AGE
wyktest   80s
```

crd-validation.yaml:

```
like crd.yaml, but with extra validation config.
validation:
    openAPIV3Schema:
      properties:
        spec:
          properties:
            replicas:
              type: integer
              minimum: 1
              maximum: 10
OpenAPIV3Schema validation conventions is used to check the data in custom resource.
here we require the value of replicas is between [1,10]

reference: https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/#validation
```

crd-status-subresource.yaml

```
like crd.yaml, but with subresources and the status subresource is enabled
https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/#subresources
```

>1.2. We tried to run sample-controller and execute following command but we got following error.
Could you explain why this command is failing? and Give us how we can fix it?
"kubectl apply -f artifacts/examples/example-foo.yaml && kubectl apply -f artifacts/examples/crd.yaml"
Error from server (NotFound): error when creating "artifacts/examples/example-foo.yaml": the server could not find the requested resource (post foos.samplecontroller.k8s.io)

The cluster cannot recognize the resource type specified in example-foo.yaml, 
this is probably because the Custom Resource "Foo" is not created or deleted.
we can recreate Custom Resource "Foo" by running: apply -f artifacts/examples/crd.yaml

Another possibility is that your k8s cluster version is outdated and you better update k8s cluster version.

>1.3. Please give us explanation about "what this sample-controller do" and "how this sample-controller work" when user create Foo resource

As I mentioned in the beginning, sample-controller watch for changes on Custom Resource "Foo", and create/update/delete nginx deployment
with the specified replicas in Custom Resource.

1. when a resource of type "Foo" is created, corresponding nginx deployment is created, with replicas equal to value of "replicas" field in 
custom resource.
2. when custom resource is updated, nginx deployment is adjusted.
3. when custom resource is deleted, nginx deployment is deleted.