// Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package vmop

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-logr/logr"
	vmopv1 "github.com/vmware-tanzu/vm-operator/api/v1alpha2"
	vmrayv1alpha1 "github.com/vmware/ray-on-vcf/vmray-cluster-operator/api/v1alpha1"
	"github.com/vmware/ray-on-vcf/vmray-cluster-operator/pkg/provider"
	"github.com/vmware/ray-on-vcf/vmray-cluster-operator/pkg/provider/vmop/cloudinit"
	"github.com/vmware/ray-on-vcf/vmray-cluster-operator/pkg/provider/vmop/translator"
	vmoputils "github.com/vmware/ray-on-vcf/vmray-cluster-operator/pkg/provider/vmop/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	HeadVMServiceAnnotation string = "vmray.kubernetes.io/ray-cluster-head"
	RayHeadDefaultPortName  string = "ray-port"
	RayDashboardPortName    string = "ray-dashboard-port"
	RayDashboardPort        int32  = 8265
	SshPortName             string = "ssh-port"
	SshPort                 int32  = 22
	RayClientPortName       string = "ray-client-port"
	RayClientPort           int32  = 10001
	Protocol_TCP                   = "TCP"
)

type VmOperatorProvider struct {
	kubeClient client.Client
	log        logr.Logger
}

func NewVmOperatorProvider(kubeClient client.Client) *VmOperatorProvider {
	return &VmOperatorProvider{
		kubeClient: kubeClient,
		log:        ctrl.Log.WithName("VmOperatorProvider"),
	}
}

func (vmopprovider *VmOperatorProvider) Deploy(ctx context.Context, req provider.VmDeploymentRequest) error {

	annotationmap := make(map[string]string)

	// Step 1:
	// a. Create k8s service account, when its head node deployment.
	// Service account name will be same as clustername.
	// b. Create selector & labels to be leveraged by VM service for head node.
	if req.HeadNodeStatus == nil {
		if err := vmoputils.CreateServiceAccountAndRole(ctx,
			vmopprovider.kubeClient, req.Namespace, req.ClusterName); err != nil {
			vmopprovider.log.Error(err, "Failed to create service account and role")
			return err
		}
		annotationmap[HeadVMServiceAnnotation] = req.VmName
	}

	// Step 2: create secret to hold VM's cloud config init.
	secret, _, err := vmoputils.CreateCloudInitSecret(ctx, vmopprovider.kubeClient, req)
	if err != nil {
		vmopprovider.log.Error(err, "Failed to create cloud init secret")
		return err
	}

	vmclass, err := getVmClass(req.NodeType, req.NodeConfig)
	if err != nil {
		vmopprovider.log.Error(err, "Failed to get vm class from CRs node config")
		return err
	}

	// Step 3: Get VM CRD obj ref using translator functon while consuming VmInfo.
	vm, err := translator.TranslateToVmCRD(req.Namespace,
		req.VmName, secret.ObjectMeta.Name, annotationmap, vmclass, req.NodeConfig)
	if err != nil {
		errmsg := fmt.Sprintf("Failure while translating VM info to VM CRD for %s:%s", req.Namespace, req.VmName)
		vmopprovider.log.Error(err, errmsg)
		return err
	}

	// Step 4: Submit VM CRD to kube-api-server, on successful
	// submisson return back without any error.
	return vmopprovider.kubeClient.Create(ctx, vm)
}

func getVmClass(nodetype string, nodeconfig vmrayv1alpha1.CommonNodeConfig) (string, error) {
	if nt, ok := nodeconfig.NodeTypes[nodetype]; ok {
		return nt.VMClass, nil
	}
	return "", fmt.Errorf("Invalid node type `%s` requested", nodetype)
}

func (vmopprovider *VmOperatorProvider) DeleteAuxiliaryResources(ctx context.Context,
	namespace, clusterName string) error {

	err := vmoputils.DeleteAllCloudInitSecret(ctx, vmopprovider.kubeClient, namespace, clusterName)
	if err != nil {
		return err
	}
	return vmoputils.DeleteServiceAccountAndRole(ctx, vmopprovider.kubeClient, namespace, clusterName)
}

func deleteVMService(ctx context.Context, kubeclient client.Client, namespace, name string) error {
	key := client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}

	// Check if vmservice exists.
	vmservice := &vmopv1.VirtualMachineService{}
	if err := kubeclient.Get(ctx, key, vmservice); err != nil {
		// If err was NotFound then vmservice is already deleted, return without failure.
		return client.IgnoreNotFound(err)
	}
	return kubeclient.Delete(ctx, vmservice)
}

