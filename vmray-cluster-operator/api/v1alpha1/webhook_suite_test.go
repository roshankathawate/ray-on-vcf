// Copyright (c) 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1_test

import (
	"testing"

	"gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/test/builder"
)

var suite *builder.TestSuite

// TODO: Find if there's a better way to take actions before/after the suite is run.

func setupTest() func() {
	suite = builder.NewTestSuiteForWebhook("default.validating.vmraynodeconfig.vmray.vmware.broadcom.com")

	suite.BeforeSuite()

	return func() {
		suite.AfterSuite()
	}
}

func TestWebhook(t *testing.T) {
	defer setupTest()()

	suite.Register(t, "Validation webhook suite", unitTests)
}
