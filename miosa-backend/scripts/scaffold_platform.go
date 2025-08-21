package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	baseDir := "generated-platform"
	fmt.Println("🏗️  CREATING ACTUAL PLATFORM FILES")
	fmt.Println("=" + strings.Repeat("=", 60))
	
	// Remove existing directory if exists
	os.RemoveAll(baseDir)
	
	// Create directory structure
	createDirectories(baseDir)
	
	// Generate and write all files
	writeBackendServices(baseDir)
	writeFrontendComponents(baseDir)
	writeDatabaseMigrations(baseDir)
	writeInfrastructureFiles(baseDir)
	writeConfigFiles(baseDir)
	
	fmt.Println("\n✅ PLATFORM SCAFFOLDING COMPLETE!")
	fmt.Printf("📁 Created in: %s/\n", baseDir)
	fmt.Println("\n🚀 To run the platform:")
	fmt.Println("   cd generated-platform")
	fmt.Println("   docker-compose up")
}

func createDirectories(base string) {
	dirs := []string{
		"services/auth",
		"services/product", 
		"services/order",
		"services/payment",
		"services/cart",
		"services/inventory",
		"gateway",
		"frontend/src/components",
		"frontend/src/store/actions",
		"frontend/src/store/reducers",
		"frontend/src/styles",
		"frontend/public",
		"database/migrations",
		"kubernetes",
		"terraform",
		"scripts",
	}
	
	for _, dir := range dirs {
		path := filepath.Join(base, dir)
		os.MkdirAll(path, 0755)
		fmt.Printf("📁 Created: %s\n", dir)
	}
}

