const express = require('express');
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
app.listen(PORT, () => console.log('Product service running on port ' + PORT));