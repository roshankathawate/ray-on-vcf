Feature: behaviour driven functional tests for vmray cluster operator

  Scenario: Deploy ray cluster config custom resource
    Given Setup raycluster config
    Given Setup vmray cluster k8s client
    Given Create validation container for cluster `ray-cluster`
      When Submit rayclusterconfig named `ray-cluster`
      When Fetch ray clusterconfig `ray-cluster` from WCP env
      When Check in interval of `30` seconds if head node for raycluster `ray-cluster` is up in `600` seconds
      When Check in interval of `30` seconds if worker nodes for raycluster `ray-cluster` are up in `900` seconds
      When Submit a job named `sleep` to ray cluster `ray-cluster`
      When Delete rayclusterconfig named `ray-cluster`
      Then Log status of raycluster named `ray-cluster` in case of failure
      Then Validate operations performed on raycluster `ray-cluster`
