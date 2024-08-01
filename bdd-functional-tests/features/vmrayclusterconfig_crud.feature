Feature: behaviour driven functional tests for vmray cluster operator

  Scenario: Deploy ray cluster config custom resource
    Given Setup raycluster config
    Given Setup vmray cluster k8s client
      When Submit rayclusterconfig named `ray-cluster`
      When Fetch ray clusterconfig `ray-cluster` from WCP env
      When Check in interval of `30` seconds if head node for raycluster `ray-cluster` is up in `600` seconds
      When Check in interval of `30` seconds if worker nodes for raycluster `ray-cluster` are up in `600` seconds
      When Delete rayclusterconfig named `ray-cluster`
      Then Validate rayclusterconfig was created
      Then Validate correct rayclusterconfig was fetched
      Then Validate rayclusterconfig was deleted
      Then Validate that head node came up for raycluster `ray-cluster`
      Then Validate that desired worker nodes came up for raycluster `ray-cluster`
      Then Log status of raycluster named `ray-cluster` in case of failure
