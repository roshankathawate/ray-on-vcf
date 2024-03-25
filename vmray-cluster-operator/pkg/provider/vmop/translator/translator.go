// Copyright (c) 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package translator

import (
	vmopv1 "github.com/vmware-tanzu/vm-operator/api/v1alpha2"
	vmrayv1alpha1 "gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/api/v1alpha1"
)

func TranslateToVmCRD(info vmrayv1alpha1.VMRayNodeConfig) (*vmopv1.VirtualMachine, error) {
	return &vmopv1.VirtualMachine{}, nil
}

func ExtractVmStatus(vm *vmopv1.VirtualMachine) (*vmrayv1alpha1.VMRayNodeStatus, error) {
	return &vmrayv1alpha1.VMRayNodeStatus{}, nil
}
