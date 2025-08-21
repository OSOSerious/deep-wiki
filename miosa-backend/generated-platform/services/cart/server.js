const express = require('express');
const { Pool } = require('pg');
const cors = require('cors');

const app = express();
app.use(cors());
app.use(express.json());

const pool = new Pool({
  connectionString: process.env.DATABASE_URL || 'postgresql://admin:secret@postgres:5432/ecommerce'
});

// Cart service endpoints
app.get('/api/cart/health', (req, res) => {
  res.json({ status: 'healthy', service: 'cart' });
});

const PORT = process.env.PORT || 3006;
app.listen(PORT, () => console.log('Cart service running on port ' + PORT));