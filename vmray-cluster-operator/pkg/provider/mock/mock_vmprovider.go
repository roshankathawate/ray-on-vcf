// Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package mock

import (
	"context"
	"errors"

	vmrayv1alpha1 "github.com/vmware/ray-on-vcf/vmray-cluster-operator/api/v1alpha1"
	"github.com/vmware/ray-on-vcf/vmray-cluster-operator/pkg/provider"
)

/*
Create VmProvider mock with response tracker to be leveraged in unittest.

For each function of provider we have set of functions which mock the said
implementation of provider with mock tracker to return set Responses &
capture Requests.
*/

type MockNamedNamespaceRequest struct {
	Namespace string
	Name      string
}

type mockFetchVmStatusResponse struct {
	Status *vmrayv1alpha1.VMRayNodeStatus
	Error  error
}

type mockDeployVmServiceResponse struct {
	Ip    string
	Error error
}

type MockVmProvider struct {
	deployFuncResponse  map[int]error
	deployFuncRequest   map[int]provider.VmDeploymentRequest
	deployFuncCallCount int

	deleteFuncResponse  map[int]error
	deleteFuncRequest   map[int]MockNamedNamespaceRequest
	deleteFuncCallCount int

	fetchVmStatusFuncResponse  map[int]mockFetchVmStatusResponse
	fetchVmStatusFuncRequest   map[int]MockNamedNamespaceRequest
	fetchVmStatusFuncCallCount int

	deleteAuxiliaryResourcesFuncResponse  map[int]error
	deleteAuxiliaryResourcesFuncRequest   map[int]MockNamedNamespaceRequest
	deleteAuxiliaryResourcesFuncCallCount int

	deployVmServiceFuncResponse  map[int]mockDeployVmServiceResponse
	deployVmServiceFuncRequest   map[int]provider.VmDeploymentRequest
	deployVmServiceFuncCallCount int
}

func NewMockVmProvider() *MockVmProvider {
	return &MockVmProvider{
		deployFuncResponse:  make(map[int]error),
		deployFuncRequest:   make(map[int]provider.VmDeploymentRequest),
		deployFuncCallCount: 0,

		deleteFuncResponse:  make(map[int]error),
		deleteFuncRequest:   make(map[int]MockNamedNamespaceRequest),
		deleteFuncCallCount: 0,

		fetchVmStatusFuncResponse:  make(map[int]mockFetchVmStatusResponse),
		fetchVmStatusFuncRequest:   make(map[int]MockNamedNamespaceRequest),
		fetchVmStatusFuncCallCount: 0,

		deleteAuxiliaryResourcesFuncResponse:  make(map[int]error),
		deleteAuxiliaryResourcesFuncRequest:   make(map[int]MockNamedNamespaceRequest),
		deleteAuxiliaryResourcesFuncCallCount: 0,

		deployVmServiceFuncResponse:  make(map[int]mockDeployVmServiceResponse),
		deployVmServiceFuncRequest:   make(map[int]provider.VmDeploymentRequest),
		deployVmServiceFuncCallCount: 0,
	}
}

// Mock tracker & implmenetation for `Deploy` function.
func (mvp *MockVmProvider) Deploy(ctx context.Context, req provider.VmDeploymentRequest) error {
	mvp.deployFuncCallCount = mvp.deployFuncCallCount + 1

	mvp.deployFuncRequest[mvp.deployFuncCallCount] = req
	if err, ok := mvp.deployFuncResponse[mvp.deployFuncCallCount]; ok {
		return err
	}
	return errors.New("no response set for function `Deploy`")
}

func (mvp *MockVmProvider) DeploySetResponse(callcount int, err error) {
	mvp.deployFuncResponse[callcount] = err
}

func (mvp *MockVmProvider) DeployGetRequest(callcount int) provider.VmDeploymentRequest {
	return mvp.deployFuncRequest[callcount]
}

// Mock tracker & implmenetation for `Delete` function.
func (mvp *MockVmProvider) Delete(ctx context.Context, namespace string, name string) error {
	mvp.deleteFuncCallCount = mvp.deleteFuncCallCount + 1

	mvp.deleteFuncRequest[mvp.deleteFuncCallCount] = MockNamedNamespaceRequest{
		Namespace: namespace,
		Name:      name,
	}
	if err, ok := mvp.deleteFuncResponse[mvp.deleteFuncCallCount]; ok {
		return err
	}
	return errors.New("no response set for function `Delete`")
}