func (vmopprovider *VmOperatorProvider) Delete(ctx context.Context, namespace string, name string) error {

	// step 1: Delete head node vmservice, for worker this operation
	// will fail but we will simply ignore it.
	if err := deleteVMService(ctx, vmopprovider.kubeClient, namespace, name); client.IgnoreNotFound(err) != nil {
		return err
	}

	// step 2: Get VM CRD obj ref using VM's namespace & name. If VM CRD doesnt
	// exist assume it was manually deleted and return with success.
	key := client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}
	vm := &vmopv1.VirtualMachine{}
	if err := vmopprovider.kubeClient.Get(ctx, key, vm); err != nil {
		return client.IgnoreNotFound(err)
	}

	// step 3: If VM CRD exists submit a deletion request to kube-api-server,
	// if submission was successful assume VM will eventually get deleted.
	return vmopprovider.kubeClient.Delete(ctx, vm)
}

func (vmopprovider *VmOperatorProvider) FetchVmStatus(ctx context.Context,
	namespace string, name string) (*vmrayv1alpha1.VMRayNodeStatus, error) {
	// step 1: Get VM CRD obj ref using VM's namespace & name, if VM CRD doesnt exist then throw error.
	vm := &vmopv1.VirtualMachine{}

	key := client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}
	if err := vmopprovider.kubeClient.Get(ctx, key, vm); err != nil {
		return nil, err
	}

	// step 2: If it does exist leverage translator package to convert VM CRD to VmStatus struct.
	return translator.ExtractVmStatus(vm), nil
}

func createVMService(ctx context.Context, kubeclient client.Client, namespace, name string,
	ports map[string]int32, selector map[string]string) error {

	vmserviceport := []vmopv1.VirtualMachineServicePort{}
	for n, p := range ports {
		v := vmopv1.VirtualMachineServicePort{
			Name:       n,
			Protocol:   Protocol_TCP,
			Port:       p,
			TargetPort: p,
		}
		vmserviceport = append(vmserviceport, v)
	}

	vmservice := &vmopv1.VirtualMachineService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: vmopv1.VirtualMachineServiceSpec{
			Selector: selector,
			Ports:    vmserviceport,
			Type:     vmopv1.VirtualMachineServiceTypeLoadBalancer,
		},
	}
	return kubeclient.Create(ctx, vmservice)
}

func (vmopprovider *VmOperatorProvider) DeployVmService(ctx context.Context,
	req provider.VmDeploymentRequest) (string, error) {

	headvmname := provider.GetHeadNodeName(req.ClusterName, req.Nounce)

	// key for cluster's ray node's VMService object.
	key := client.ObjectKey{
		Namespace: req.Namespace,
		Name:      headvmname,
	}

	// Check if VMService exists and fetch its service ingress IP.
	vmservice := &vmopv1.VirtualMachineService{}
	if err := vmopprovider.kubeClient.Get(ctx, key, vmservice); err != nil {
		if client.IgnoreNotFound(err) == nil {

			annotationmap := map[string]string{
				HeadVMServiceAnnotation: headvmname,
			}

			// Produce vm service port list.
			ports := make(map[string]int32)

			var port = cloudinit.RayHeadDefaultPort
			if req.HeadNodeConfig.Port != nil {
				port = int32(*req.HeadNodeConfig.Port)
			}

			ports[RayHeadDefaultPortName] = port

			// TODO: Currently dashboard port is set to default one
			// moving forward give users ability to pass it via CRD.
			ports[RayDashboardPortName] = RayDashboardPort
			ports[RayClientPortName] = RayClientPort
			ports[SshPortName] = SshPort

			err = createVMService(ctx, vmopprovider.kubeClient, req.Namespace, headvmname, ports, annotationmap)
			if err != nil {
				vmopprovider.log.Error(err, "Failed to create VM service")
				return "", err
			}
		}
		return "", nil
	}

	ingress := vmservice.Status.LoadBalancer.Ingress
	if len(ingress) > 0 {
		return ingress[0].IP, nil
	}
	vmopprovider.log.Info("VM service IP is not assigned for ray head node", "vm", headvmname)
	return "", errors.New("Head node VM service IP is not assigned")
}
