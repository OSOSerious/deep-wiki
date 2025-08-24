const { pool } = require('../utils/db');
const { promisify } = require('util');
const redis = require('redis');
const client = redis.createClient({ host: config.REDIS_HOST, port: config.REDIS_PORT });
const getAsync = promisify(client.get).bind(client);
const setAsync = promisify(client.set).bind(client);

async function handleConnection(socket) io {
  const userId = socket.user.id;
  await setAsync(`online:${userId}`, Date.now());
  socket.join(userId);
  socket.broadcast.emit('user online', { userId });

  socket.on('join room', async ({ roomId }) => {
    socket.join(roomId);
    const room = await pool.query('SELECT * FROM rooms WHERE id=$1', [roomId]);
    socket.to(roomId).emit('user joined', { username: socket.user.username });
  });

  socket.on('typing', ({ roomId, isTyping }) => {
    socket.to(roomId).emit('typing', { userId, username: socket.user.username, isTyping });
  });

  socket.on('chat message', async ({ roomId, body, fileUrl }) => {
    const messageId = uuid();
    await pool.query(
      'INSERT INTO messages(id,room_id,sender_id,body,file_url) VALUES($1,$2,$3,$4,$5)',
      [messageId, roomId, userId, body, fileUrl]
    );
    const message = { id: messageId, body, fileUrl, sender: socket.user.username };
    io.to(roomId).emit('chat message', message);
  });

  socket.on('reaction', async ({ messageId, emoji }) => {
    await pool.query(
      `INSERT INTO reactions(message_id,user_id,emoji) VALUES($1,$2,$3)
       ON CONFLICT(message_id,user_id,emoji) DO NOTHING`,
      [messageId, userId, emoji]
    );
    io.to(messageId).emit('reaction added', { messageId, emoji, userId });
  });

  socket.on('disconnect', async () => {
    await client.del(`online:${userId}`);
    socket.broadcast.emit('user offline', { userId });
  });
}

module.exports = { handleConnection };