// Copyright (c) 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package translator

import (
	vmopv1 "github.com/vmware-tanzu/vm-operator/api/v1alpha2"
	vmopv1common "github.com/vmware-tanzu/vm-operator/api/v1alpha2/common"
	vmrayv1alpha1 "gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/api/v1alpha1"
	"gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/pkg/provider/vmop/cloudinit"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TranslateToVmCRD(namespace,
	vmName,
	cloudConfigSecretName string,
	spec vmrayv1alpha1.VMRayNodeConfigSpec) (*vmopv1.VirtualMachine, error) {
	return &vmopv1.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vmName,
			Namespace: namespace,
		},
		Spec: vmopv1.VirtualMachineSpec{
			ImageName:    spec.VMImage,
			ClassName:    spec.VMClass,
			PowerState:   vmopv1.VirtualMachinePowerStateOn,
			StorageClass: spec.StorageClass,
			// TODO: Networking, should we let user select preferred network
			// or create a new overlay ourself?
			// Network: &vmopv1.VirtualMachineNetworkSpec{},
			Bootstrap: &vmopv1.VirtualMachineBootstrapSpec{
				CloudInit: &vmopv1.VirtualMachineBootstrapCloudInitSpec{
					RawCloudConfig: &vmopv1common.SecretKeySelector{
						Name: cloudConfigSecretName,
						Key:  cloudinit.CloudConfigUserDataKey,
					},
				},
			},
		},
	}, nil
}

func ExtractVmStatus(vm *vmopv1.VirtualMachine) *vmrayv1alpha1.VMRayNodeStatus {
	return &vmrayv1alpha1.VMRayNodeStatus{
		Name:       vm.ObjectMeta.Name,
		Ip:         vm.Status.Network.PrimaryIP4,
		Conditions: vm.Status.Conditions,
		// status change depends on previous status of the VM & ray process.
		VmStatus:  "",
		RayStatus: "",
	}
}
