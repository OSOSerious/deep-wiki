<script lang="ts">
  import { onMount } from 'svelte';
  
  interface FileNode {
    name: string;
    path: string;
    isDir: boolean;
    children?: FileNode[];
    expanded?: boolean;
  }

  interface Tab {
    path: string;
    name: string;
    content: string;
    modified: boolean;
  }

  let fileTree: FileNode | null = null;
  let tabs: Tab[] = [];
  let activeTabIndex = -1;
  let editorContent = '';
  let loading = false;
  let error = '';

  const API_BASE = 'http://localhost:8089/api/ide';

  onMount(async () => {
    await loadFileTree();
  });

  async function loadFileTree() {
    loading = true;
    error = '';
    console.log('Loading file tree from:', `${API_BASE}/tree`);
    try {
      const response = await fetch(`${API_BASE}/tree`);
      if (!response.ok) throw new Error('Failed to load file tree');
      fileTree = await response.json();
      console.log('File tree loaded:', fileTree);
      if (fileTree) expandNode(fileTree);
    } catch (err: any) {
      console.error('Error loading file tree:', err);
      error = `Error loading file tree: ${err.message}`;
    } finally {
      loading = false;
    }
  }

  function expandNode(node: FileNode) {
    node.expanded = true;
    if (node.children) {
      // Only expand the first level directories
      node.children.forEach(child => {
        if (child.isDir) {
          child.expanded = true; // Expand first level dirs to show content
        }
      });
    }
  }

  async function openFile(path: string, name: string) {
    const existingIndex = tabs.findIndex(t => t.path === path);
    if (existingIndex !== -1) {
      activeTabIndex = existingIndex;
      editorContent = tabs[existingIndex].content;
      return;
    }

    loading = true;
    error = '';
    try {
      const response = await fetch(`${API_BASE}/file?path=${encodeURIComponent(path)}`);
      if (!response.ok) throw new Error('Failed to load file');
      const fileData = await response.json();
      
      const newTab: Tab = {
        path: fileData.path,
        name,
        content: fileData.content,
        modified: false
      };
      
      tabs = [...tabs, newTab];
      activeTabIndex = tabs.length - 1;
      editorContent = fileData.content;
    } catch (err: any) {
      error = `Error loading file: ${err.message}`;
    } finally {
      loading = false;
    }
  }

  async function saveFile() {
    if (activeTabIndex === -1) return;
    
    const tab = tabs[activeTabIndex];
    loading = true;
    error = '';
    
    try {
      const response = await fetch(`${API_BASE}/file`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          path: tab.path,
          content: editorContent
        })
      });
      
      if (!response.ok) throw new Error('Failed to save file');
      
      tabs[activeTabIndex] = { ...tab, content: editorContent, modified: false };
      tabs = [...tabs];
    } catch (err: any) {
      error = `Error saving file: ${err.message}`;
    } finally {
      loading = false;
    }
  }

  function closeTab(index: number) {
    tabs = tabs.filter((_, i) => i !== index);
    if (activeTabIndex >= tabs.length) {
      activeTabIndex = tabs.length - 1;
    }
    if (activeTabIndex >= 0) {
      editorContent = tabs[activeTabIndex].content;
    } else {
      editorContent = '';
    }
  }

  function toggleDir(node: FileNode) {
    node.expanded = !node.expanded;
    fileTree = fileTree;
  }

  function handleEditorInput() {
    if (activeTabIndex !== -1) {
      tabs[activeTabIndex].modified = editorContent !== tabs[activeTabIndex].content;
      tabs = [...tabs];
    }
  }

  function handleKeyDown(e: KeyboardEvent) {
    if ((e.ctrlKey || e.metaKey) && e.key === 's') {
      e.preventDefault();
      saveFile();
    }
  }
</script>

<svelte:window on:keydown={handleKeyDown} />

