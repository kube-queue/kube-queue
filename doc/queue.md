# Queue

## Motivations

In practical applications, multiple tenants often share a cluster, and resource isolation between tenants is required. We need to support Queue Unit under multiple queues.

### Goals

- Support multiple queue queuing

## Proposal

A "CRD" is needed to interact with the kube-queue. The CRD kind name is subject to change. We use "Queue" in the proposal. The CRD defines the information related to multiple queue queuing. "Queue" is namespace scoped.

```yaml
apiVersion: scheduling.x-k8s.io/v1alpha1
kind: Queue
metadata:
  name: queue
  namespace: queue
spec:
  queuePolicy: Priority # queuing strategy
  priority: 100
  priorityClassName: high-priority
```

The API and Status are described below:

```go
type Queue struct {
metav1.TypeMeta   `json:",inline"`
metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,name=metadata"`

Spec   QueueSpec   `json:"spec,omitempty" protobuf:"bytes,2,name=spec"`
Status QueueStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

type QueueSpec struct {
QueuePolicy         QueuePolicy    `json:"queuePolicy,omitempty" protobuf:"bytes,1,opt,name=queuePolicy`
Priority            *int32         `json:"priority,omitempty" protobuf:"varint,2,opt,name=priority"`
PriorityClassName   string         `json:"priorityClassName,omitempty" protobuf:"bytes,3,opt,name=priorityClassName"`
}

type QueueStatus struct {
// TODO
}

type QueuePolicy string

const (
QueuePolicyFIFO       QueuePolicy = "FIFO"
QueuePolicyPriority   QueuePolicy = "Priority"
)
```

### Lifecycle of CRD

#### Create CRD

#### Update CRD

### Delete CRD

## Use Case
### 1. Create Queue and ResourceQuota for two namespace
```shell
$ kubectl create -f examples/multiqueues/queue1.yaml;kubectl create -f examples/multiqueues/queue2.yaml;kubectl create -f examples/multiqueues/resource_quota_q1.yaml;kubectl create -f examples/multiqueues/resource_quota_q2.yaml
queue.scheduling.x-k8s.io/queue1 created
queue.scheduling.x-k8s.io/queue2 created
resourcequota/queue1 created
resourcequota/queue2 created

$ kubectl get queue -A
NAMESPACE   NAME     AGE
queue1      queue1   4m40s
queue2      queue2   4m7s

$ kubectl get resourcequota  -A -o wide
NAMESPACE                  NAME            AGE     REQUEST                                    LIMIT
queue1                     queue1          2m25s   cpu: 0/4, memory: 0/4Gi
queue2                     queue2          2m10s   cpu: 0/4, memory: 0/4Gi
```
#### 2. Submit tf jobs
```shell
$ kubectl create -f examples/multiqueues/job1_q1.yaml;kubectl create -f examples/multiqueues/job2_q1.yaml;kubectl create -f examples/multiqueues/job1_q2.yaml;kubectl create -f examples/multiqueues/job2_q2.yaml
tfjob.kubeflow.org/job1_q1 created
tfjob.kubeflow.org/job2_q1 created
tfjob.kubeflow.org/job1_q2 created
tfjob.kubeflow.org/job2_q2 created
```

#### 3. Check the status of tf jobs
3.1 At the beginning, only one job creates the pod and runs successfully.
```shell
$ kubectl get tfjob -n queue1
NAME      STATE     AGE
job1_q1   Running   5s
job2_q1   Queuing   5s

$ kubectl get pods -n queue1
NAME               READY   STATUS    RESTARTS   AGE
job1_q1-ps-0       1/1     Running   0          8s
job1_q1-worker-0   1/1     Running   0          8s
job1_q1-worker-1   1/1     Running   0          8s

$ kubectl get tfjob -n queue2
NAME      STATE     AGE
job1_q2   Running   20s
job2_q2   Queuing   20s

$ kubectl get pods -n queue2
NAME               READY   STATUS    RESTARTS   AGE
job1_q2-ps-0       1/1     Running   0          25s
job1_q2-worker-0   1/1     Running   0          25s
job1_q2-worker-1   1/1     Running   0          25s
```
3.2 When the state of job1 is `Succeeded`. Job2 will continue to run.
```shell
$ kubectl get tfjob -n queue1
NAME      STATE       AGE
job1_q1   Succeeded   38s
job2_q1   Running     38s

$ kubectl get pods -n queue1
NAME               READY   STATUS      RESTARTS   AGE
job1_q1-worker-0   0/1     Completed   0          54s
job1_q1-worker-1   0/1     Completed   0          54s
job2_q1-ps-0       1/1     Running     0          22s
job2_q1-worker-0   1/1     Running     0          22s
job2_q1-worker-1   1/1     Running     0          21s

$ kubectl get tfjob -n queue2
NAME      STATE       AGE
job1_q2   Succeeded   39s
job2_q2   Running     39s

$ kubectl get pods -n queue2
NAME               READY   STATUS      RESTARTS   AGE
job1_q2-worker-0   0/1     Completed   0          56s
job1_q2-worker-1   0/1     Completed   0          56s
job2_q2-ps-0       1/1     Running     0          25s
job2_q2-worker-0   1/1     Running     0          25s
job2_q2-worker-1   1/1     Running     0          24s
```

3.3 Finally, the state of the two jobs are `Succeeded`.
```shell
$ kubectl get tfjob -n queue1
NAME      STATE       AGE
job1_q1   Succeeded   71s
job2_q1   Succeeded   71s

$ kubectl get pods -n queue1
NAME               READY   STATUS      RESTARTS   AGE
job1_q1-worker-0   0/1     Completed   0          5m
job1_q1-worker-1   0/1     Completed   0          5m
job2_q1-ps-0       0/1     Completed   0          4m28s
job2_q1-worker-0   0/1     Completed   0          4m28s

$ kubectl get tfjob -n queue2
NAME      STATE       AGE
job1_q2   Succeeded   73s
job2_q2   Succeeded   73s

$ kubectl get pods -n queue2
NAME               READY   STATUS      RESTARTS   AGE
job1_q2-worker-0   0/1     Completed   0          5m
job1_q2-worker-1   0/1     Completed   0          5m
job2_q2-ps-0       0/1     Completed   0          4m30s
job2_q2-worker-0   0/1     Completed   0          4m30s
```

## Implementation History