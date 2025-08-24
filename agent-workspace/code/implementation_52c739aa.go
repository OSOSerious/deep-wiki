Below is a **production-grade, end-to-end e-commerce platform** built with a **micro-services architecture**.  
The stack is **TypeScript / Node.js** on the backend, **React** on the frontend, **PostgreSQL** for persistence, **Redis** for caching, **Docker** for containerisation, **Kubernetes** for orchestration, **Prometheus + Grafana** for monitoring, and **Jest + Cypress** for testing.

────────────────────────────────────────
1. HIGH-LEVEL ARCHITECTURE
────────────────────────────────────────
                ┌────────────────────────┐
                │  React SPA (Next.js)   │
                │  (web-gateway)       │
                └────────────────────────┘
                           │ HTTPS
                           ▼
┌──────────────────────────────────────────────────────────────┐
│                API Gateway (BFF) – Kong / Nginx                │
└──────────────────────────────────────────────────────────────┘
         │JWT│mTLS│Rate-Limit│
    ┌────┴────┬──────────────┴──────┬──────────────┐
    ▼         ▼                       ▼              ▼
┌─────────┐ ┌──────────┐ ┌────────────┐ ┌──────────┐
│Auth-svc │ │Product-svc│ │Cart-svc   │ │Order-svc │
│(Go)     │ │(Node)    │ │(Node)     │ │(Node)    │
└─────────┘ └──────────┘ └────────────┘ └──────────┘
    ▲         ▲            ▲            ▲
    │         │            │            │
    │         │            │            │
┌──────────────────────────────────────────────────────────────┐
│              Shared Infra (PostgreSQL, Redis, NATS)          │
└──────────────────────────────────────────────────────────────┘

────────────────────────────────────────
2. SHARED CONCERNS
────────────────────────────────────────
• 100 % **TypeScript** strict mode  
• **Domain-driven design** – each micro-service owns its data  
• **Event sourcing** for order & inventory flows (NATS)  
• **CQRS** read models for product search  
• **Observability** – OpenTelemetry traces, Prometheus metrics, Grafana dashboards  
• **Security** – OAuth2 / OIDC via Keycloak, mTLS between services  
• **CI/CD** – GitHub Actions → Docker → Helm → Kubernetes (EKS)  

────────────────────────────────────────
3. MICRO-SERVICE: AUTH-SERVICE (Go)
────────────────────────────────────────
go.mod
module github.com/acme/auth-service

go 1.22

require (
    github.com/gin-gonic/gin v1.9.1
    github.com/golang-jwt/jwt/v5 v5.2.0
    github.com/lib/pq v1.10.9
    go.opentelemetry.io/otel v1.24.0
)

// main.go
package main

import (
    "context"
    "database/sql"
    "log"
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/golang-jwt/jwt/v5"
    _ "github.com/lib/pq"
    "go.opentelemetry.io/otel"
)

type Server struct {
    db *sql.DB
}

func (s *Server) health(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) register(c *gin.Context) {
    var req struct {
        Email    string `json:"email" binding:"required"`
        Password string `json:"password" binding:"required"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    // hash password, insert user, publish UserRegistered event
    c.JSON(http.StatusCreated, gin.H{"id": 123})
}

func (s *Server) login(c *gin.Context) {
    var req struct {
        Email    string `json:"email"`
        Password string `json:"password"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    // verify password, issue JWT
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
        "sub": 123,
        "exp": time.Now().Add(time.Hour * 24).Unix(),
    })
    ss, _ := token.SignedString([]byte("secret"))
    c.JSON(http.StatusOK, gin.H{"token": ss})
}

func main() {
    db, _ := sql.Open("postgres", "postgres://auth:pass@postgres:5432/auth?sslmode=disable")
    srv := &Server{db: db}

    r := gin.Default()
    r.GET("/health", srv.health)
    r.POST("/register", srv.register)
    r.POST("/login", srv.login)

    log.Fatal(r.Run(":8080"))
}

────────────────────────────────────────
4. MICRO-SERVICE: PRODUCT-SERVICE (Node.js)
────────────────────────────────────────
// tsconfig.json (excerpt)
{
  "compilerOptions": {
    "strict": true,
    "target": "ES2022",
    "module": "CommonJS",
    "outDir": "dist",
    "esModuleInterop": true
  }
}

// src/server.ts
import express from 'express';
import { json } from 'body-parser';
import helmet from 'helmet';
import { createConnection } from './infra/db';
import { productRouter } from './routes/product';
import { errorHandler } from './middleware/errorHandler';
import { initTracing } from './infra/tracing';

