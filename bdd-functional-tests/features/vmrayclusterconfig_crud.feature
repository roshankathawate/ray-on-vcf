Feature: behaviour driven functional tests for vmray cluster operator

  Scenario: Deploy vmray nodeconfig custom resource
    Given Setup node config
    Given Setup vmray cluster k8s client
      When Submit nodeconfig named `ray-nc`
      When Fetch ray nodeconfig `ray-nc` from WCP env
      When Delete nodeconfig named `ray-nc`
      Then Validate vmraynodeconfig was created
      Then Validate correct vmraynodeconfig was fetched
      Then Validate vmraynodeconfig was deleted

  Scenario: Deploy ray cluster config custom resource
    Given Setup node config
    Given Setup raycluster config
    Given Setup vmray cluster k8s client
      When Submit nodeconfig named `ray-nc`
      When Submit rayclusterconfig named `ray-cluster`
      When Fetch ray nodeconfig `ray-nc` from WCP env
      When Fetch ray clusterconfig `ray-cluster` from WCP env
      When Delete nodeconfig named `ray-nc`
      When Delete rayclusterconfig named `ray-cluster`
      Then Validate vmraynodeconfig was created
      Then Validate rayclusterconfig was created
      Then Validate correct vmraynodeconfig was fetched
      Then Validate correct rayclusterconfig was fetched
      Then Validate vmraynodeconfig was deleted
      Then Validate rayclusterconfig was deleted
