
# Recommendations for performance tuning

It is recommended to use one of the autoscalers at a time. Confiuring both horizontal and vertical autoscaler can cause unpredictable behavior.

When the model server is deployed with pods of restricted CPUn allocation, it is recommended to enable the cluster CPU manager.
On Kubernetes it can be enabled like described in [documentation](https://kubernetes.io/docs/tasks/administer-cluster/cpu-management-policies/) by setting a static `--cpu-manager-policy`.
In Openshift it can be enabled based on [documented](https://docs.openshift.com/container-platform/4.10/scalability_and_performance/using-cpu-manager.html).

While the model server deployment has restricted CPU resources, it is important to configure adequate number of execution streams.
The optimal latency results with small numberl or parallel clients is expected with a single execution stream: plugin_config set to `{\"CPU_THROUGHPUT_STREAMS\":1}`. The higher throughput results with high muli-concurrency is expected with the number of execution streams equal to the number of assigned CPU cores. 

While planning the service loadbalancing, take into account gRPC connection preserving behavior. With the default cluster loadbalancing, gRPC connection between the client and the replica is maintained. It means a single client cannot distribute the calls on mulitple replicas. A solution might be to use multiple clients in parallel or using Mesh service which dispatches individual inference calls. An alternative might be also using REST API, which migth be less effective for big data volumes. 