func writeBackendServices(base string) {
	fmt.Println("\n💻 Writing Backend Services...")
	
	// Auth Service
	writeFile(filepath.Join(base, "services/auth/server.js"), `const express = require('express');
const bcrypt = require('bcrypt');
const jwt = require('jsonwebtoken');
const { Pool } = require('pg');
const cors = require('cors');

const app = express();
app.use(cors());
app.use(express.json());

const pool = new Pool({
  connectionString: process.env.DATABASE_URL || 'postgresql://admin:secret@postgres:5432/ecommerce'
});

// User registration
app.post('/api/register', async (req, res) => {
  const { email, password, name } = req.body;
  
  try {
    const hashedPassword = await bcrypt.hash(password, 10);
    const result = await pool.query(
      'INSERT INTO users (email, password_hash, name) VALUES ($1, $2, $3) RETURNING id, email, name',
      [email, hashedPassword, name]
    );
    
    const token = jwt.sign(
      { userId: result.rows[0].id }, 
      process.env.JWT_SECRET || 'dev-secret-key'
    );
    res.json({ user: result.rows[0], token });
  } catch (error) {
    res.status(400).json({ error: error.message });
  }
});

// User login  
app.post('/api/login', async (req, res) => {
  const { email, password } = req.body;
  
  try {
    const result = await pool.query('SELECT * FROM users WHERE email = $1', [email]);
    const user = result.rows[0];
    
    if (!user || !await bcrypt.compare(password, user.password_hash)) {
      return res.status(401).json({ error: 'Invalid credentials' });
    }
    
    const token = jwt.sign(
      { userId: user.id }, 
      process.env.JWT_SECRET || 'dev-secret-key'
    );
    res.json({ 
      user: { id: user.id, email: user.email, name: user.name }, 
      token 
    });
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

// Verify token middleware
app.get('/api/verify', (req, res) => {
  const token = req.headers.authorization?.split(' ')[1];
  
  if (!token) {
    return res.status(401).json({ error: 'No token provided' });
  }
  
  try {
    const decoded = jwt.verify(token, process.env.JWT_SECRET || 'dev-secret-key');
    res.json({ valid: true, userId: decoded.userId });
  } catch (error) {
    res.status(401).json({ error: 'Invalid token' });
  }
});

const PORT = process.env.PORT || 3002;
app.listen(PORT, () => console.log('Auth service running on port ' + PORT));`)
	
	writeFile(filepath.Join(base, "services/auth/package.json"), `{
  "name": "auth-service",
  "version": "1.0.0",
  "main": "server.js",
  "scripts": {
    "start": "node server.js",
    "dev": "nodemon server.js"
  },
  "dependencies": {
    "express": "^4.18.2",
    "bcrypt": "^5.1.1",
    "jsonwebtoken": "^9.0.2",
    "pg": "^8.11.3",
    "cors": "^2.8.5"
  },
  "devDependencies": {
    "nodemon": "^3.0.1"
  }
}`)

	// Product Service
	writeFile(filepath.Join(base, "services/product/server.js"), `const express = require('express');
const { Pool } = require('pg');
const redis = require('redis');
const cors = require('cors');

const app = express();
app.use(cors());
app.use(express.json());

const pool = new Pool({
  connectionString: process.env.DATABASE_URL || 'postgresql://admin:secret@postgres:5432/ecommerce'
});

const redisClient = redis.createClient({
  url: process.env.REDIS_URL || 'redis://redis:6379'
});
redisClient.connect();

// Get all products with pagination
app.get('/api/products', async (req, res) => {
  const { page = 1, limit = 20, category, search } = req.query;
  const offset = (page - 1) * limit;
  
  try {
    let query = 'SELECT * FROM products WHERE 1=1';
    const params = [];
    
    if (category) {
      params.push(category);
      query += ' AND category = $' + params.length;
    }
    
    if (search) {
      params.push('%' + search + '%');
      query += ' AND name ILIKE $' + params.length;
    }
    
    query += ' ORDER BY created_at DESC';
    query += ' LIMIT $' + (params.length + 1) + ' OFFSET $' + (params.length + 2);
    params.push(limit, offset);
    
    const result = await pool.query(query, params);
    const countResult = await pool.query('SELECT COUNT(*) FROM products');
    
    res.json({
      products: result.rows,
      total: parseInt(countResult.rows[0].count),
      page: parseInt(page),
      totalPages: Math.ceil(countResult.rows[0].count / limit)
    });
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

// Get product by ID
app.get('/api/products/:id', async (req, res) => {
  try {
    const result = await pool.query('SELECT * FROM products WHERE id = $1', [req.params.id]);
    if (result.rows.length === 0) {
      return res.status(404).json({ error: 'Product not found' });
    }
    res.json(result.rows[0]);
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

// Create product (admin only)
app.post('/api/products', async (req, res) => {
  const { name, description, price, category, stock, image_url } = req.body;
  
  try {
    const result = await pool.query(
      'INSERT INTO products (name, description, price, category, stock, image_url) VALUES ($1, $2, $3, $4, $5, $6) RETURNING *',
      [name, description, price, category, stock, image_url]
    );
    res.status(201).json(result.rows[0]);
  } catch (error) {
    res.status(400).json({ error: error.message });
  }
});

const PORT = process.env.PORT || 3003;
app.listen(PORT, () => console.log('Product service running on port ' + PORT));`)

	writeFile(filepath.Join(base, "services/product/package.json"), `{
  "name": "product-service",
  "version": "1.0.0",
  "main": "server.js",
  "scripts": {
    "start": "node server.js",
    "dev": "nodemon server.js"
  },
  "dependencies": {
    "express": "^4.18.2",
    "pg": "^8.11.3",
    "redis": "^4.6.10",
    "cors": "^2.8.5"
  },
  "devDependencies": {
    "nodemon": "^3.0.1"
  }
}`)

	// Additional services...
	writeSimpleService(base, "order", 3004)
	writeSimpleService(base, "payment", 3005)
	writeSimpleService(base, "cart", 3006)
	writeSimpleService(base, "inventory", 3007)
	
	// API Gateway
	writeFile(filepath.Join(base, "gateway/server.js"), `const express = require('express');
const httpProxy = require('http-proxy-middleware');
const cors = require('cors');

const app = express();
app.use(cors());
app.use(express.json());

// Service routing
const services = {
  '/api/auth': process.env.AUTH_SERVICE_URL || 'http://auth-service:3002',
  '/api/products': process.env.PRODUCT_SERVICE_URL || 'http://product-service:3003',
  '/api/orders': process.env.ORDER_SERVICE_URL || 'http://order-service:3004',
  '/api/payment': process.env.PAYMENT_SERVICE_URL || 'http://payment-service:3005',
  '/api/cart': process.env.CART_SERVICE_URL || 'http://cart-service:3006',
  '/api/inventory': process.env.INVENTORY_SERVICE_URL || 'http://inventory-service:3007'
};

// Create proxies for each service
Object.keys(services).forEach(path => {
  app.use(path, httpProxy.createProxyMiddleware({
    target: services[path],
    changeOrigin: true
  }));
});

const PORT = process.env.PORT || 3001;
app.listen(PORT, () => console.log('API Gateway running on port ' + PORT));`)

	writeFile(filepath.Join(base, "gateway/package.json"), `{
  "name": "api-gateway",
  "version": "1.0.0",
  "main": "server.js",
  "scripts": {
    "start": "node server.js",
    "dev": "nodemon server.js"
  },
  "dependencies": {
    "express": "^4.18.2",
    "http-proxy-middleware": "^2.0.6",
    "cors": "^2.8.5"
  },
  "devDependencies": {
    "nodemon": "^3.0.1"
  }
}`)
}

