// Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"

	"github.com/vmware/ray-on-vcf/vmray-cluster-operator/test/builder"
)

var suite *builder.TestSuite = builder.NewTestSuite(true)

func webhookTests() {
	Describe("VMRayCluster", vmRayClusterUnitTests)
}

func TestWebhook(t *testing.T) {
	suite.Register(t, "Validation webhook suite", webhookTests)
}

var _ = BeforeSuite(suite.BeforeSuite)

var _ = AfterSuite(suite.AfterSuite)
