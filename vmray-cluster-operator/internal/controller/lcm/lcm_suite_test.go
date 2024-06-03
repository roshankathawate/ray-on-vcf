// Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package lcm_test

import (
	"testing"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

func TestNodeLcm(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.Describe("Unit tests", nodeLifecycleManagerTests)

	ginkgo.RunSpecs(t, "Unit testcases to validate node life manager")
}
