import React, { useState } from 'react';

function Register({ register }) {
  const [username, setUsername] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');

  return (
    <div>
      <h2>Register</h2>
      <input value={username} onChange={e => setUsername(e.target.value)} placeholder="username" />
      <input value={email} onChange={e => setEmail(e.target.value)} placeholder="email" />
      <input type="password" value={password} onChange={e => setPassword(e.target.value)} placeholder="password" />
      <button onClick={() => register(username, email, password)}>Register</button>
    </div>
  );
}

export default Register;