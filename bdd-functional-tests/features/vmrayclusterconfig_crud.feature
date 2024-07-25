Feature: behaviour driven functional tests for vmray cluster operator

  Scenario: Deploy ray cluster config custom resource
    Given Setup raycluster config
    Given Setup vmray cluster k8s client
      When Submit rayclusterconfig named `ray-cluster`
      When Fetch ray clusterconfig `ray-cluster` from WCP env
      When Delete rayclusterconfig named `ray-cluster`
      Then Validate rayclusterconfig was created
      Then Validate correct rayclusterconfig was fetched
      Then Validate rayclusterconfig was deleted
