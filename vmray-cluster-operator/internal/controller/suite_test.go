// Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package controller_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"

	"github.com/vmware/ray-on-vcf/vmray-cluster-operator/test/builder"
)

var suite *builder.TestSuite = builder.NewTestSuiteWithoutManager()

func tests() {
	Describe("ray head node tests", rayHeadUnitTests)
	Describe("ray worker worker tests", rayWorkerUnitTests)
}

func TestRayControllers(t *testing.T) {
	suite.Register(t, "Suite to validate controller's reconcile logic", tests)
}

var _ = BeforeSuite(suite.BeforeSuite)

var _ = AfterSuite(suite.AfterSuite)
