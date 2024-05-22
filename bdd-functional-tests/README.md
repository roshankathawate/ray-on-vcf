## BDD FTs to validate vmray cluster operator

We are leveraging behavior-driven framework [behave](https://github.com/behave/behave) to write FTs which will validate operators logical sanity in WCP env.

## How to these run tests ?

### Initialize environment:
1. python -m venv ray-bdd-fts
2. source ray-bdd-fts/bin/activate
3. pip install -r requirements.txt

### leverage behave framework to runs tests:
1. from directory vmray/bdd-functional-tests run command: `behave`

### Dev notes:
1. To update requirements.txt after adding/installing new dependency, run following command: `pip freeze > requirements.txt`
2. Install [black](https://pypi.org/project/black/) linter for formatting.


### Tests:
Before we run the tests, make sure your supervisior master node is reachable with valid context:
1. https://docs.vmware.com/en/VMware-vSphere/7.0/vmware-vsphere-with-tanzu/GUID-F5114388-1838-4B3B-8A8D-4AE17F33526A.html

#### Common env parameters for all tests: 
1. NAMESPACE (required), Namespace where you will deploy k8s custom resources.
2. KUBE_CONFIG_FILE (optional), k8s config file to be leveraged by client to make k8s API calls.

#### Test specific env parameters:
1. cluster_deployment feature:
   a. Submit nodeconfig custom resource and validate happy path submisson -> `VM_IMAGE` set VMI produced after associating ubuntu image from content library to a namespace.
   Example run: `KUBE_CONFIG_FILE=<kubeconfig.yaml> VM_IMAGE=vmi-ca4ae5c8de892e539 NAMESPACE=bdd-test behave features/vmraynodeconfig_crud.feature`