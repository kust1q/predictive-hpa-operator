# Predictive Kubernetes Autoscaler
A smart Kubernetes controller that proactively scales application pods based on machine learning forecasts, utilizing historical metrics rather than reacting retroactively like standard HPA.

# Technologies
[![Go](https://img.shields.io/badge/-Go-464646?style=flat-square&logo=go)](https://go.dev/)
[![Python](https://img.shields.io/badge/-Python-464646?style=flat-square&logo=python)](https://www.python.org/)
[![Kubernetes](https://img.shields.io/badge/-Kubernetes-464646?style=flat-square&logo=kubernetes)](https://kubernetes.io/)
[![Prometheus](https://img.shields.io/badge/-Prometheus-464646?style=flat-square&logo=prometheus)](https://prometheus.io/)
[![gRPC](https://img.shields.io/badge/-gRPC-464646?style=flat-square&logo=grpc)](https://grpc.io/)
[![Docker](https://img.shields.io/badge/-Docker-464646?style=flat-square&logo=docker)](https://www.docker.com/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

## Tech Stack

### Languages & Frameworks
- **Programming Language**: Go 1.25 (Operator), Python 3.11 (Predictor)
- **Framework**: [Kubebuilder](https://book.kubebuilder.io/) — Scaffold for building Kubernetes APIs in Go
- **Machine Learning**: [Facebook Prophet](https://facebook.github.io/prophet/) — Forecasting procedure implemented in Python
- **gRPC**: Low-latency binary protocol for communication between the Go operator and Python ML service

### Monitoring & Analytics
- **Metrics**: [Prometheus](https://prometheus.io/) — Time-series database for fetching historical resource usage (e.g., CPU, Memory)

### Infrastructure & Orchestration
- **Containerization**: [Docker](https://www.docker.com/)
- **Orchestration**: [Kubernetes](https://kubernetes.io/)
- **Deployment**: Kustomize

## Project Structure

```
predictive-hpa-operator/
├── proto/                         # Shared protobuf contracts (gRPC)
│   └── predictor.proto            #   Predictor service contract
│
├── operator/                      # Go Kubernetes Operator
│   ├── api/v1/              #   CRD definitions and generated Go gRPC code
│   ├── cmd/app/                   #   Entry point
│   │   └── main.go
│   ├── internal/
│   │   ├── config/                #   Application configuration
│   │   └── controller/            #   Reconciliation logic, Prometheus & gRPC clients
│   ├── test/                      #   E2E Testing utilities
│   ├── Dockerfile
│   └── Makefile
│
├── predictor/                     # Python ML Microservice
│   ├── server.py                  #   gRPC server and Prophet integration
│   ├── test_server.py             #   Unit tests
│   ├── requirements.txt           #   Python dependencies
│   └── Dockerfile
│
├── k8s/                           # Kubernetes manifests (Kustomize)
│   ├── crd/                       #   Custom Resource Definitions
│   ├── operator/                  #   Operator Deployment
│   ├── predictor/                 #   Predictor Deployment and Service
│   ├── rbac/                      #   Role-Based Access Control
│   ├── prometheus/                #   Prometheus ServiceMonitor (optional)
│   └── samples/                   #   Sample CRDs for testing
│
├── docker-compose.yml             # Local development build
├── .dockerignore
└── .github/workflows/             # CI/CD pipelines
```

## How It Works (Architecture)

1. **Metrics Gathering (Go)**: The custom operator (`PredictiveHPA` CRD) runs in the cluster and periodically fetches historical metrics via Prometheus API based on a custom query.
2. **Data Transmission (gRPC)**: The controller packs the time-series data into a Protobuf message and sends it to the external ML analytics service via gRPC.
3. **Analysis and Prediction (Python + Prophet)**: A lightweight microservice receives the data, calculates the trend using Facebook Prophet, and returns a prediction for the required replicas in the future (e.g., +15 minutes).
4. **Proactive Scaling (K8s API)**: The Go controller receives the prediction, applies min/max constraints, and patches the target Deployment's `/scale` subresource proactively.

## Quick Start

### Requirements
- Go 1.25+
- Python 3.11+
- Docker
- A Kubernetes cluster (e.g., [Kind](https://kind.sigs.k8s.io/)) with Prometheus installed.

### Local Build & Test

To build the images locally:
```bash
docker-compose build
```

To run unit tests:
```bash
# Go Operator tests
cd operator && make setup-envtest && go test ./...

# Python Predictor tests
cd predictor && pip install -r requirements.txt pytest && pytest
```

### Deployment

Deploy to your Kubernetes cluster using Kustomize:

1. Build and push the images, or load them into your local Kind cluster.
2. Install the CRDs and deploy the operator/predictor:
```bash
cd operator && make install
cd .. && kubectl apply -k k8s/default
```
3. Apply a sample configuration:
```bash
kubectl apply -f k8s/samples/autoscaling_v1_predictivehpa.yaml
```

## License

This project is licensed under the [Apache 2.0 License](LICENSE).
