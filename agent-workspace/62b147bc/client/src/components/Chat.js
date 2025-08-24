import React, { useEffect, useState, useRef } from 'react';
import axios from 'axios';
import EmojiPicker from './EmojiPicker';
import FileUpload from './FileUpload';

function Chat({ user, socket, logout }) }) {
  const [rooms, setRooms] = useState([]);
  const [room, setRoom] = useState(null);
  const [messages, setMessages] = useState([]);
  const [message, setMessage] = useState('');
  const [typing, setTyping] = useState([]);
  const [online, setOnline] = useState([]);
  const [file, setFile] = useState(null);
  const messagesEndRef = useRef(null);

  useEffect(() => {
    axios.get('/api/chat/rooms').then(({ data }) => setRooms(data));
  }, []);

  useEffect(() => {
    socket.on('user online', ({ userId }) => setOnline(o => [...o, userId]));
    socket.on('user offline', ({ userId }) => setOnline(o => o.filter(id => id !== userId)));
    socket.on('user joined', ({ username }) => alert(`${username} joined`));
    socket.on('chat message', msg => setMessages(m => [...m, msg]));
    socket.on('typing', ({ username, isTyping }) => {
      setTyping(t => isTyping ? [...t, username] : t.filter(n => n !== username));
    });
    socket.on('reaction added', ({ messageId, emoji }) => {
      setMessages(m => m.map(msg => msg.id === messageId ? { ...msg, reactions: [...msg.reactions, emoji] } : msg));
    });
    return () => socket.off();
  }, [socket]);

  const scrollToBottom = () => messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });

  useEffect(scrollToBottom, [messages]);

  const sendMessage = () => {
    if (!message.trim() && !file) return;
    socket.emit('chat message', { roomId: room.id, body: message, fileUrl: file });
    setMessage('');
    setFile(null);
  };

  const sendReaction = (messageId, emoji) => socket.emit('reaction', { messageId, emoji });

  const sendTyping = (isTyping) => socket.emit('typing', { roomId: room.id, isTyping });

  return (
    <div>
      <h2>Chat</h2>
      <button onClick={logout}>Logout</button>
      <select onChange={e => setRoom(rooms.find(r => r.id === e.target.value))}>
        <option>Select room</option>
        {rooms.map(r => <option key={r.id} value={r.id}>{r.name}</option>)}
      </select>
      <div>
        {messages.map(m => (
          <div key={m.id}>
            <p>{m.sender}: {m.body}</p>
            {m.fileUrl && <img src={m.fileUrl} alt="upload" />}
            <EmojiPicker onSelect={e => sendReaction(m.id, e.native)} />
            <span>{m.reactions.join(' ')}</span>
          </div>
        ))}
        <div ref={messagesEndRef} />
      </div>
      <textarea value={message} onChange={e => setMessage(e.target.value)} onKeyUp={e => sendTyping(true)} onBlur={e => sendTyping(false)} />
      <FileUpload onUpload={f => setFile(f)} />
      <button onClick={sendMessage}>Send</button>
    </div>
  );
}

export default Chat;