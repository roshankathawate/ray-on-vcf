// Copyright (c) 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"

	"gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/test/builder"
)

var suite *builder.TestSuite = builder.NewTestSuite(true)

func webhookTests() {
	Describe("VMRayNodeConfig", rayNodeConfigUnitTests)
	Describe("VMRayCluster", vmRayClusterUnitTests)
}

func TestWebhook(t *testing.T) {
	suite.Register(t, "Validation webhook suite", webhookTests)
}

var _ = BeforeSuite(suite.BeforeSuite)

var _ = AfterSuite(suite.AfterSuite)
