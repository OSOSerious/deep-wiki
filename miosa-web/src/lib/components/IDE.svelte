<script>
  import { onMount } from 'svelte';

  let files = [];
  let currentFile = null;
  let fileContent = '';
  let loading = true;
  let error = null;

  onMount(async () => {
    try {
      const response = await fetch('http://localhost:8080/api/ide/tree');
      if (response.ok) {
        const tree = await response.json();
        files = flattenTree(tree);
        loading = false;
      } else {
        throw new Error('Failed to load files');
      }
    } catch (err) {
      error = err.message;
      loading = false;
    }
  });

  function flattenTree(node, level = 0) {
    let result = [];
    if (node.name) {
      result.push({ ...node, level });
    }
    if (node.children) {
      for (const child of node.children) {
        result.push(...flattenTree(child, level + 1));
      }
    }
    return result;
  }

  async function openFile(file) {
    if (file.isDir) return;
    
    try {
      const response = await fetch(`http://localhost:8080/api/ide/file?path=${encodeURIComponent(file.path)}`);
      if (response.ok) {
        const data = await response.json();
        currentFile = file;
        fileContent = data.content;
      }
    } catch (err) {
      console.error('Failed to open file:', err);
    }
  }

  async function saveFile() {
    if (!currentFile) return;
    
    try {
      const response = await fetch('http://localhost:8080/api/ide/file', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          path: currentFile.path,
          content: fileContent
        })
      });
      
      if (response.ok) {
        alert('File saved successfully!');
      }
    } catch (err) {
      alert('Failed to save file: ' + err.message);
    }
  }

  function handleKeydown(event) {
    if (event.ctrlKey && event.key === 's') {
      event.preventDefault();
      saveFile();
    }
  }
</script>

<svelte:window on:keydown={handleKeydown} />

<div class="ide-container">
  <div class="sidebar">
    <div class="sidebar-header">
      🗂️ OSA IDE Explorer
    </div>
    
    {#if loading}
      <div class="loading">Loading files...</div>
    {:else if error}
      <div class="error">Error: {error}</div>
    {:else}
      <div class="file-list">
        {#each files as file}
          <div 
            class="file-item" 
            class:selected={currentFile?.path === file.path}
            style="padding-left: {12 + file.level * 16}px"
            on:click={() => openFile(file)}
            on:keydown={(e) => e.key === 'Enter' && openFile(file)}
            role="button"
            tabindex="0"
          >
            <span class="file-icon">
              {file.isDir ? '📁' : '📄'}
            </span>
            <span>{file.name}</span>
          </div>
        {/each}
      </div>
    {/if}
  </div>

  <div class="main-content">
    <div class="editor-header">
      {#if currentFile}
        <span class="file-name">{currentFile.name}</span>
        <button class="save-btn" on:click={saveFile}>💾 Save (Ctrl+S)</button>
      {:else}
        <span>Select a file to edit</span>
      {/if}
    </div>
    
    <div class="editor-container">
      {#if currentFile}
        <textarea 
          bind:value={fileContent}
          class="editor"
          placeholder="File content will appear here..."
        ></textarea>
      {:else}
        <div class="empty-editor">
          <h3>Welcome to OSA IDE</h3>
          <p>Select a file from the explorer to start editing</p>
        </div>
      {/if}
    </div>
    
    <div class="status-bar">
      {#if currentFile}
        File: {currentFile.path} | Lines: {fileContent.split('\n').length}
      {:else}
        Ready
      {/if}
    </div>
  </div>
</div>

<style>
  .ide-container {
    display: flex;
    height: 100vh;
    width: 100vw;
    font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
    background: #1e1e1e;
    color: #d4d4d4;
  }

  .sidebar {
    width: 300px;
    background: #252526;
    border-right: 1px solid #3e3e42;
    display: flex;
    flex-direction: column;
  }

  .sidebar-header {
    padding: 12px;
    background: #2d2d30;
    border-bottom: 1px solid #3e3e42;
    font-weight: 600;
    font-size: 14px;
  }

  .file-list {
    flex: 1;
    overflow-y: auto;
  }

  .file-item {
    padding: 6px 12px;
    cursor: pointer;
    display: flex;
    align-items: center;
    font-size: 13px;
    border-bottom: 1px solid #2d2d30;
    user-select: none;
  }

  .file-item:hover {
    background: #2a2d2e;
  }

  .file-item.selected {
    background: #094771;
  }

  .file-icon {
    margin-right: 8px;
    font-size: 12px;
  }

  .main-content {
    flex: 1;
    display: flex;
    flex-direction: column;
  }

  .editor-header {
    background: #2d2d30;
    border-bottom: 1px solid #3e3e42;
    padding: 8px 16px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    font-size: 13px;
  }

  .file-name {
    font-weight: 600;
  }

  .save-btn {
    background: #0e639c;
    color: white;
    border: none;
    padding: 6px 12px;
    border-radius: 2px;
    cursor: pointer;
    font-size: 12px;
  }

  .save-btn:hover {
    background: #1177bb;
  }

  .editor-container {
    flex: 1;
    position: relative;
  }

  .editor {
    width: 100%;
    height: 100%;
    background: #1e1e1e;
    color: #d4d4d4;
    border: none;
    outline: none;
    font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
    font-size: 14px;
    line-height: 1.5;
    padding: 16px;
    resize: none;
  }

  .empty-editor {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: #888;
    text-align: center;
  }

  .empty-editor h3 {
    margin-bottom: 1rem;
    color: #ccc;
  }

  .status-bar {
    height: 24px;
    background: #007acc;
    color: white;
    display: flex;
    align-items: center;
    padding: 0 12px;
    font-size: 12px;
  }

  .loading {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 200px;
    font-size: 14px;
    color: #888;
  }

  .error {
    color: #f48771;
    background: #5a1d1d;
    padding: 8px 12px;
    margin: 8px;
    border-radius: 4px;
    font-size: 12px;
  }
</style>