func (mvp *MockVmProvider) DeleteSetResponse(callcount int, err error) {
	mvp.deleteFuncResponse[callcount] = err
}

func (mvp *MockVmProvider) DeleteGetRequest(callcount int) MockNamedNamespaceRequest {
	return mvp.deleteFuncRequest[callcount]
}

// Mock tracker & implmenetation for `FetchVmStatus` function.
func (mvp *MockVmProvider) FetchVmStatus(ctx context.Context,
	namespace string, name string) (*vmrayv1alpha1.VMRayNodeStatus, error) {
	mvp.fetchVmStatusFuncCallCount = mvp.fetchVmStatusFuncCallCount + 1

	mvp.fetchVmStatusFuncRequest[mvp.fetchVmStatusFuncCallCount] = MockNamedNamespaceRequest{
		Namespace: namespace,
		Name:      name,
	}
	if resp, ok := mvp.fetchVmStatusFuncResponse[mvp.fetchVmStatusFuncCallCount]; ok {
		return resp.Status, resp.Error
	}
	return nil, errors.New("no response set for function `FetchVmStatus`")
}

func (mvp *MockVmProvider) FetchVmStatusSetResponse(callcount int, status *vmrayv1alpha1.VMRayNodeStatus, err error) {
	mvp.fetchVmStatusFuncResponse[callcount] = mockFetchVmStatusResponse{
		Status: status,
		Error:  err,
	}
}

func (mvp *MockVmProvider) FetchVmStatusGetRequest(callcount int) MockNamedNamespaceRequest {
	return mvp.fetchVmStatusFuncRequest[callcount]
}

// Mock tracker & implmenetation for `DeleteAuxiliaryResources` function.
func (mvp *MockVmProvider) DeleteAuxiliaryResources(ctx context.Context, namespace string, clustername string) error {
	mvp.deleteAuxiliaryResourcesFuncCallCount = mvp.deleteAuxiliaryResourcesFuncCallCount + 1

	mvp.deleteAuxiliaryResourcesFuncRequest[mvp.deleteAuxiliaryResourcesFuncCallCount] = MockNamedNamespaceRequest{
		Namespace: namespace,
		Name:      clustername,
	}
	if err, ok := mvp.deleteAuxiliaryResourcesFuncResponse[mvp.deleteAuxiliaryResourcesFuncCallCount]; ok {
		return err
	}
	return errors.New("no response set for function `DeleteAuxiliaryResources`")
}

func (mvp *MockVmProvider) DeleteAuxiliaryResourcesSetResponse(callcount int, err error) {
	mvp.deleteAuxiliaryResourcesFuncResponse[callcount] = err
}

func (mvp *MockVmProvider) DeleteAuxiliaryResourcesGetRequest(callcount int) MockNamedNamespaceRequest {
	return mvp.deleteAuxiliaryResourcesFuncRequest[callcount]
}

// Mock tracker & implmenetation for `DeployVmService` function.
func (mvp *MockVmProvider) DeployVmService(ctx context.Context, req provider.VmDeploymentRequest) (string, error) {
	mvp.deployVmServiceFuncCallCount = mvp.deployVmServiceFuncCallCount + 1

	mvp.deployVmServiceFuncRequest[mvp.deployVmServiceFuncCallCount] = req
	if resp, ok := mvp.deployVmServiceFuncResponse[mvp.deployVmServiceFuncCallCount]; ok {
		return resp.Ip, resp.Error
	}
	return "", errors.New("no response set for function `DeployVmService`")
}

func (mvp *MockVmProvider) DeployVmServiceSetResponse(callcount int, ip string, err error) {
	mvp.deployVmServiceFuncResponse[callcount] = mockDeployVmServiceResponse{
		Ip:    ip,
		Error: err,
	}
}

func (mvp *MockVmProvider) DeployVmServiceGetRequest(callcount int) (string, error) {
	resp := mvp.deployVmServiceFuncResponse[callcount]
	return resp.Ip, resp.Error
}
