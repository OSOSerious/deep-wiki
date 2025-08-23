<script>
  import { ideStore } from '../stores/ideStore.js';

  export let fileTree = null;
  export let recentFiles = [];

  $: ({ fileTree, recentFiles } = $ideStore);

  function getFileIcon(language) {
    const icons = {
      'go': '🐹',
      'javascript': '🟨',
      'typescript': '🔷',
      'python': '🐍',
      'html': '🌐',
      'css': '🎨',
      'json': '📋',
      'markdown': '📝',
      'yaml': '⚙️',
      'text': '📄'
    };
    return icons[language] || '📄';
  }

  function handleFileClick(file) {
    if (!file.isDir) {
      ideStore.openFile(file.path);
    }
  }

  function renderFileTree(node, level = 0) {
    if (!node) return [];
    
    const items = [];
    
    if (node.name) {
      items.push({
        ...node,
        level,
        icon: node.isDir ? '📁' : getFileIcon(node.language)
      });
    }
    
    if (node.children) {
      for (const child of node.children) {
        items.push(...renderFileTree(child, level + 1));
      }
    }
    
    return items;
  }

  $: fileItems = renderFileTree(fileTree);
</script>

<div class="file-explorer">
  {#each fileItems as item}
    <div 
      class="file-item" 
      class:selected={$ideStore.currentFile === item.path}
      style="padding-left: {12 + item.level * 16}px"
      on:click={() => handleFileClick(item)}
      on:keydown={(e) => e.key === 'Enter' && handleFileClick(item)}
      role="button"
      tabindex="0"
    >
      <div class="file-icon" class:folder-icon={item.isDir}>
        {item.icon}
      </div>
      <span>{item.name}</span>
    </div>
  {/each}
</div>

{#if recentFiles.length > 0}
  <div class="recent-files">
    <div class="recent-header">Recent Files</div>
    {#each recentFiles.slice(0, 10) as file}
      <div 
        class="file-item"
        on:click={() => handleFileClick(file)}
        on:keydown={(e) => e.key === 'Enter' && handleFileClick(file)}
        role="button"
        tabindex="0"
      >
        <div class="file-icon">
          {getFileIcon(file.language)}
        </div>
        <span>{file.name}</span>
      </div>
    {/each}
  </div>
{/if}
