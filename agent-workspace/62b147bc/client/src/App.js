import React, { useEffect, useState } from 'react';
import axios from 'axios';
import io from 'socket.io-client';
import Chat from './components/Chat';
import Login from './components/Login';
import Register from './components/Register';
import './App.css';

const socket = io('http://localhost:8080');

function App() {
  const [user, setUser] = useState(null);
  const [token, setToken] = useState(localStorage.getItem('token'));

  useEffect(() => {
    if (!token) return;
    axios.defaults.headers.common['Authorization'] = `Bearer ${token}`;
    socket.auth = { token };
    socket.connect();
  }, [token]);

  const login = async (email, password) => {
    try {
      const { data } = await axios.post('/api/auth/login', { email, password });
      localStorage.setItem('token', data.token);
      setToken(data.token);
      setUser(data.user);
    } catch (e) {
      alert(e.response.data.error);
    }
  };

  const register = async (username, email, password) => {
    try {
      const { data } = await axios.post('/api/auth/register', { username, email, password });
      localStorage.setItem('token', data.token);
      setToken(data.token);
      setUser(data.user);
    } catch (e) {
      alert(e.response.data.error);
    }
  };

  const logout = () => {
    localStorage.removeItem('token');
    setToken(null);
    setUser(null);
    socket.disconnect();
  };

  return (
    <div className="App">
      {!token ? (
        <>
          <Login login={login} />
          <Register register={register} />
        </>
      ) : (
        <Chat user={user} socket={socket} logout={logout} />
      )}
    </div>
  );
}

export default App;