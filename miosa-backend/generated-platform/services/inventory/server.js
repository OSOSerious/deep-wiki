const express = require('express');
const { Pool } = require('pg');
const cors = require('cors');

const app = express();
app.use(cors());
app.use(express.json());

const pool = new Pool({
  connectionString: process.env.DATABASE_URL || 'postgresql://admin:secret@postgres:5432/ecommerce'
});

// Inventory service endpoints
app.get('/api/inventory/health', (req, res) => {
  res.json({ status: 'healthy', service: 'inventory' });
});

const PORT = process.env.PORT || 3007;
app.listen(PORT, () => console.log('Inventory service running on port ' + PORT));