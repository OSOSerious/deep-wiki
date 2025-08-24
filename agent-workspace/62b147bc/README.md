# Real-Time Chat Application

A full-stack real-time chat application built with Node.js, React, PostgreSQL, Redis, and Socket.IO.

## Features
- Real-time messaging via WebSockets
- User authentication & JWT
- Message history
- Typing indicators
- Online presence
- File sharing
- Emoji reactions

## Quick Start

### Prerequisites
- Docker & Docker Compose
- Node.js 18+

### Run
```bash
docker-compose up --build
```

Then open http://localhost:3000

### Local Development
```bash
# Server
cd server
npm install
npm run dev

# Client
cd client
npm install
npm start
```

### Tests
```bash
cd server
npm test
```

### Database
PostgreSQL schema is in `server/schema.sql`.

### Environment Variables
Create `.env` in server/:
```
PORT=8080
DB_HOST=localhost
DB_USER=chatapp
DB_PASSWORD=chatapp
DB_NAME=chatapp
JWT_SECRET=supersecretjwtkey
```

Enjoy chatting!