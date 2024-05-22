Feature: behaviour driven functional tests for vmray cluster operator 

  Scenario: [test-1] Deploy vmray nodeconfig custom resource
    Given Setup node config
    Given Setup vmray cluster k8s client
      When Submit nodeconfig named `ray-nc`
      When Fetch ray nodeconfig `ray-nc` from WCP env
      When Delete nodeconfig named `ray-nc`
      Then Validate vmraynodeconfig was created
      Then Validate correct vmraynodeconfig was fetched
      Then Validate vmraynodeconfig was deleted