<div class="ide-container">
  <!-- Sidebar -->
  <div class="sidebar">
    <div class="sidebar-header">
      <h3>🚀 OSA IDE</h3>
      <button class="refresh-btn" on:click={loadFileTree} disabled={loading} title="Refresh file tree">
        {loading ? '⏳ Loading...' : '🔄 Refresh'}
      </button>
    </div>

    <!-- File Explorer -->
    <div class="file-explorer">
      {#if fileTree}
        <div class="file-tree">
          {@render FileTreeNode({node: fileTree, toggleDir, openFile})}
        </div>
      {/if}
    </div>
  </div>

  <!-- Editor -->
  <div class="editor-container">
    <!-- Tabs -->
    {#if tabs.length > 0}
      <div class="tabs">
        {#each tabs as tab, i}
          <div 
            class="tab" 
            class:active={i === activeTabIndex}
            on:click={() => {
              activeTabIndex = i;
              editorContent = tab.content;
            }}
          >
            <span class="tab-name">
              {tab.name} {tab.modified ? '*' : ''}
            </span>
            <button class="close-tab" on:click|stopPropagation={() => closeTab(i)}>
              ×
            </button>
          </div>
        {/each}
      </div>
    {/if}

    <!-- Editor Area -->
    {#if activeTabIndex !== -1}
      <div class="editor-header">
        <span class="file-path">{tabs[activeTabIndex].path}</span>
        <button class="save-btn" on:click={saveFile} disabled={loading}>
          {loading ? 'Saving...' : 'Save (Ctrl+S)'}
        </button>
      </div>
      <textarea
        class="editor"
        bind:value={editorContent}
        on:input={handleEditorInput}
        spellcheck="false"
      />
    {:else}
      <div class="welcome">
        <h2>Welcome to OSA IDE</h2>
        <p>Select a file from the explorer to start editing</p>
      </div>
    {/if}

    <!-- Error Display -->
    {#if error}
      <div class="error-message">
        {error}
        <button on:click={() => error = ''}>×</button>
      </div>
    {/if}
  </div>
</div>

{#snippet FileTreeNode({node, toggleDir, openFile, level = 0})}
  <div class="tree-node" style="padding-left: {level * 15}px">
    <div 
      class="tree-item"
      on:click={() => {
        if (node.isDir) {
          toggleDir(node);
        } else {
          openFile(node.path, node.name);
        }
      }}
    >
      <span class="icon">
        {#if node.isDir}
          {node.expanded ? '📂' : '📁'}
        {:else if node.name.endsWith('.go')}
          🐹
        {:else if node.name.endsWith('.svelte')}
          🔧
        {:else if node.name.endsWith('.js') || node.name.endsWith('.ts')}
          📜
        {:else if node.name.endsWith('.json')}
          📋
        {:else}
          📄
        {/if}
      </span>
      <span class="name">{node.name}</span>
    </div>
    
    {#if node.isDir && node.expanded && node.children}
      {#each node.children as child}
        {@render FileTreeNode({node: child, toggleDir, openFile, level: level + 1})}
      {/each}
    {/if}
  </div>
{/snippet}

<style>
  :global(body) {
    margin: 0;
    padding: 0;
  }

  .ide-container {
    display: flex;
    height: 100vh;
    font-family: 'Monaco', 'Menlo', monospace;
    background: #1e1e1e;
    color: #d4d4d4;
  }

  .sidebar {
    width: 280px;
    background: #252526;
    border-right: 1px solid #3e3e42;
    display: flex;
    flex-direction: column;
  }

  .sidebar-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 12px 15px;
    border-bottom: 1px solid #3e3e42;
  }

  .sidebar-header h3 {
    margin: 0;
    font-size: 14px;
    color: #cccccc;
  }

  .refresh-btn {
    background: none;
    border: none;
    color: #cccccc;
    cursor: pointer;
    font-size: 16px;
    padding: 4px;
  }

  .refresh-btn:hover:not(:disabled) {
    background: #3e3e42;
    border-radius: 3px;
  }

  .file-explorer {
    flex: 1;
    overflow-y: auto;
    padding: 8px 0;
  }

  .tree-node {
    user-select: none;
  }

  .tree-item {
    display: flex;
    align-items: center;
    padding: 3px 8px;
    cursor: pointer;
    font-size: 13px;
  }

  .tree-item:hover {
    background: #2a2d2e;
  }

  .tree-item .icon {
    margin-right: 6px;
  }

  .tree-item .name {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .editor-container {
    flex: 1;
    display: flex;
    flex-direction: column;
  }

  .tabs {
    display: flex;
    background: #2d2d30;
    border-bottom: 1px solid #3e3e42;
    overflow-x: auto;
  }

  .tab {
    display: flex;
    align-items: center;
    padding: 8px 12px;
    background: #2d2d30;
    border-right: 1px solid #3e3e42;
    cursor: pointer;
    font-size: 13px;
  }

  .tab.active {
    background: #1e1e1e;
  }

  .tab:hover {
    background: #2a2a2a;
  }

  .close-tab {
    background: none;
    border: none;
    color: #888;
    cursor: pointer;
    font-size: 18px;
    padding: 0 4px;
    margin-left: 8px;
  }

  .close-tab:hover {
    color: #fff;
  }

  .editor-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 8px 15px;
    background: #2d2d30;
    border-bottom: 1px solid #3e3e42;
  }

  .file-path {
    font-size: 12px;
    color: #888;
  }

  .save-btn {
    padding: 5px 12px;
    background: #007acc;
    color: white;
    border: none;
    border-radius: 3px;
    cursor: pointer;
    font-size: 12px;
  }

  .save-btn:hover:not(:disabled) {
    background: #005a9e;
  }

  .save-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .editor {
    flex: 1;
    padding: 15px;
    background: #1e1e1e;
    color: #d4d4d4;
    border: none;
    font-family: 'Monaco', 'Menlo', monospace;
    font-size: 14px;
    line-height: 1.6;
    resize: none;
    outline: none;
  }

  .welcome {
    flex: 1;
    display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: center;
    color: #666;
  }

  .welcome h2 {
    margin: 0 0 10px;
  }

  .error-message {
    position: fixed;
    bottom: 20px;
    right: 20px;
    background: #ff4444;
    color: white;
    padding: 10px 15px;
    border-radius: 5px;
    display: flex;
    align-items: center;
    gap: 10px;
    font-size: 13px;
    box-shadow: 0 2px 8px rgba(0,0,0,0.3);
  }

  .error-message button {
    background: none;
    border: none;
    color: white;
    font-size: 20px;
    cursor: pointer;
  }
</style>