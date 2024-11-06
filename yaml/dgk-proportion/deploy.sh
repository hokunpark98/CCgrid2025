#!/bin/bash

# autoscale 네임스페이스 생성
kubectl create namespace k8s-custom-metrics

# ConfigMap 적용
kubectl apply -f ConfigMap.yaml -n k8s-custom-metrics

# ClusterRole 적용
kubectl apply -f ClusterRole.yaml -n k8s-custom-metrics

# Service 적용
kubectl apply -f Deployments.yaml -n k8s-custom-metrics

kubectl apply -f Service.yaml -n k8s-custom-metrics

echo "Deployment completed successfully."