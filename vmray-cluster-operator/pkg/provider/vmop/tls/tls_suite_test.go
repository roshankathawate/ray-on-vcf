package tls_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"

	"gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/test/builder"
)

var suite = builder.NewTestSuite(true)

var _ = BeforeSuite(suite.BeforeSuite)

var _ = AfterSuite(suite.AfterSuite)

func TestUtils(t *testing.T) {
	suite.Register(t, "Unit testcases to validate TLS", tlsTests)
}
