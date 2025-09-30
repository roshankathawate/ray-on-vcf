// Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
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
	labels map[string]string,
	vmclass string,
	nodeconfig vmrayv1alpha1.CommonNodeConfig) (*vmopv1.VirtualMachine, error) {
	return &vmopv1.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vmName,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: vmopv1.VirtualMachineSpec{
			ImageName:    nodeconfig.VMImage,
			ClassName:    vmclass,
			PowerState:   vmopv1.VirtualMachinePowerStateOn,
			StorageClass: nodeconfig.StorageClass,
			Network:      nodeconfig.Network,
			Bootstrap: &vmopv1.VirtualMachineBootstrapSpec{
				CloudInit: &vmopv1.VirtualMachineBootstrapCloudInitSpec{
					RawCloudConfig: &vmopv1common.SecretKeySelector{
						Name: cloudConfigSecretName,
						Key:  cloudinit.CloudInitConfigUserDataKey,
					},
				},
			},
		},
	}, nil
}

func ExtractVmStatus(vm *vmopv1.VirtualMachine) *vmrayv1alpha1.VMRayNodeStatus {
	var ip string
	// extract IP from VM CR.
	if vm.Status.Network != nil {
		ip = vm.Status.Network.PrimaryIP4
	}
	return &vmrayv1alpha1.VMRayNodeStatus{
		Ip:         ip,
		Conditions: vm.Status.Conditions,
		// status change depends on previous status of the VM & ray process.
		VmStatus:  "",
		RayStatus: "",
	}
}
