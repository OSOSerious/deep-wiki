<script lang="ts">
  import { onMount } from 'svelte';
  
  let message = '';
  let messages: Array<{role: string, content: string}> = [];
  let loading = false;
  let selectedModel = 'fast';
  let apiHealthy = false;
  
  onMount(async () => {
    // Check API health
    try {
      const response = await fetch('http://localhost:8080/health');
      if (response.ok) {
        apiHealthy = true;
        const data = await response.json();
        console.log('API Health:', data);
      }
    } catch (error) {
      console.error('API not reachable:', error);
    }
  });
  
  async function sendMessage() {
    if (!message.trim() || loading) return;
    
    const userMessage = message;
    messages = [...messages, { role: 'user', content: userMessage }];
    message = '';
    loading = true;
    
    try {
      let endpoint = '/api/chat';
      let payload: any = { message: userMessage };
      
      if (selectedModel === 'deep') {
        endpoint = '/api/analyze';
        payload = { 
          content: userMessage,
          type: 'general'
        };
      } else if (selectedModel === 'consultation') {
        endpoint = '/api/consultation';
        payload = {
          topic: userMessage,
          context: 'User query',
          phase: 'initial'
        };
      }
      
      const response = await fetch(`http://localhost:8080${endpoint}`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(payload),
      });
      
      const data = await response.json();
      
      if (data.success) {
        messages = [...messages, { 
          role: 'assistant', 
          content: data.data + `\n\n[Model: ${data.model}]`
        }];
      } else {
        messages = [...messages, { 
          role: 'error', 
          content: `Error: ${data.error}` 
        }];
      }
    } catch (error) {
      messages = [...messages, { 
        role: 'error', 
        content: `Failed to connect: ${error}` 
      }];
    } finally {
      loading = false;
    }
  }
  
  function handleKeypress(e: KeyboardEvent) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      sendMessage();
    }
  }
</script>