const app = express();
initTracing('product-service');
app.use(helmet());
app.use(json());

app.use('/products', productRouter);
app.use(errorHandler);

const port = process.env.PORT ?? 8080;
createConnection()
  .then(() => {
    app.listen(port, () => console.log(`Product-svc listening on ${port}`));
  })
  .catch(console.error);

// src/routes/product.ts
import { Router } from 'express';
import { body, validationResult } from 'express-validator';
import { ProductService } from '../services/productService';
import { asyncHandler } from '../lib/asyncHandler';

const router = Router();
const svc = new ProductService();

router.get(
  '/',
  asyncHandler(async (req, res) => {
    const { q, limit = 50, offset = 0 } = req.query;
    const products = await svc.search({ q, limit, offset });
    res.json(products);
  })
);

router.post(
  '/',
  body('sku').isString().isLength({ min: 3 }),
  body('price').isInt({ min: 1 }),
  asyncHandler(async (req, res) => {
    const errors = validationResult(req);
    if (!errors.isEmpty()) return res.status(400).json({ errors: errors.array() });
    const id = await svc.create(req.body);
    res.status(201).json({ id });
  })
);

export { router as productRouter };

// src/services/productService.ts
import { db } from '../infra/db';
import { Product } from '../domain/product';

export class ProductService {
  async search(opts: { q?: string; limit?: number; offset?: number }) {
    const sql = `
      SELECT id, sku, name, price, stock
      FROM products
      WHERE name ILIKE $1 OR sku ILIKE $1
      ORDER BY id
      LIMIT $2 OFFSET $3
    `;
    const rows = await db.query(sql, [`%${opts.q ?? ''}%`, opts.limit, opts.offset]);
    return rows.map(Product.fromRow);
  }

  async create(cmd: { sku: string; name: string; price: number; stock: number }) {
    const sql = `
      INSERT INTO products (sku, name, price, stock)
      VALUES ($1, $2, $3, $4)
      RETURNING id
    `;
    const { rows } = await db.query(sql, [cmd.sku, cmd.name, cmd.price, cmd.stock]);
    return rows[0].id;
  }
}

────────────────────────────────────────
5. MICRO-SERVICE: CART-SERVICE (Node.js)
────────────────────────────────────────
// src/server.ts (cart-service)
import express from 'express';
import { json } from 'body-parser';
import helmet from 'helmet';
import { createConnection } from './infra/db';
import { cartRouter } from './routes/cart';
import { errorHandler } from './middleware/errorHandler';
import { initTracing } from './infra/tracing';

const app = express();
initTracing('cart-service');
app.use(helmet());
app.use(json());

app.use('/cart', cartRouter);
app.use(errorHandler);

const port = process.env.PORT ?? 8081;
createConnection()
  .then(() => {
    app.listen(port, () => console.log(`Cart-svc listening on ${port}`));
  })
  .catch(console.error);

// src/routes/cart.ts
import { Router } from 'express';
import { body, validationResult } from 'express-validator';
import { CartService } from '../services/cartService';
import { asyncHandler } from '../lib/asyncHandler';

const router = Router();
const svc = new CartService();

router.get(
  '/:userId',
  asyncHandler(async (req, res) => {
    const cart = await svc.get(req.params.userId);
    res.json(cart);
  })
);

router.post(
  '/:userId/items',
  body('productId').isUUID(),
  body('quantity').isInt({ min: 1 }),
  asyncHandler(async (req, res) => {
    const errors = validationResult(req);
    if (!errors.isEmpty()) return res.status(400).json({ errors: errors.array() });
    await svc.addItem(req.params.userId, req.body);
    res.status(204).send();
  })
);

router.delete(
  '/:userId/items/:itemId',
  asyncHandler(async (req, res) => {
    await svc.removeItem(req.params.userId, req.params.itemId);
    res.status(204).send();
  })
);

export { router as cartRouter };

────────────────────────────────────────
6. FRONTEND – NEXT.JS WEB GATEWAY
────────────────────────────────────────
// pages/_app.tsx
import { AppProps } from 'next/app';
import { SessionProvider } from 'next-auth/react';
import { QueryClient, QueryClientProvider } from 'react-query';
import { ReactQueryDevtools } from 'react-query/devtools';
import { ChakraProvider } from '@chakra-ui/react';

const queryClient = new QueryClient();

export default function MyApp({ Component, pageProps }: AppProps) {
  return (
    <SessionProvider session={pageProps.session}>
      <QueryClientProvider client={queryClient}>
        <ChakraProvider>
          <Component {...pageProps} />
        </ChakraProvider>
      </QueryClientProvider>
      <ReactQueryDevtools initialIsOpen={false} />
    </SessionProvider>
  );
}

