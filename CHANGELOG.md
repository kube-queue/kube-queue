# Kube-queue Release Notes

## v0.1.0
### Features
- Build a framework to support different queueing policies and quota systems
- Support two basic queueing policies for the elements in single queue: FIFO、Priority
- Support dynamic adjustment of job priority in queue
- Integrate with the ResourceQuota in Kubernetes
- Integrate with TFJob, PyTorchJob for distribute deep learning training
- Integrate with ET-operator for elastic training 