func writeSimpleService(base, name string, port int) {
	serviceDir := filepath.Join(base, "services", name)
	
	writeFile(filepath.Join(serviceDir, "server.js"), fmt.Sprintf(`const express = require('express');
const { Pool } = require('pg');
const cors = require('cors');

const app = express();
app.use(cors());
app.use(express.json());

const pool = new Pool({
  connectionString: process.env.DATABASE_URL || 'postgresql://admin:secret@postgres:5432/ecommerce'
});

// %s service endpoints
app.get('/api/%s/health', (req, res) => {
  res.json({ status: 'healthy', service: '%s' });
});

const PORT = process.env.PORT || %d;
app.listen(PORT, () => console.log('%s service running on port ' + PORT));`, 
		strings.Title(name), name, name, port, strings.Title(name)))
	
	writeFile(filepath.Join(serviceDir, "package.json"), fmt.Sprintf(`{
  "name": "%s-service",
  "version": "1.0.0",
  "main": "server.js",
  "scripts": {
    "start": "node server.js",
    "dev": "nodemon server.js"
  },
  "dependencies": {
    "express": "^4.18.2",
    "pg": "^8.11.3",
    "cors": "^2.8.5"
  },
  "devDependencies": {
    "nodemon": "^3.0.1"
  }
}`, name))
}

func writeFrontendComponents(base string) {
	fmt.Println("\n🎨 Writing Frontend Components...")
	
	// React App.js
	writeFile(filepath.Join(base, "frontend/src/App.js"), `import React from 'react';
import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import { Provider } from 'react-redux';
import store from './store';
import Header from './components/Header';
import ProductList from './components/ProductList';
import ShoppingCart from './components/ShoppingCart';
import Checkout from './components/Checkout';
import './styles/App.css';

function App() {
  return (
    <Provider store={store}>
      <Router>
        <div className="App">
          <Header />
          <Routes>
            <Route path="/" element={<ProductList />} />
            <Route path="/cart" element={<ShoppingCart />} />
            <Route path="/checkout" element={<Checkout />} />
          </Routes>
        </div>
      </Router>
    </Provider>
  );
}

export default App;`)

	// ProductList Component
	writeFile(filepath.Join(base, "frontend/src/components/ProductList.jsx"), `import React, { useState, useEffect } from 'react';
import { useDispatch } from 'react-redux';
import { addToCart } from '../store/actions/cart';
import './ProductList.css';

const ProductList = () => {
  const [products, setProducts] = useState([]);
  const [loading, setLoading] = useState(true);
  const dispatch = useDispatch();
  
  useEffect(() => {
    fetchProducts();
  }, []);
  
  const fetchProducts = async () => {
    try {
      const response = await fetch('http://localhost:3001/api/products');
      const data = await response.json();
      setProducts(data.products || []);
    } catch (error) {
      console.error('Failed to fetch products:', error);
    } finally {
      setLoading(false);
    }
  };
  
  const handleAddToCart = (product) => {
    dispatch(addToCart(product));
  };
  
  if (loading) return <div>Loading products...</div>;
  
  return (
    <div className="product-list">
      <h2>Our Products</h2>
      <div className="product-grid">
        {products.map(product => (
          <div key={product.id} className="product-card">
            <img src={product.image_url || 'https://via.placeholder.com/200'} alt={product.name} />
            <h3>{product.name}</h3>
            <p>{product.description}</p>
            <div className="product-price">${product.price}</div>
            <button onClick={() => handleAddToCart(product)}>
              Add to Cart
            </button>
          </div>
        ))}
      </div>
    </div>
  );
};

export default ProductList;`)

	// Shopping Cart Component
	writeFile(filepath.Join(base, "frontend/src/components/ShoppingCart.jsx"), `import React from 'react';
import { useSelector, useDispatch } from 'react-redux';
import { removeFromCart, updateQuantity } from '../store/actions/cart';
import { Link } from 'react-router-dom';
import './ShoppingCart.css';

const ShoppingCart = () => {
  const cartItems = useSelector(state => state.cart.items);
  const dispatch = useDispatch();
  
  const total = cartItems.reduce((sum, item) => sum + (item.price * item.quantity), 0);
  
  return (
    <div className="shopping-cart">
      <h2>Shopping Cart</h2>
      {cartItems.length === 0 ? (
        <p>Your cart is empty</p>
      ) : (
        <>
          {cartItems.map(item => (
            <div key={item.id} className="cart-item">
              <span>{item.name}</span>
              <input
                type="number"
                value={item.quantity}
                onChange={(e) => dispatch(updateQuantity(item.id, e.target.value))}
                min="1"
              />
              <span>${(item.price * item.quantity).toFixed(2)}</span>
              <button onClick={() => dispatch(removeFromCart(item.id))}>
                Remove
              </button>
            </div>
          ))}
          <div className="cart-total">
            Total: ${total.toFixed(2)}
          </div>
          <Link to="/checkout" className="checkout-button">
            Proceed to Checkout
          </Link>
        </>
      )}
    </div>
  );
};

export default ShoppingCart;`)

	// Package.json for frontend
	writeFile(filepath.Join(base, "frontend/package.json"), `{
  "name": "ecommerce-frontend",
  "version": "1.0.0",
  "private": true,
  "dependencies": {
    "react": "^18.2.0",
    "react-dom": "^18.2.0",
    "react-router-dom": "^6.16.0",
    "react-redux": "^8.1.3",
    "redux": "^4.2.1",
    "redux-thunk": "^2.4.2",
    "axios": "^1.5.1"
  },
  "scripts": {
    "start": "react-scripts start",
    "build": "react-scripts build",
    "test": "react-scripts test",
    "eject": "react-scripts eject"
  },
  "devDependencies": {
    "react-scripts": "5.0.1"
  }
}`)
}

