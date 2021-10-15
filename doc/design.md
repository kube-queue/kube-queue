# Design

## Architecture

Kube-Queue is designed with compliance of micro-service, making extensibility as one its priorities.

The overall system can be separated into 3 parts:

1. Queue Controller
2. Queue Scheduler
3. Extension Servers

Since the `QueueUnits` are stored as CRs ([Custom Resources](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)), the **Queue Controller** listens to the APIServer and manages multiple queues regarding QueueUnit-related events.

The **Queue Scheduler** watches these queues and decides which job, represented by a QueueUnit, should be released. The processing of QueueUnits insides the Queue Scheduler consists of one or more plugins registered in `pkg/framework/plugins/registry.go`. By default, the resource quota plugin is used to determine whether sufficient resource left. Developers can replace it with customized script.

The real job CRs like TFJob, MPIJob are monitotred and will be updated by corresponding **Extension Server**s. When a new job is created, the **Extension Server** will post a `QueueUnit` in APIServer and update the job when the `QueueUnit` is dequeued.

*To support Kube-queue in operators, subtle modifications will be introduced.*

![arch](./img/architecture-updated.jpg)
