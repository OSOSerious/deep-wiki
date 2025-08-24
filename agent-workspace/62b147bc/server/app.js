const express = require('express');
const http = require('http');
const socketIo = require('socket.io');
const helmet = require('helmet');
const cors = require('cors');
const config = require('./config/config');
const authRoutes = require('./routes/auth');
const chatRoutes = require('./routes/chat');
const { authenticateSocket } = require('./utils/auth');
const { handleConnection } = require('./utils/socket');

const app = express();
const server = http.createServer(app);
const io = socketIo(server, {
  cors: { origin: config.CORS_ORIGIN, credentials: true }
});

app.use(helmet());
app.use(cors({ origin: config.CORS_ORIGIN, credentials: true }));
app.use(express.json({ limit: '10mb' }));
app.use(express.urlencoded({ extended: true, limit: '10mb' }));
app.use('/uploads', express.static('uploads'));

app.use('/api/auth', authRoutes);
app.use('/api/chat', chatRoutes);

io.use(authenticateSocket);
io.on('connection', handleConnection);

const PORT = config.PORT || 8080;
server.listen(PORT, () => console.log(`Server listening on port ${PORT}`));

module.exports = { app, server };