#!/bin/bash

# autoscale 네임스페이스 및 모든 리소스 삭제
kubectl delete namespace k8s-custom-metrics

echo "Namespace 'k8s-custom-metrics' and all its resources have been deleted."