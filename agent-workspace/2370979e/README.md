# Microservices E-Commerce Platform

A fully-featured e-commerce system built with microservices, Kubernetes, Istio, and monitoring.

## Architecture

- **Product Catalog** (Go + Gin + PostgreSQL)
- **Order Service** (Python + FastAPI + PostgreSQL)
- **Payment Service** (Node.js + Express + PostgreSQL)
- **User Service** (Java 21 + Spring Boot + PostgreSQL)
- **API Gateway** (Node.js + Express)
- **Istio Service Mesh** (mTLS, traffic management)
- **Prometheus + Grafana** (metrics & dashboards)

## Quick Start

### Local (Docker Compose)

```bash
docker-compose up --build
```

Gateway available at http://localhost:3000

### Kubernetes (Skaffold)

Prerequisites:
- Kubernetes cluster
- Istio installed (`istioctl install`)
- Skaffold & kubectl

```bash
skaffold run
```

Access via Istio Ingress Gateway:

```bash
kubectl -n istio-system get svc istio-ingressgateway
# export INGRESS_IP=...
curl http://$INGRESS_IP/products
```

## Endpoints

| Service        | Endpoint (via gateway) | Description         |
|----------------|--------------------------|---------------------|
| Product Catalog| GET /products            | List products       |
|                | POST /products           | Create product      |
| Order Service  | POST /orders             | Create order        |
|                | GET /orders/{id}         | Get order           |
| Payment Service| POST /payments           | Process payment     |
| User Service   | POST /users              | Create user         |
|                | GET /users               | List users          |

## Monitoring

- Prometheus: http://localhost:9090 (port-forward)
- Grafana: http://localhost:3000 (admin/admin)

## Development

Each service has its own README under `services/<service>/README.md`.

## Testing

Each service includes unit tests:

```bash
cd services/product-catalog && go test ./...
cd services/order && pytest
cd services/payment && npm test
cd services/user && ./gradlew test
```

## Security

- Istio mTLS between services
- NetworkPolicies can be added
- Secrets managed via SealedSecrets or Vault

## CI/CD

GitHub Actions workflows are provided under `.github/workflows/` (build, test, push, deploy).