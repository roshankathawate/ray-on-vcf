## BDD FTs to validate vmray cluster operator

We are leveraging behavior-driven framework [behave](https://github.com/behave/behave) to write FTs which will validate operators logical sanity in WCP env.

## How to these run tests ?

### Initialize environment:
1. Go to functional tests root directory: `cd bdd-functional-tests`
2. python -m venv ray-bdd-fts
3. source ray-bdd-fts/bin/activate

### leverage behave framework to runs tests:
1. from directory vmray/bdd-functional-tests run command: `behave`

### Dev notes:
1. [Optional] To create/update requirements.txt after adding/installing new dependency, run following command: `pip freeze > requirements.txt`
2. Install [black](https://pypi.org/project/black/) linter for formatting.


### Tests:
Before we run the tests, make sure your supervisior master node is reachable with valid context:
1. https://docs.vmware.com/en/VMware-vSphere/7.0/vmware-vsphere-with-tanzu/GUID-F5114388-1838-4B3B-8A8D-4AE17F33526A.html

#### Running test locally:
1. From functional tests root directory activate the python env that was created: `source ray-bdd-fts/bin/activate`
2. Create a bdd-test.env file with required env variables:
   ```
   KUBE_CONFIG_FILE=<kubeconfig.yaml>
   VMI=<VMI_NAME>
   NAMESPACE=<NAMESPACE_NAME>
   CPVM_IP=<CPVM_IP>
   VM_CLASS=<VM_CLASS_NAME>
   STORAGE_CLASS=<STORAGE_CLASS_NAME>
   ```
   Note: KUBE_CONFIG_FILE should contain path to k8s config file to be leveraged by client to make k8s API calls to CPVM.
3. Set those env variables: `export $(tail -n +3 bdd-test.env | xargs -0)` # tail -n +3 skips first two lines of env file.
4. Go to root directory, and run behave cmd as mentioned above.
5. Running it using ray's docker image, in gateway VM:
   ```
   export BUILD_ENV_FILE=bdd-test.env
   export RAY_IMAGE=rayproject/ray:latest

	docker run -v ${PWD}/bdd-functional-tests:/bdd-functional-tests \
			   -v ${BUILD_ENV_FILE}:/bdd-functional-tests/build.env \
			   -v ${DEV_KUBECONFIG}:/bdd-functional-tests/kubeconfig \
			   -e NAMESPACE="${BDD_NAMESPACE}" \
			   ${RAY_IMAGE} bash -c "source /bdd-functional-tests/build.env; pip install behave; KUBE_CONFIG_FILE=/bdd-functional-tests/kubeconfig behave /bdd-functional-tests/features"
   ```