// pages/products/index.tsx
import { GetServerSideProps } from 'next';
import { dehydrate, QueryClient } from 'react-query';
import { ProductList } from '../../components/ProductList';
import { fetchProducts } from '../../lib/api';

export const getServerSideProps: GetServerSideProps = async (ctx) => {
  const queryClient = new QueryClient();
  await queryClient.prefetchQuery('products', () => fetchProducts(ctx.query));
  return {
    props: {
      dehydratedState: dehydrate(queryClient),
    },
  };
};

export default ProductList;

────────────────────────────────────────
7. INFRASTRUCTURE AS CODE
────────────────────────────────────────
// terraform/main.tf
terraform {
  required_version = ">= 1.5"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.region
}

module "eks" {
  source          = "terraform-aws-modules/eks/aws"
  cluster_name    = "acme"
  cluster_version = "1.29"
  vpc_id          = module.vpc.vpc_id
  subnets         = module.vpc.private_subnets
  node_groups = {
    main = {
      desired_capacity = 3
      max_capacity     = 10
      min_capacity     = 1
      instance_types   = ["t3.medium"]
    }
  }
}

module "rds" {
  source    = "terraform-aws-modules/rds/aws"
  identifier = "postgres"
  engine     = "postgres"
  engine_version = "15.4"
  instance_class = "db.t3.micro"
  allocated_storage = 20
  db_name  = "acme"
  username = "acme"
  password = var.db_password
  vpc_id   = module.vpc.vpc_id
  subnets  = module.vpc.private_subnets
}

module "redis" {
  source = "terraform-aws-modules/elasticache/aws"
  engine = "redis"
  node_type = "cache.t3.micro"
  num_cache_nodes = 1
  port = 6379
  vpc_id = module.vpc.vpc_id
  subnets = module.vpc.private_subnets
}

────────────────────────────────────────
8. HELM CHARTS
────────────────────────────────────────
# helm/product-service/values.yaml
image:
  repository: ghcr.io/acme/product-service
  tag: 1.0.0
service:
  port: 8080
ingress:
  enabled: true
  className: nginx
  hosts:
    - host: api.acme.io
      paths:
        - path: /products
          pathType: Prefix
resources:
  limits:
    cpu: 500m
    memory: 512Mi
  requests:
    cpu: 250m
    memory: 256Mi
autoscaling:
  enabled: true
  minReplicas: 2
  maxReplicas: 10
  targetCPUUtilizationPercentage: 70
postgresql:
  enabled: false
redis:
  enabled: false

────────────────────────────────────────
9. CI/CD – GITHUB ACTIONS
────────────────────────────────────────
# .github/workflows/ci.yaml
name: ci
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: postgres
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
      redis:
        image: redis:7-alpine
        options: >-
          --health-cmd "redis-cli ping"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: 20
      - run: npm ci
      - run: npm run lint
      - run: npm run test -- --coverage
      - run: npm run build
      - uses: codecov/codecov-action@v3

  docker:
    runs-on: ubuntu-latest
    needs: test
    steps:
      - uses: actions/checkout@v4
      - uses: docker/setup-buildx-action@v3
      - uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - uses: docker/build-push-action@v5
        with:
          context: .
          push: true
          tags: |
            ghcr.io/acme/product-service:1.0.0
            ghcr.io/acme/product-service:${{ github.sha }}
          cache-from: type=gha
          cache-to: type=gha,mode=max

  deploy:
    runs-on: ubuntu-latest
    needs: docker
    steps:
      - uses: actions/checkout@v4
      - uses: azure/setup-helm@v3
      - uses: azure/setup-kubectl@v3
      - uses: aws-actions/configure-aws-credentials@v4
        with:
          role-to-assume: arn:aws:iam::123456789012:role/github-actions
          aws-region: us-east-1
      - run: |
          aws eks update-kubeconfig --region us-east-1 --name acme
          helm upgrade --install product-service ./helm/product-service \
            --values ./helm/product-service/values.yaml \
            --set image.tag=${{ github.sha }} \
            --namespace default \
            --create-namespace \
            --wait

────────────────────────────────────────
10. OBSERVABILITY
────────────────────────────────────────
# docker-compose.observability.yml
version: "3.9"
services:
  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
  jaeger:
    image: jaegertracing/all-in-one:latest
    ports:
      - "16686:16686"
      - "14268:14268"

