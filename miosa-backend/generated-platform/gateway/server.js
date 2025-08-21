const express = require('express');
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
app.listen(PORT, () => console.log('API Gateway running on port ' + PORT));