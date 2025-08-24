import express from 'express';
import { createProxyMiddleware } from 'http-proxy-middleware';
import helmet from 'helmet';
import cors from 'cors';

const app = express();
app.use(helmet());
app.use(cors());

const services = {
  '/products': 'http://product-catalog:8080',
  '/orders': 'http://order-service:8080',
  '/payments': 'http://payment-service:8080',
  '/users': 'http://user-service:8080'
};

Object.entries(services).forEach(([path, target]) => {
  app.use(path, createProxyMiddleware({ target, changeOrigin: true }));
});

app.get('/healthz', (_, res) => res.send('ok'));

const port = process.env.PORT || 3000;
app.listen(port, () => console.log(`API Gateway on :${port}`));