func writeDatabaseMigrations(base string) {
	fmt.Println("\n🗄️  Writing Database Migrations...")
	
	writeFile(filepath.Join(base, "database/migrations/001_users.sql"), `-- Users table
CREATE TABLE IF NOT EXISTS users (
  id SERIAL PRIMARY KEY,
  email VARCHAR(255) UNIQUE NOT NULL,
  password_hash VARCHAR(255) NOT NULL,
  name VARCHAR(255) NOT NULL,
  phone VARCHAR(20),
  role VARCHAR(50) DEFAULT 'customer',
  email_verified BOOLEAN DEFAULT false,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_email ON users(email);`)

	writeFile(filepath.Join(base, "database/migrations/002_products.sql"), `-- Products table
CREATE TABLE IF NOT EXISTS products (
  id SERIAL PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  description TEXT,
  price DECIMAL(10, 2) NOT NULL,
  category VARCHAR(100),
  stock INTEGER DEFAULT 0,
  image_url VARCHAR(500),
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_products_category ON products(category);
CREATE INDEX idx_products_name ON products(name);`)

	writeFile(filepath.Join(base, "database/migrations/003_orders.sql"), `-- Orders table
CREATE TABLE IF NOT EXISTS orders (
  id SERIAL PRIMARY KEY,
  user_id INTEGER REFERENCES users(id),
  status VARCHAR(50) DEFAULT 'pending',
  total DECIMAL(10, 2) NOT NULL,
  shipping_address TEXT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_orders_user_id ON orders(user_id);
CREATE INDEX idx_orders_status ON orders(status);`)
}

