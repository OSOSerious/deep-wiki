const request = require('supertest');
const app = require('../app');

describe('GET /api/chat/rooms', () => {
  it('should fetch rooms', async () => {
    const res = await request(app)
      .get('/api/chat/rooms')
      .set('Authorization', 'Bearer validtoken');
    expect(res.statusCode).toBe(200);
    expect(Array.isArray(res.body)).toBe(true);
  });
});

describe('POST /api/chat/rooms', () => {
  it('should create a room', async () => {
    const res = await request(app)
      .post('/api/chat/rooms')
      .set('Authorization', 'Bearer validtoken')
      .send({ name: 'test room' });
    expect(res.statusCode).toBe(201);
    expect(res.body).toHaveProperty('id');
  });
});