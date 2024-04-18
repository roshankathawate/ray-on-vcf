// Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package builder

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/client-go/rest"

	vmrayv1alpha1 "gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	testutil "gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/test/builder/utils"
)

/*
This file serves as a util file for consumers to write test cases and helps to achieve the following:
1. Create a test environment.
2. Start the test environment.
3. Create a manager.
4. Initialize the manager.
5. Start the manager.
6. Start webhook server.
*/

type TestSuite struct {
	Context    context.Context
	envTest    envtest.Environment
	cancelFunc context.CancelFunc
	manager    manager.Manager
	isWebhook  bool
	config     *rest.Config
	k8sClient  client.Client
}

func NewTestSuite(
	isWebhook bool) *TestSuite {

	testSuite := &TestSuite{
		isWebhook: isWebhook,
	}

	testSuite.init()

	return testSuite
}

func (s *TestSuite) init() {

	rootDir := testutil.GetRootDirOrDie()

	s.envTest = envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join(rootDir, "config", "crd", "bases")},
		ErrorIfCRDPathMissing: false,
		WebhookInstallOptions: envtest.WebhookInstallOptions{
			Paths: []string{filepath.Join(rootDir, "config", "webhook")},
		},
	}
}

func (s *TestSuite) Register(t *testing.T, name string, runUnitTestsFn func()) {

	Describe("Unit tests", runUnitTestsFn)

	RunSpecs(t, name)
}

func (s *TestSuite) BeforeSuite() {
	RegisterFailHandler(Fail)

	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	s.Context, s.cancelFunc = context.WithCancel(context.TODO())

	var err error

	s.config, err = s.envTest.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(s.config).NotTo(BeNil())

	s.createManager()
	s.startManager()
	s.initializerManager()

	if s.isWebhook {
		s.startWebhookServer()
	}
}

func (s *TestSuite) AfterSuite() {
	s.cancelFunc()
	err := s.envTest.Stop()
	Expect(err).NotTo(HaveOccurred())
}

func (s *TestSuite) GetApiServer() string {
	address := s.envTest.ControlPlane.APIServer.SecureServing.ListenAddr.Address
	port := s.envTest.ControlPlane.APIServer.SecureServing.ListenAddr.Port

	return fmt.Sprintf("https://%s:%s/", address, port)
}

func (s *TestSuite) startWebhookServer() {
	// wait for the webhook server to get ready
	dialer := &net.Dialer{Timeout: time.Second}
	addrPort := fmt.Sprintf("%s:%d",
		s.envTest.WebhookInstallOptions.LocalServingHost, s.envTest.WebhookInstallOptions.LocalServingPort)
	Eventually(func() error {
		conn, err := tls.DialWithDialer(dialer, "tcp", addrPort, &tls.Config{InsecureSkipVerify: true})
		if err != nil {
			return err
		}
		err = conn.Close()
		if err != nil {
			return err
		}
		return nil
	}).Should(Succeed())
}

func (s *TestSuite) initializerManager() {
	// If it is a webhook test then setup the webhook with the manager
	if s.isWebhook {
		svr := s.manager.GetWebhookServer().(*webhook.DefaultServer)
		svr.Options.Host = s.envTest.WebhookInstallOptions.LocalServingHost
		svr.Options.Port = s.envTest.WebhookInstallOptions.LocalServingPort
		svr.Options.CertDir = s.envTest.WebhookInstallOptions.LocalServingCertDir

		var err error = (&vmrayv1alpha1.VMRayNodeConfig{}).SetupWebhookWithManager(s.manager)
		Expect(err).NotTo(HaveOccurred())
		err = (&vmrayv1alpha1.VMRayCluster{}).SetupWebhookWithManager(s.manager)
		Expect(err).NotTo(HaveOccurred())
	}
}

func (s *TestSuite) GetK8sClient() client.Client {
	// Used by the test cases to create k8s resources
	return s.k8sClient
}

func (s *TestSuite) createManager() {
	var err error

	scheme := runtime.NewScheme()
	err = vmrayv1alpha1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	err = admissionv1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	err = corev1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	err = rbacv1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	s.k8sClient, err = client.New(s.config, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(s.k8sClient).NotTo(BeNil())

	s.manager, err = ctrl.NewManager(s.config, ctrl.Options{
		Scheme:         scheme,
		LeaderElection: false,
		Metrics: metricsserver.Options{
			BindAddress: "0",
		},
	})

	Expect(err).NotTo(HaveOccurred())
}

func (s *TestSuite) startManager() {
	var err error

	go func() {
		defer GinkgoRecover()
		err = s.manager.Start(s.Context)
		Expect(err).NotTo(HaveOccurred())
	}()
}
