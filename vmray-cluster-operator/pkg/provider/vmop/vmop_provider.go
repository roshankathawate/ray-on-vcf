// Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package vmop

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	vmopv1 "github.com/vmware-tanzu/vm-operator/api/v1alpha2"
	vmrayv1alpha1 "gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/api/v1alpha1"
	"gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/pkg/provider"
	"gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/pkg/provider/vmop/cloudinit"
	"gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/pkg/provider/vmop/translator"
	vmoputils "gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/pkg/provider/vmop/utils"
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

	// Step 1: Create k8s service account, when its head node deployment.
	// Service account name will be same as clustername.
	if req.HeadNodeStatus == nil {
		if err := vmoputils.CreateServiceAccountAndRole(ctx,
			vmopprovider.kubeClient, req.Namespace, req.ClusterName); err != nil {
			vmopprovider.log.Error(err, "Failed to create service account and role")
			return err
		}
	}

	// Step 2: create secret to hold VM's cloud config init.
	secret, _, err := vmoputils.CreateCloudInitSecret(ctx, vmopprovider.kubeClient, req)
	if err != nil {
		vmopprovider.log.Error(err, "Failed to create cloud init secret")
		return err
	}

	// Create selector & labels to be leveraged by VM service for head node.
	annotationmap := make(map[string]string)
	if req.HeadNodeStatus == nil {
		annotationmap[HeadVMServiceAnnotation] = req.VmName

		// Step 3: Create vmservice for head node.
		ports := make(map[string]int32)

		var port = cloudinit.RayHeadDefaultPort
		if req.HeadNodeConfig.Port != nil {
			port = int32(*req.HeadNodeConfig.Port)
		}

		ports[RayHeadDefaultPortName] = port

		// TODO: Currently dashboard port is set to default one
		// moving forward give users ability to pass it via CRD.
		ports[RayDashboardPortName] = RayDashboardPort
		ports[SshPortName] = SshPort

		err = vmoputils.CreateVMService(ctx, vmopprovider.kubeClient, req.Namespace, req.VmName, ports, annotationmap)
		if err != nil {
			vmopprovider.log.Error(err, "Failed to create VM service")
			return err
		}
	}

	vmclass, err := getVmClass(req.NodeType, req.NodeConfig)
	if err != nil {
		vmopprovider.log.Error(err, "Failed to get vm class from CRs node config")
		return err
	}

	// Step 4: Get VM CRD obj ref using translator functon while consuming VmInfo.
	vm, err := translator.TranslateToVmCRD(req.Namespace,
		req.VmName, secret.ObjectMeta.Name, annotationmap, vmclass, req.NodeConfig)
	if err != nil {
		errmsg := fmt.Sprintf("Failure while translating VM info to VM CRD for %s:%s", req.Namespace, req.VmName)
		vmopprovider.log.Error(err, errmsg)
		return err
	}

	// Step 5: Submit VM CRD to kube-api-server, on successful
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

func (vmopprovider *VmOperatorProvider) Delete(ctx context.Context, namespace string, name string) error {

	// step 1: Delete head node vmservice, for worker this operation
	// will fail but we will simply ignore it.
	if err := vmoputils.DeleteVMService(ctx, vmopprovider.kubeClient, namespace, name); client.IgnoreNotFound(err) != nil {
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
