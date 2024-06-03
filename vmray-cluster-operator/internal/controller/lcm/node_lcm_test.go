// Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package lcm_test

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	vmrayv1alpha1 "gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/api/v1alpha1"
	"gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/internal/controller/lcm"
	mockvmpv "gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/pkg/provider/mock"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	namespace   = "lcm-namespace"
	clustername = "lcm-cn"
	vmname      = "vm-name"
	dockerimg   = "docker-img"
)

func getNodeLcmRequest() lcm.NodeLcmRequest {
	return lcm.NodeLcmRequest{
		Namespace:        namespace,
		Clustername:      clustername,
		Name:             vmname,
		DockerImage:      dockerimg,
		ApiServer:        vmrayv1alpha1.ApiServerInfo{},
		NodeConfigSpec:   vmrayv1alpha1.VMRayNodeConfigSpec{},
		HeadNodeConfig:   vmrayv1alpha1.HeadNodeConfig{},
		WorkerNodeConfig: vmrayv1alpha1.WorkerNodeConfig{},
		HeadNodeStatus:   nil,
		NodeStatus: &vmrayv1alpha1.VMRayNodeStatus{
			Ip:         "",
			Conditions: []metav1.Condition{},
			VmStatus:   "",
			RayStatus:  "",
		},
	}
}

