import React, { useState } from 'react';

function Login({ login }) {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');

  return (
    <div>
      <h2>Login</h2>
      <input value={email} onChange={e => setEmail(e.target.value)} placeholder="email" />
      <input type="password" value={password} onChange={e => setPassword(e.target.value)} placeholder="password" />
      <button onClick={() => login(email, password)}>Login</button>
    </div>
  );
}

export default Login;