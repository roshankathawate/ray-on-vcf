// Copyright (c) 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package vmop

import (
	"context"

	vmopv1 "github.com/vmware-tanzu/vm-operator/api/v1alpha2"
	vmrayv1alpha1 "gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/api/v1alpha1"
	"gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/pkg/provider/vmop/translator"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type VmOperatorProvider struct {
	kubeClient client.Client
}

// TODO [remove comment] : Controller manager (i.e. sigs.k8s.io/controller-runtime/pkg/manager)
// houses func GetClient() to get kube client configured with required Config.
func NewVmOperatorProvider(kubeClient client.Client) *VmOperatorProvider {
	return &VmOperatorProvider{
		kubeClient: kubeClient,
	}
}

func (vmopprovider *VmOperatorProvider) Deploy(ctx context.Context, config vmrayv1alpha1.VMRayNodeConfig) error {
	// step 1: Get VM CRD obj ref using translator functon while consuming VmInfo.
	vm, err := translator.TranslateToVmCRD(config)
	if err != nil {
		// TODO: Add logging.
		return err
	}

	// step 2: Submit VM CRD to kube-api-server on successful
	// submisson return back without any error.
	return vmopprovider.kubeClient.Create(ctx, vm)
}

func (vmopprovider *VmOperatorProvider) Delete(ctx context.Context, namespace string, name string) error {
	// step 1: Get VM CRD obj ref using VM's namespace & name. If VM CRD doesnt
	// exist assume it was manually deleted and return with success.
	vm := &vmopv1.VirtualMachine{}

	key := client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}
	if err := vmopprovider.kubeClient.Get(ctx, key, vm); err != nil {
		return client.IgnoreNotFound(err)
	}

	// step 2: If VM CRD exists submit a deletion request to kube-api-server,
	// if submission was successful assume VM will eventually get deleted.
	return vmopprovider.kubeClient.Delete(ctx, vm)
}

func (vmopprovider *VmOperatorProvider) FetchVmStatus(ctx context.Context, namespace string, name string) (*vmrayv1alpha1.VMRayNodeStatus, error) {
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
	return translator.ExtractVmStatus(vm)
}
