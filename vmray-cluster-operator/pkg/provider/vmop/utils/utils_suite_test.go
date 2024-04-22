// Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package utils_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"

	"gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/test/builder"
)

var suite = builder.NewTestSuite(true)

var _ = BeforeSuite(suite.BeforeSuite)

var _ = AfterSuite(suite.AfterSuite)

func TestUtils(t *testing.T) {
	suite.Register(t, "Unit testcases to validate cloud config secret creation", cloudInitSecretCreationTests)
}
