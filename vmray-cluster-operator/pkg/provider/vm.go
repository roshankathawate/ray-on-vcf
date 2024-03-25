// Copyright (c) 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	vmrayv1alpha1 "gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/api/v1alpha1"
)

type VmProvider interface {
	Deploy(vmrayv1alpha1.VMRayNodeConfig) error
	Delete(string) error
	FetchVmStatus(string) vmrayv1alpha1.VMRayNodeStatus
}