func nodeLifecycleManagerTests() {

	ctx := context.Background()

	Describe("Test Node LCM logic flow from different status", func() {

		Context("Validate LCM with mock VM provider", func() {

			It("Test node deployment without failure", func() {

				provider := mockvmpv.NewMockVmProvider()
				nlcmReq := getNodeLcmRequest()

				// Set mock provider deploy function.
				provider.DeploySetResponse(1, nil)

				// Validate response we get from lcm using provider.
				nlcm := lcm.NewNodeLifecycleManager(provider)
				err := nlcm.ProcessNodeVmState(ctx, nlcmReq)
				Expect(err).To(BeNil())
			})

			It("Test node deployment with already exists failure", func() {

				provider := mockvmpv.NewMockVmProvider()
				nlcmReq := getNodeLcmRequest()

				// Set mock provider deploy function.
				groupRes := schema.GroupResource{Group: "vmoperator.vmware.com", Resource: "virtualmachines"}
				alreadyexistserr := k8serrors.NewAlreadyExists(groupRes, vmname)

				provider.DeploySetResponse(1, alreadyexistserr)
				provider.DeploySetResponse(2, errors.New("Not already exists error"))

				// Validate response we get from lcm using provider.
				nlcm := lcm.NewNodeLifecycleManager(provider)
				err := nlcm.ProcessNodeVmState(ctx, nlcmReq)
				Expect(err).To(BeNil())

				nlcmReq.NodeStatus.VmStatus = ""
				err = nlcm.ProcessNodeVmState(ctx, nlcmReq)
				Expect(err.Error()).To(Equal("Not already exists error"))
				Expect(nlcmReq.NodeStatus.VmStatus).To(Equal(vmrayv1alpha1.FAIL))
			})

			It("Test node deployment, happy run", func() {

				provider := mockvmpv.NewMockVmProvider()
				nlcmReq := getNodeLcmRequest()

				// Set mock provider deploy function.
				provider.DeploySetResponse(1, nil)
				provider.FetchVmStatusSetResponse(1, &vmrayv1alpha1.VMRayNodeStatus{}, nil)
				provider.FetchVmStatusSetResponse(2, &vmrayv1alpha1.VMRayNodeStatus{Ip: "10.10.10.10"}, nil)
				provider.FetchVmStatusSetResponse(3, &vmrayv1alpha1.VMRayNodeStatus{Ip: "10.10.10.10"}, nil)

				// Validate response we get from lcm using provider.
				nlcm := lcm.NewNodeLifecycleManager(provider)
				err := nlcm.ProcessNodeVmState(ctx, nlcmReq)
				Expect(err).To(BeNil())

				err = nlcm.ProcessNodeVmState(ctx, nlcmReq)
				Expect(err).To(BeNil())
				Expect(nlcmReq.NodeStatus.VmStatus).To(Equal(vmrayv1alpha1.INITIALIZED))

				err = nlcm.ProcessNodeVmState(ctx, nlcmReq)
				Expect(err).To(BeNil())
				Expect(nlcmReq.NodeStatus.VmStatus).To(Equal(vmrayv1alpha1.RUNNING))

				err = nlcm.ProcessNodeVmState(ctx, nlcmReq)
				Expect(err).To(BeNil())
				Expect(nlcmReq.NodeStatus.VmStatus).To(Equal(vmrayv1alpha1.RUNNING))
			})

			It("Test node deployment, failure recovery", func() {

				provider := mockvmpv.NewMockVmProvider()
				nlcmReq := getNodeLcmRequest()

				groupRes := schema.GroupResource{Group: "vmoperator.vmware.com", Resource: "virtualmachines"}
				notfounderr := k8serrors.NewNotFound(groupRes, vmname)

				// Set mock provider deploy function.
				provider.DeploySetResponse(1, nil)
				provider.FetchVmStatusSetResponse(1, &vmrayv1alpha1.VMRayNodeStatus{Ip: "10.10.10.10"}, nil)
				provider.FetchVmStatusSetResponse(2, nil, errors.New("Failure to fetch VM status"))
				provider.FetchVmStatusSetResponse(3, nil, notfounderr)
				provider.FetchVmStatusSetResponse(4, &vmrayv1alpha1.VMRayNodeStatus{Ip: "10.10.10.10"}, nil)

				// Validate response we get from lcm using provider.
				nlcm := lcm.NewNodeLifecycleManager(provider)

				// Successful deploy, move status from "" to INITIALIZED.
				err := nlcm.ProcessNodeVmState(ctx, nlcmReq)
				Expect(err).To(BeNil())

				// FetchVMStatus -> call 1, move status from INITIALIZED to RUNNING.
				err = nlcm.ProcessNodeVmState(ctx, nlcmReq)
				Expect(err).To(BeNil())
				Expect(nlcmReq.NodeStatus.VmStatus).To(Equal(vmrayv1alpha1.RUNNING))

				// FetchVMStatus failure -> call 2, move status from RUNNING TO FAIL
				err = nlcm.ProcessNodeVmState(ctx, nlcmReq)
				Expect(err.Error()).To(Equal("Failure to fetch VM status"))
				Expect(nlcmReq.NodeStatus.VmStatus).To(Equal(vmrayv1alpha1.FAIL))

				// FetchVMStatus not found failure -> call 3, move status from Fail TO "" (empty state)
				err = nlcm.ProcessNodeVmState(ctx, nlcmReq)
				Expect(err).To(BeNil())
				Expect(nlcmReq.NodeStatus.VmStatus).To(Equal(vmrayv1alpha1.EMPTY))

			})

			It("Test node deployment, initialize to failure path and back", func() {

				provider := mockvmpv.NewMockVmProvider()
				nlcmReq := getNodeLcmRequest()

				// Set mock provider deploy function.
				provider.FetchVmStatusSetResponse(1, nil, errors.New("Failure to fetch VM status"))
				provider.FetchVmStatusSetResponse(2, &vmrayv1alpha1.VMRayNodeStatus{}, nil)

				// Validate response we get from lcm using provider.
				nlcmReq.NodeStatus.VmStatus = vmrayv1alpha1.INITIALIZED
				nlcm := lcm.NewNodeLifecycleManager(provider)

				err := nlcm.ProcessNodeVmState(ctx, nlcmReq)
				Expect(err.Error()).To(Equal("Failure to fetch VM status"))
				Expect(nlcmReq.NodeStatus.VmStatus).To(Equal(vmrayv1alpha1.FAIL))

				err = nlcm.ProcessNodeVmState(ctx, nlcmReq)
				Expect(err).To(BeNil())
				Expect(nlcmReq.NodeStatus.VmStatus).To(Equal(vmrayv1alpha1.INITIALIZED))

			})

			It("Test node deployment, No IP when VM is running", func() {

				provider := mockvmpv.NewMockVmProvider()
				nlcmReq := getNodeLcmRequest()

				// Set mock provider deploy function.
				provider.FetchVmStatusSetResponse(1, &vmrayv1alpha1.VMRayNodeStatus{}, nil)

				// Validate response we get from lcm using provider.
				nlcmReq.NodeStatus.VmStatus = vmrayv1alpha1.RUNNING
				nlcm := lcm.NewNodeLifecycleManager(provider)

				err := nlcm.ProcessNodeVmState(ctx, nlcmReq)
				Expect(err.Error()).To(Equal("Primary IPv4 not found for vm-name Node"))
				Expect(nlcmReq.NodeStatus.VmStatus).To(Equal(vmrayv1alpha1.FAIL))
			})

			It("Test node deployment, No IP when VM is running", func() {

				provider := mockvmpv.NewMockVmProvider()
				nlcmReq := getNodeLcmRequest()

				// Validate response we get from lcm using provider.
				var invalidstatus vmrayv1alpha1.VMNodeStatus = "invalid"
				nlcmReq.NodeStatus.VmStatus = invalidstatus
				nlcm := lcm.NewNodeLifecycleManager(provider)

				err := nlcm.ProcessNodeVmState(ctx, nlcmReq)
				Expect(err.Error()).To(Equal("lcm detected invalid node status"))
			})
		})
	})
}
