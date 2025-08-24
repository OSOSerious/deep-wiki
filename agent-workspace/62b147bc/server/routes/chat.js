const express = require('express');
const multer = require('multer');
const { pool } = require('../utils/db');
const { v4: uuid } = require('uuid');
const path = require('path');

const router = express.Router();
const upload = multer({ dest: 'uploads/', limits: { fileSize: 5 * 1024 * 1024 } });

router.get('/rooms', async (req, res) => {
  try {
    const { rows } = await pool.query(`
      SELECT r.id, r.name, u.username as created_by
      FROM rooms r
      JOIN users u ON r.created_by = u.id
    `);
    res.json(rows);
  } catch (e) {
    res.status(500).json({ error: e.message });
  }
});

router.post('/rooms', async (req, res) => {
  try {
    const { name } = req.body;
    const id = uuid();
    await pool.query(
      'INSERT INTO rooms(id,name,created_by) VALUES($1,$2,$3)',
      [id, name, req.user.id]
    );
    await pool.query(
      'INSERT INTO participants(user_id,room_id) VALUES($1,$2)',
      [req.user.id, id]
    );
    res.status(201).json({ id, name });
  } catch (e) {
    res.status(500).json({ error: e.message });
  }
});

router.get('/rooms/:roomId/messages', async (req, res) => {
  try {
    const { roomId } = req.params;
    const { rows } = await pool.query(`
      SELECT m.id, m.body, m.file_url, m.created_at,
             u.username as sender
      FROM messages m
      JOIN users u ON m.sender_id = u.id
      WHERE m.room_id=$1
      ORDER BY m.created_at ASC
    `, [roomId]);
    res.json(rows);
  } catch (e) {
    res.status(500).json({ error: e.message });
  }
});

router.post('/rooms/:roomId/messages', upload.single('file'), async (req, res) => {
  try {
    const { roomId } = req.params;
    const { body } = req.body;
    const fileUrl = req.file ? `/uploads/${req.file.filename}` : null;
    const id = uuid();
    await pool.query(
      'INSERT INTO messages(id,room_id,sender_id,body,file_url) VALUES($1,$2,$3,$4,$5)',
      [id, roomId, req.user.id, body, fileUrl]
    );
    res.status(201).json({ id, body, fileUrl });
  } catch (e) {
    res.status(500).json({ error: e.message });
  }
});

module.exports = router;