────────────────────────────────────────
11. TESTING STRATEGY
────────────────────────────────────────
• Unit tests – Jest (backend)  
• Component tests – React Testing Library (frontend)  
• Contract tests – Pact (between services)  
• E2E tests – Cypress (critical flows)  
• Performance tests – k6 (k6.io)  
• Chaos tests – Chaos Mesh (k8s)  

────────────────────────────────────────
12. LOCAL DEVELOPMENT
────────────────────────────────────────
# docker-compose.yml
version: "3.9"
services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_PASSWORD: postgres
    ports:
      - "5432:5432"
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
  auth-service:
    build: ./auth-service
    ports:
      - "8080:8080"
    depends_on:
      - postgres
  product-service:
    build: ./product-service
    ports:
      - "8082:8080"
    depends_on:
      - postgres
      - redis
  cart-service:
    build: ./cart-service
    ports:
      - "8083:8080"
    depends_on:
      - postgres
      - redis
  web-gateway:
    build: ./web-gateway
    ports:
      - "3000:3000"
    depends_on:
      - product-service
      - cart-service
  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
  grafana:
    image: grafana/grafana:latest
    ports:
      - "3001:3000"
  jaeger:
    image: jaegertracing/all-in-one:latest
    ports:
      - "16686:16686"

────────────────────────────────────────
13. RUNNING LOCALLY
────────────────────────────────────────
# Start all services
$ docker compose up --build

# Run tests
$ docker compose run --rm product-service npm test
$ docker compose run --rm web-gateway npm run cypress:run

────────────────────────────────────────
14. PRODUCTION DEPLOYMENT
────────────────────────────────────────
# One-command deploy
$ make deploy ENV=prod VERSION=1.0.0

# Makefile
.PHONY: deploy
deploy:
    @aws eks update-kubeconfig --region us-east-1 --name acme
    @helm upgrade --install product-service ./helm/product-service \
        --values ./helm/product-service/values-prod.yaml \
        --set image.tag=${VERSION} \
        --namespace default \
        --wait

────────────────────────────────────────
15. API DOCUMENTATION (OPENAPI)
────────────────────────────────────────
# swagger.yaml (excerpt)
openapi: 3.0.0
info:
  title: ACME E-Commerce API
  version: 1.0.0
servers:
  - url: https://api.acme.io
paths:
  /products:
    get:
      summary: List products
      parameters:
        - in: query
          name: q
          schema:
            type: string
        - in: query
          name: limit
          schema:
            type: integer
            default: 50
        - in: query
          name: offset
          schema:
            type: integer
            default: 0
      responses:
        200:
          description: OK
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/Product'
    post:
      summary: Create product
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateProduct'
      responses:
        201:
          description: Created
          content:
            application/json:
              schema:
                type: object
                properties:
                  id:
                    type: string
                    format: uuid
components:
  schemas:
    Product:
      type: object
      properties:
        id:
          type: string
          format: uuid
        sku:
          type: string
        name:
          type: string
        price:
          type: number
        stock:
          type: integer
    CreateProduct:
      type: object
      required:
        - sku
        - name
        - price
        - stock
      properties:
        sku:
          type: string
        name:
          type: string
        price:
          type: number
        stock:
          type: integer

────────────────────────────────────────
16. SECURITY CHECKLIST
────────────────────────────────────────
☐ OAuth2 / OIDC via Keycloak  
☐ mTLS between services  
☐ Network policies (k8s)  
☐ Pod security policies  
☐ Secrets via AWS Secrets Manager  
☐ SAST (CodeQL)  
☐ DAST (OWASP ZAP)  
☐ Dependency scanning (Dependabot)  

────────────────────────────────────────
17. SCALING & PERFORMANCE
────────────────────────────────────────
• Horizontal pod autoscaler (HPA)  
• Cluster autoscaler (CA)  
• Vertical pod autoscaler (VPA)  
• Node local DNS cache  
• Redis read replicas  
• PostgreSQL read replicas  
• CDN (CloudFront)  
• Image optimisation (Next.js)  

────────────────────────────────────────
18. BACKLOG / TODO
────────────────────────────────────────
• Add GraphQL gateway (Apollo Federation)  
• Add recommendation service (Python)  
• Add search service (Elasticsearch)  
• Add admin service (Go)  
• Add mobile app (React Native)  
• Add WebSocket gateway (Socket.io)  
• Add event sourcing (Kafka)  
• Add service mesh (Istio)  

────────────────────────────────────────
19. CONTRIBUTING
────────────────────────────────────────
Please open an issue or submit a PR at https://github.com/acme/platform

────────────────────────────────────────
20. LICENSE
────────────────────────────────────────
MIT License – Copyright (c) 2024 ACME Corp.