<div class="container">
  <header>
    <h1>🚀 MIOSA Chat</h1>
    <div class="status">
      {#if apiHealthy}
        <span class="badge success">✅ API Connected</span>
      {:else}
        <span class="badge error">❌ API Disconnected</span>
      {/if}
    </div>
  </header>
  
  <div class="model-selector">
    <label>
      <input type="radio" bind:group={selectedModel} value="fast" />
      Fast (Llama 3.1 8B)
    </label>
    <label>
      <input type="radio" bind:group={selectedModel} value="deep" />
      Deep Analysis (Kimi K2)
    </label>
    <label>
      <input type="radio" bind:group={selectedModel} value="consultation" />
      Consultation Mode
    </label>
  </div>
  
  <div class="chat-container">
    <div class="messages">
      {#if messages.length === 0}
        <div class="welcome">
          <h2>Welcome to MIOSA!</h2>
          <p>Select a model above and start chatting. Try:</p>
          <ul>
            <li><strong>Fast:</strong> Quick responses for general queries</li>
            <li><strong>Deep Analysis:</strong> Detailed analysis with Kimi K2</li>
            <li><strong>Consultation:</strong> Structured advice and guidance</li>
          </ul>
        </div>
      {/if}
      
      {#each messages as msg}
        <div class="message {msg.role}">
          <div class="message-role">{msg.role === 'user' ? '👤 You' : msg.role === 'assistant' ? '🤖 MIOSA' : '⚠️ System'}</div>
          <div class="message-content">{msg.content}</div>
        </div>
      {/each}
      
      {#if loading}
        <div class="message assistant">
          <div class="message-role">🤖 MIOSA</div>
          <div class="message-content typing">Thinking...</div>
        </div>
      {/if}
    </div>
    
    <div class="input-container">
      <textarea
        bind:value={message}
        on:keypress={handleKeypress}
        placeholder="Type your message here... (Press Enter to send)"
        disabled={loading || !apiHealthy}
        rows="3"
      ></textarea>
      <button 
        on:click={sendMessage} 
        disabled={loading || !message.trim() || !apiHealthy}
      >
        {loading ? 'Sending...' : 'Send'}
      </button>
    </div>
  </div>
</div>

<style>
  :global(body) {
    margin: 0;
    padding: 0;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    min-height: 100vh;
  }
  
  .container {
    max-width: 900px;
    margin: 0 auto;
    padding: 2rem;
    height: 100vh;
    display: flex;
    flex-direction: column;
  }
  
  header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1rem;
    color: white;
  }
  
  h1 {
    margin: 0;
    font-size: 2rem;
  }
  
  .status {
    display: flex;
    gap: 0.5rem;
  }
  
  .badge {
    padding: 0.5rem 1rem;
    border-radius: 20px;
    font-size: 0.875rem;
    font-weight: 600;
  }
  
  .badge.success {
    background: #10b981;
    color: white;
  }
  
  .badge.error {
    background: #ef4444;
    color: white;
  }
  
  .model-selector {
    display: flex;
    gap: 1rem;
    margin-bottom: 1rem;
    padding: 1rem;
    background: white;
    border-radius: 12px;
    box-shadow: 0 2px 8px rgba(0,0,0,0.1);
  }
  
  .model-selector label {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    cursor: pointer;
    font-weight: 500;
  }
  
  .chat-container {
    flex: 1;
    display: flex;
    flex-direction: column;
    background: white;
    border-radius: 12px;
    box-shadow: 0 4px 16px rgba(0,0,0,0.15);
    overflow: hidden;
  }
  
  .messages {
    flex: 1;
    overflow-y: auto;
    padding: 1.5rem;
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }
  
  .welcome {
    text-align: center;
    padding: 2rem;
    color: #666;
  }
  
  .welcome h2 {
    color: #333;
    margin-bottom: 1rem;
  }
  
  .welcome ul {
    text-align: left;
    max-width: 400px;
    margin: 0 auto;
  }
  
  .message {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
    animation: fadeIn 0.3s ease-in;
  }
  
  @keyframes fadeIn {
    from {
      opacity: 0;
      transform: translateY(10px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }
  
  .message-role {
    font-size: 0.875rem;
    font-weight: 600;
    color: #666;
  }
  
  .message-content {
    padding: 1rem;
    border-radius: 8px;
    white-space: pre-wrap;
    word-wrap: break-word;
  }
  
  .message.user .message-content {
    background: #f3f4f6;
    margin-left: auto;
    max-width: 70%;
  }
  
  .message.assistant .message-content {
    background: #eff6ff;
    border: 1px solid #dbeafe;
    max-width: 80%;
  }
  
  .message.error .message-content {
    background: #fee;
    border: 1px solid #fcc;
    color: #c00;
  }
  
  .typing {
    animation: pulse 1.5s infinite;
  }
  
  @keyframes pulse {
    0%, 100% {
      opacity: 1;
    }
    50% {
      opacity: 0.5;
    }
  }
  
  .input-container {
    display: flex;
    gap: 1rem;
    padding: 1.5rem;
    border-top: 1px solid #e5e7eb;
    background: #f9fafb;
  }
  
  textarea {
    flex: 1;
    padding: 0.75rem;
    border: 2px solid #e5e7eb;
    border-radius: 8px;
    font-size: 1rem;
    resize: none;
    font-family: inherit;
    transition: border-color 0.2s;
  }
  
  textarea:focus {
    outline: none;
    border-color: #667eea;
  }
  
  textarea:disabled {
    background: #f3f4f6;
    cursor: not-allowed;
  }
  
  button {
    padding: 0.75rem 2rem;
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    color: white;
    border: none;
    border-radius: 8px;
    font-size: 1rem;
    font-weight: 600;
    cursor: pointer;
    transition: transform 0.2s, opacity 0.2s;
  }
  
  button:hover:not(:disabled) {
    transform: translateY(-2px);
  }
  
  button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
</style>