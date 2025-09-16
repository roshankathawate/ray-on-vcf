// Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package cloudinit_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestClusterModules(t *testing.T) {

	// Register failure handler.
	RegisterFailHandler(Fail)

	// Register unit testcases.
	Describe("Cloud init templating unit testcases", templatingTests)

	// Run the tests.
	RunSpecs(t, "Bootstrap cloud config init creation Suite")
}
