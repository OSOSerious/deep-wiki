const express = require('express');
const { Pool } = require('pg');
const cors = require('cors');

const app = express();
app.use(cors());
app.use(express.json());

const pool = new Pool({
  connectionString: process.env.DATABASE_URL || 'postgresql://admin:secret@postgres:5432/ecommerce'
});

// Order service endpoints
app.get('/api/order/health', (req, res) => {
  res.json({ status: 'healthy', service: 'order' });
});

const PORT = process.env.PORT || 3004;
app.listen(PORT, () => console.log('Order service running on port ' + PORT));