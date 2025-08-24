import express from 'express';
import pg from 'pg';
import promClient from 'prom-client';
const { Pool } = pg;

const pool = new Pool({
  connectionString: process.env.DATABASE_URL || 'postgres://payment_user:payment_pass@postgres:5432/payment_service'
});

const app = express();
app.use(express.json());

promClient.collectDefaultMetrics();
app.get('/metrics', (_, res) => res.end(promClient.register.metrics()));

app.get('/healthz', (_, res) => res.send('ok'));

app.post('/payments', async (req, res) => {
  const { order_id, amount, currency } = req.body;
  const client = await pool.connect();
  try {
    const { rows } = await client.query(
      'INSERT INTO payments(order_id, amount, currency, status) VALUES($1,$2,$3,$4) RETURNING *',
      [order_id, amount, currency, 'COMPLETED']
    );
    res.status(201).json(rows[0]);
  } catch (e) {
    res.status(500).json({ error: e.message });
  } finally {
    client.release();
  }
});

app.listen(8080, () => console.log('Payment service on :8080'));