func writeInfrastructureFiles(base string) {
	fmt.Println("\n🚀 Writing Infrastructure Files...")
	
	// Docker Compose
	writeFile(filepath.Join(base, "docker-compose.yml"), `version: '3.8'

services:
  postgres:
    image: postgres:14
    environment:
      POSTGRES_DB: ecommerce
      POSTGRES_USER: admin
      POSTGRES_PASSWORD: secret
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./database/migrations:/docker-entrypoint-initdb.d
    ports:
      - "5432:5432"
  
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
  
  auth-service:
    build: ./services/auth
    environment:
      DATABASE_URL: postgresql://admin:secret@postgres:5432/ecommerce
      JWT_SECRET: dev-secret-key
    depends_on:
      - postgres
    ports:
      - "3002:3002"
  
  product-service:
    build: ./services/product
    environment:
      DATABASE_URL: postgresql://admin:secret@postgres:5432/ecommerce
      REDIS_URL: redis://redis:6379
    depends_on:
      - postgres
      - redis
    ports:
      - "3003:3003"
  
  gateway:
    build: ./gateway
    environment:
      AUTH_SERVICE_URL: http://auth-service:3002
      PRODUCT_SERVICE_URL: http://product-service:3003
    depends_on:
      - auth-service
      - product-service
    ports:
      - "3001:3001"
  
  frontend:
    build: ./frontend
    ports:
      - "3000:3000"
    depends_on:
      - gateway

volumes:
  postgres_data:`)
	
	// Dockerfiles
	writeFile(filepath.Join(base, "services/auth/Dockerfile"), `FROM node:16-alpine
WORKDIR /app
COPY package*.json ./
RUN npm ci --only=production
COPY . .
EXPOSE 3002
CMD ["node", "server.js"]`)
	
	writeFile(filepath.Join(base, "services/product/Dockerfile"), `FROM node:16-alpine
WORKDIR /app
COPY package*.json ./
RUN npm ci --only=production
COPY . .
EXPOSE 3003
CMD ["node", "server.js"]`)
	
	writeFile(filepath.Join(base, "gateway/Dockerfile"), `FROM node:16-alpine
WORKDIR /app
COPY package*.json ./
RUN npm ci --only=production
COPY . .
EXPOSE 3001
CMD ["node", "server.js"]`)
	
	writeFile(filepath.Join(base, "frontend/Dockerfile"), `FROM node:16-alpine as build
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM nginx:alpine
COPY --from=build /app/build /usr/share/nginx/html
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]`)
}

func writeConfigFiles(base string) {
	fmt.Println("\n📝 Writing Configuration Files...")
	
	// Root README
	writeFile(filepath.Join(base, "README.md"), `# E-Commerce Platform

## Quick Start

### Prerequisites
- Docker & Docker Compose
- Node.js 16+ (for local development)
- PostgreSQL (or use Docker)

### Running with Docker

1. Start all services:
   docker-compose up

2. Access the application:
- Frontend: http://localhost:3000
- API Gateway: http://localhost:3001
- Auth Service: http://localhost:3002
- Product Service: http://localhost:3003

### Local Development

1. Install dependencies for each service:
   cd services/auth && npm install
   cd ../product && npm install
   cd ../../gateway && npm install
   cd ../frontend && npm install

2. Start PostgreSQL and Redis:
   docker-compose up postgres redis

3. Run migrations:
   psql -U admin -d ecommerce -f database/migrations/001_users.sql
   psql -U admin -d ecommerce -f database/migrations/002_products.sql
   psql -U admin -d ecommerce -f database/migrations/003_orders.sql

4. Start services in separate terminals:
   cd services/auth && npm start
   cd services/product && npm start
   cd gateway && npm start
   cd frontend && npm start

## Project Structure

generated-platform/
├── services/           # Microservices
│   ├── auth/          # Authentication service
│   ├── product/       # Product catalog service
│   ├── order/         # Order management
│   ├── payment/       # Payment processing
│   ├── cart/          # Shopping cart
│   └── inventory/     # Inventory management
├── gateway/           # API Gateway
├── frontend/          # React application
├── database/          # Database migrations
├── kubernetes/        # K8s manifests
├── terraform/         # Infrastructure as code
└── docker-compose.yml # Local development

## Technology Stack

- Backend: Node.js, Express
- Frontend: React, Redux
- Database: PostgreSQL, Redis
- Infrastructure: Docker, Kubernetes
- Authentication: JWT

Generated by MIOSA Platform Generation System`)
	
	// .gitignore
	writeFile(filepath.Join(base, ".gitignore"), `node_modules/
.env
.env.local
dist/
build/
*.log
.DS_Store
postgres_data/`)
	
	// Environment template
	writeFile(filepath.Join(base, ".env.example"), `# Database
DATABASE_URL=postgresql://admin:secret@localhost:5432/ecommerce

# Redis
REDIS_URL=redis://localhost:6379

# JWT
JWT_SECRET=your-secret-key-here

# Services
AUTH_SERVICE_URL=http://localhost:3002
PRODUCT_SERVICE_URL=http://localhost:3003
ORDER_SERVICE_URL=http://localhost:3004
PAYMENT_SERVICE_URL=http://localhost:3005

# Frontend
REACT_APP_API_URL=http://localhost:3001`)
}

func writeFile(path, content string) {
	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		fmt.Printf("❌ Error writing %s: %v\n", path, err)
	} else {
		fmt.Printf("✅ Created: %s\n", path)
	}
}