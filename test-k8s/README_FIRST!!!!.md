Before starting the assignment, I would like to mention that, I used to write a blog and some sample code 
for CRD & Custom Controller.

1. the blog(in Chinese) [link](https://blog.csdn.net/weiyuanke/article/details/97938299)
2. the code: [link](https://github.com/weiyuanke/PodInfoLookup)

The above Custom Controller does something like this:
1. use DynamicClient to manipulate custom resource -- "Testcr"
2. use ClientSet to access k8s api server
3. construct a podController to watch any "Add/Delete" operations to Pod
4. when such Add or Delete operation happen, "Change" will push to workQueue;
5. a consumer go routing consume "Change" in workQueue and use dynamicClient to create or delete a "Testcr" resource.



how to run the code

```
$go get github.com/weiyuanke/PodInfoLookup

$go run main.go --kubeconfig=/Users/yuankewei/.kube/config
```

terminal output

```
#when a pod created: kubectl apply -f yaml/pod.yaml, terminal outputs:

add:  default/nginx

current list:
&{map[kind:Testcr metadata:map[generation:1 name:29b36e2c6835dded8a115aee874d1ddc resourceVersion:113140 selfLink:/apis/stable.example.com/v1/testcrs/29b36e2c6835dded8a115aee874d1ddc uid:a2440f67-9ca4-11e9-9f23-080027f1737d creationTimestamp:2019-07-02T08:37:32Z] spec:map[podip:nginx podkey:a242bc2d-9ca4-11e9-9f23-080027f1737d podname:29b36e2c6835dded8a115aee874d1ddc poduid:] apiVersion:stable.example.com/v1]}
```

--------------------------------------------

--------------------------------------------

## In the process of trying to run the sample-controller, following error encountered
```
yuankes-MacBook-Ali:sample-controller yuankewei$ go build -o sample-controller .
# k8s.io/utils/trace
../utils/trace/trace.go:100:57: invalid operation: stepThreshold == 0 || stepDuration > stepThreshold || klog.V(4) (mismatched types bool and klog.Verbose)
../utils/trace/trace.go:112:56: invalid operation: stepThreshold == 0 || stepDuration > stepThreshold || klog.V(4) (mismatched types bool and klog.Verbose)
# k8s.io/client-go/transport
../client-go/transport/round_trippers.go:70:11: cannot convert klog.V(9) (type klog.Verbose) to type bool
../client-go/transport/round_trippers.go:72:11: cannot convert klog.V(8) (type klog.Verbose) to type bool
../client-go/transport/round_trippers.go:74:11: cannot convert klog.V(7) (type klog.Verbose) to type bool
../client-go/transport/round_trippers.go:76:11: cannot convert klog.V(6) (type klog.Verbose) to type bool
```

## the solution is to checkout another branch of klog
```
yuankes-MacBook-Ali:klog yuankewei$ pwd
/Users/yuankewei/go/src/k8s.io/klog
yuankes-MacBook-Ali:klog yuankewei$ git checkout -b release-1.x origin/release-1.x
Branch 'release-1.x' set up to track remote branch 'release-1.x' from 'origin'.
Switched to a new branch 'release-1.x'
yuankes-MacBook-Ali:klog yuankewei$
```

then everything works fun
```
yuankes-MacBook-Ali:sample-controller yuankewei$ pwd
/Users/yuankewei/go/src/k8s.io/sample-controller
yuankes-MacBook-Ali:sample-controller yuankewei$ go build -o sample-controller .
yuankes-MacBook-Ali:sample-controller yuankewei$
```