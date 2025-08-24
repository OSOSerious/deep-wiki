module.exports = {
  PORT: process.env.PORT || 8080,
  CORS_ORIGIN: process.env.CORS_ORIGIN || 'http://localhost:3000',
  DB_HOST: process.env.DB_HOST || 'localhost',
  DB_PORT: process.env.DB_PORT || 5432,
  DB_USER: process.env.DB_USER || 'chatapp',
  DB_PASSWORD: process.env.DB_PASSWORD || 'chatapp',
  DB_NAME: process.env.DB_NAME || 'chatapp',
  REDIS_HOST: process.env.REDIS_HOST || 'localhost',
  REDIS_PORT: process.env.REDIS_PORT || 6379,
  JWT_SECRET: process.env.JWT_SECRET || 'supersecretjwtkey',
  BCRYPT_ROUNDS: 12
};