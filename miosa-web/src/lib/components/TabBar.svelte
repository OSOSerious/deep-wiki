<script>
  import { ideStore } from '../stores/ideStore.js';

  function handleTabClick(path) {
    ideStore.setActiveTab(path);
  }

  function handleTabClose(event, path) {
    event.stopPropagation();
    ideStore.closeTab(path);
  }

  function getFileName(path) {
    return path.split(/[/\\]/).pop();
  }
</script>

<div class="tabs">
  {#each Array.from($ideStore.openTabs.entries()) as [path, tab]}
    <div 
      class="tab"
      class:active={$ideStore.activeTab === path}
      on:click={() => handleTabClick(path)}
      on:keydown={(e) => e.key === 'Enter' && handleTabClick(path)}
      role="button"
      tabindex="0"
    >
      <span class="tab-name">
        {getFileName(path)}{tab.modified ? ' •' : ''}
      </span>
      <button 
        class="tab-close"
        on:click={(e) => handleTabClose(e, path)}
        title="Close tab"
      >
        ×
      </button>
    </div>
  {/each}
</div>

<style>
  .tabs {
    display: flex;
    background: #2d2d30;
    border-bottom: 1px solid #3e3e42;
    min-height: 35px;
    overflow-x: auto;
  }

  .tab {
    display: flex;
    align-items: center;
    padding: 8px 16px;
    background: #2d2d30;
    border-right: 1px solid #3e3e42;
    cursor: pointer;
    font-size: 13px;
    white-space: nowrap;
    min-width: 120px;
    max-width: 200px;
  }

  .tab.active {
    background: #1e1e1e;
  }

  .tab:hover {
    background: #37373d;
  }

  .tab.active:hover {
    background: #1e1e1e;
  }

  .tab-name {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    margin-right: 8px;
  }

  .tab-close {
    background: none;
    border: none;
    color: #cccccc;
    cursor: pointer;
    padding: 2px 4px;
    border-radius: 2px;
    opacity: 0.6;
    font-size: 16px;
    line-height: 1;
  }

  .tab-close:hover {
    opacity: 1;
    background: rgba(255, 255, 255, 0.1);
  }
</style>
