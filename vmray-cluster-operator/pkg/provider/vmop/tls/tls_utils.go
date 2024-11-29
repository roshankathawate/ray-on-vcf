package tls

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	TLSSecretSuffix    = "-tls"
	RootCaSecretSuffix = "-root-cert"
	rootCaCertKey      = "ca.cert"
	rootCaKeyKey       = "ca.key"
	error_tmpl_ca_cert = "failure to read ca certificate: secret %s:%s doesn't contain `%s` key"
	error_tmpl_ca_key  = "failure to read ca key: secret %s:%s doesn't contain `%s` key"
)

func CreateVMRayClusterRootSecret(ctx context.Context, kubeclient client.Client,
	namespace, clusterName string) error {
	bitSize := 4096
	secretName := clusterName + RootCaSecretSuffix

	secretObjectkey := client.ObjectKey{
		Namespace: namespace,
		Name:      secretName,
	}

	// Check if secret exists.
	var validSecret corev1.Secret
	if err := kubeclient.Get(ctx, secretObjectkey, &validSecret); err == nil {
		return nil
	} else if client.IgnoreNotFound(err) != nil {
		return err
	}
	// https://shaneutt.com/blog/golang-ca-and-signed-cert-go/
	// set up our CA certificate
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		Subject: pkix.Name{
			Organization:  []string{"Broadcom, INC."},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"San Jose"},
			StreetAddress: []string{"Hillview Avenue"},
			PostalCode:    []string{"94304"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	// create our private and public key
	ca_key, err := rsa.GenerateKey(rand.Reader, bitSize)
	if err != nil {
		return err
	}

	// create the CA certificate
	ca_cert, err := x509.CreateCertificate(rand.Reader, ca, ca, &ca_key.PublicKey, ca_key)
	if err != nil {
		return err
	}

	// pem encode certificate
	caPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: ca_cert,
		},
	)

	// pem encode private key
	keyPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(ca_key),
		},
	)

	// Store new tls key and tls cert in secret
	return kubeclient.Create(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			rootCaCertKey: caPEM,
			rootCaKeyKey:  keyPEM,
		},
	})
}

func ReadCaCrtAndCaKeyFromSecret(ctx context.Context,
	kubeclient client.Client, namespace, name string) (string, string, error) {

	secretObjectkey := client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}
	// Check if secret exists.
	secret := &corev1.Secret{}
	if err := kubeclient.Get(ctx, secretObjectkey, secret); err != nil {
		return "", "", err
	}

	ca_crt, ok := secret.Data[rootCaCertKey]
	if !ok {
		return "", "", fmt.Errorf(error_tmpl_ca_cert, namespace, name, rootCaCertKey)
	}

	ca_key, ok := secret.Data[rootCaKeyKey]
	if !ok {
		return "", "", fmt.Errorf(error_tmpl_ca_cert, namespace, name, rootCaKeyKey)
	}
	return string(ca_key), string(ca_crt), nil
}

func GetRayTLSConfigString() string {
	TLSConfigString := `#!/bin/sh

## Create tls.key
openssl genrsa -out /home/ray/tls.key 2048

IP_3=""
if [ "${RAY_VMSERVICE_IP}" ]; then
IP_3="IP.3 = $RAY_VMSERVICE_IP"
fi

## Write CSR Config
cat > /home/ray/csr.conf <<EOF
	[ req ]
	default_bits = 2048
	prompt = no
	default_md = sha256
	req_extensions = req_ext
	distinguished_name = dn

	[ dn ]
	C = US
	ST = California
	L = San Fransisco
	O = ray
	OU = ray
	CN = *.ray.io

	[ req_ext ]
	subjectAltName = @alt_names

	[ alt_names ]
	DNS.1 = localhost
	IP.1 = 127.0.0.1
	IP.2 = $(hostname -I | awk '{print $1}')
	$IP_3
EOF

## Create CSR using tls.key
openssl req -new -key /home/ray/tls.key -out /home/ray/ca.csr -config /home/ray/csr.conf

## Write cert config
cat > /home/ray/cert.conf <<EOF
	authorityKeyIdentifier=keyid,issuer
	basicConstraints=CA:FALSE
	keyUsage = digitalSignature, nonRepudiation, keyEncipherment, dataEncipherment
	subjectAltName = @alt_names

	[alt_names]
	DNS.1 = localhost
	IP.1 = 127.0.0.1
	IP.2 = $(hostname -I | awk '{print $1}')
	$IP_3
EOF

## Generate tls.cert
openssl x509 -req \
	-in /home/ray/ca.csr \
	-CA /home/ray/ca.crt -CAkey /home/ray/ca.key \
	-CAcreateserial -out /home/ray/tls.crt \
	-days 365 \
	-sha256 -extfile /home/ray/cert.conf`
	return TLSConfigString
}
