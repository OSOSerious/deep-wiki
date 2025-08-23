<script>
  import { ideStore } from '../stores/ideStore.js';

  function getCurrentFileInfo() {
    if (!$ideStore.activeTab) return null;
    
    const tab = $ideStore.openTabs.get($ideStore.activeTab);
    if (!tab) return null;
    
    const fileName = $ideStore.activeTab.split(/[/\\]/).pop();
    return {
      fileName,
      language: tab.language,
      lines: tab.lines || 0,
      modified: tab.modified
    };
  }

  $: fileInfo = getCurrentFileInfo();
</script>

<div class="status-bar">
  <div class="status-left">
    <span>{$ideStore.status}</span>
  </div>
  
  <div class="status-right">
    {#if fileInfo}
      <span class="file-info">
        {fileInfo.fileName}
        {#if fileInfo.modified}
          <span class="modified">●</span>
        {/if}
      </span>
      <span class="separator">|</span>
      <span class="language">{fileInfo.language}</span>
      <span class="separator">|</span>
      <span class="lines">{fileInfo.lines} lines</span>
    {/if}
  </div>
</div>

<style>
  .status-bar {
    height: 24px;
    background: #007acc;
    color: white;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 12px;
    font-size: 12px;
  }

  .status-right {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .separator {
    opacity: 0.7;
  }

  .modified {
    color: #ffd700;
    margin-left: 4px;
  }

  .file-info {
    display: flex;
    align-items: center;
  }
</style>
