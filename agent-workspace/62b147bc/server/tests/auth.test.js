const request = require('supertest');
const app = require('../app');

describe('POST /api/auth/register', () => {
  it('should register a new user', async () => {
    const res = await request(app)
      .post('/api/auth/register')
      .send({ username: 'test', email: 'test@test.com', password: 'password' });
    expect(res.statusCode).toBe(201);
    expect(res.body).toHaveProperty('id');
  });
});

describe('POST /api/auth/login', () => {
  it('should login a user', async () => {
    const res = await request(app)
      .post('/api/auth/login')
      .send({ email: 'test@test.com', password: 'password' });
    expect(res.statusCode).toBe(200);
    expect(res.body).toHaveProperty('token');
  });
});