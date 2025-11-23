#!/bin/sh

set -e

minikube status >/dev/null 2>&1 || minikube start
kubectl get ns observability >/dev/null 2>&1 || kubectl create ns observability
kubectl get ns lexigo >/dev/null 2>&1 || kubectl create ns lexigo