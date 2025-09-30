FROM dockerhub.packages.vcfd.broadcom.net/library/ubuntu:22.04

ARG GO_VERSION=1.21.12
ARG TARGETOS=linux
ARG TARGETARCH=amd64

# Install Essentials
RUN <<EOF
set -e
apt-get update -y
apt-get install -y build-essential
apt-get install python3-virtualenv -y
apt-get -y install curl
apt-get install git -y
EOF


# Install Go
RUN <<EOF
set -e
curl -sLo /tmp/go.tgz "https://dl.google.com/go/go${GO_VERSION}.$TARGETOS-$TARGETARCH.tar.gz"
tar -C /usr/local -xzf /tmp/go.tgz
rm -rf /tmp/go.tgz
mkdir -p /opt/gopath
EOF
ENV GOPATH "/opt/gopath"
ENV PATH "$GOPATH/bin:/usr/local/go/bin:${PATH}"
RUN go version


# Install infra tool: docker and its CLI plugins
COPY --from=dockerhub.packages.vcfd.broadcom.net/library/docker:26.1.0 /usr/local/bin/docker /usr/local/bin/
COPY --from=dockerhub.packages.vcfd.broadcom.net/docker/buildx-bin:0.14.0 /buildx /usr/libexec/docker/cli-plugins/docker-buildx
RUN <<EOF
docker version
docker buildx version

mkdir -p /usr/local/lib/docker/cli-plugins
curl \
    -L "https://github.com/docker/compose/releases/download/v2.27.0/docker-compose-$TARGETOS-$(uname -m)" \
    -o /usr/local/lib/docker/cli-plugins/docker-compose
chmod 0755 /usr/local/lib/docker/cli-plugins/docker-compose
docker compose version
EOF

# Install infra tools: kubectl binary
RUN <<EOF
set -eu
# kubectl
curl \
    -L "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/$TARGETOS/$TARGETARCH/kubectl" \
    -o /usr/local/bin/kubectl
chmod 0755 /usr/local/bin/kubectl
kubectl version --client
EOF

# Install Carvel Tools
# https://docs.vmware.com/en/VMware-Tanzu-Kubernetes-Grid/2.5/using-tkg/workload-carvel-tools.html
RUN <<EOF
set -eu
curl \
    -sLo /tmp/tkg-carvel-tools-linux-amd64.tar.gz \
    "https://artifactory.vcfd.broadcom.net/artifactory/paih-generic-local/carvel-tools/tkg-carvel-tools-linux-amd64.tar.gz"
tar -C /tmp -xvf /tmp/tkg-carvel-tools-linux-amd64.tar.gz
gunzip /tmp/cli/*.2.gz
chmod -R ugo+x /tmp/cli/

mv /tmp/cli/ytt-linux-amd64-v0.45.0+vmware.2 /usr/local/bin/ytt
ytt --version

mv /tmp/cli/imgpkg-linux-amd64-v0.36.0+vmware.2 /usr/local/bin/imgpkg
imgpkg --version

mv /tmp/cli/kapp-linux-amd64-v0.55.0+vmware.2 /usr/local/bin/kapp
kapp --version

mv /tmp/cli/kbld-linux-amd64-v0.37.0+vmware.2 /usr/local/bin/kbld
kbld --version

rm -rf /tmp/cli/
rm -rf /tmp/tkg-carvel-tools-linux-amd64.tar.gz
EOF

# install yq
ARG YQ_VERSION=v4.44.3
ADD --link --chmod=0755 "https://github.com/mikefarah/yq/releases/download/${YQ_VERSION}/yq_linux_${TARGETARCH}" /usr/local/bin/yq

# Install kustomize
RUN <<EOF
curl -s "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"  | bash
chmod 0755 kustomize
mv kustomize /usr/local/bin/kustomize
EOF
