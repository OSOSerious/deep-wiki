const express = require('express');
const { Pool } = require('pg');
const cors = require('cors');

const app = express();
app.use(cors());
app.use(express.json());

const pool = new Pool({
  connectionString: process.env.DATABASE_URL || 'postgresql://admin:secret@postgres:5432/ecommerce'
});

// Payment service endpoints
app.get('/api/payment/health', (req, res) => {
  res.json({ status: 'healthy', service: 'payment' });
});

const PORT = process.env.PORT || 3005;
app.listen(PORT, () => console.log('Payment service running on port ' + PORT));