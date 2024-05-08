#!/usr/bin/env bash
set +x
# Delete all the namespaces which were created by CI pipeline
kubectl get ns -o go-template --template '{{range .items}}{{.metadata.name}} {{.metadata.creationTimestamp}}{{"\n"}}{{end}}' | \
    awk '$2 <= "'$(date -d '1 week ago' -Ins --utc | \
    sed 's/+0000/Z/')'" { print $1 }' | \
    { grep ci- || true; } | \
    xargs -I {} --no-run-if-empty python3 ci/scripts/vsphere-automation/client.py -c namespace -o delete -ns "{}"
