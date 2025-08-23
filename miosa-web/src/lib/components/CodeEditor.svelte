<script>
  import { onMount, onDestroy } from 'svelte';
  import { EditorView, basicSetup } from 'codemirror';
  import { EditorState } from '@codemirror/state';
  import { oneDark } from '@codemirror/theme-one-dark';
  import { javascript } from '@codemirror/lang-javascript';
  import { html } from '@codemirror/lang-html';
  import { css } from '@codemirror/lang-css';
  import { json } from '@codemirror/lang-json';
  import { markdown } from '@codemirror/lang-markdown';
  import { python } from '@codemirror/lang-python';
  import { go } from '@codemirror/lang-go';
  import { ideStore } from '../stores/ideStore.js';

  let editorElement;
  let editorView;
  let currentContent = '';

  $: currentTab = $ideStore.openTabs.get($ideStore.activeTab);
  $: if (currentTab && editorView && currentTab.content !== currentContent) {
    updateEditorContent(currentTab.content);
  }

  function getLanguageExtension(language) {
    switch (language) {
      case 'javascript':
      case 'typescript':
        return javascript();
      case 'html':
        return html();
      case 'css':
      case 'scss':
        return css();
      case 'json':
        return json();
      case 'markdown':
        return markdown();
      case 'python':
        return python();
      case 'go':
        return go();
      default:
        return [];
    }
  }

  function updateEditorContent(content) {
    if (editorView && content !== currentContent) {
      currentContent = content;
      editorView.dispatch({
        changes: {
          from: 0,
          to: editorView.state.doc.length,
          insert: content
        }
      });
    }
  }

  function createEditor(content = '', language = 'text') {
    if (editorView) {
      editorView.destroy();
    }

    const extensions = [
      basicSetup,
      oneDark,
      getLanguageExtension(language),
      EditorView.updateListener.of((update) => {
        if (update.docChanged) {
          const newContent = update.state.doc.toString();
          currentContent = newContent;
          
          if ($ideStore.activeTab) {
            const tab = $ideStore.openTabs.get($ideStore.activeTab);
            if (tab && tab.content !== newContent) {
              ideStore.markTabModified($ideStore.activeTab);
            }
          }
        }
      }),
      EditorView.theme({
        '&': {
          height: '100%'
        },
        '.cm-scroller': {
          fontFamily: 'Consolas, Monaco, "Courier New", monospace'
        }
      })
    ];

    const state = EditorState.create({
      doc: content,
      extensions
    });

    editorView = new EditorView({
      state,
      parent: editorElement
    });

    currentContent = content;
  }

  async function saveCurrentFile() {
    if ($ideStore.activeTab && editorView) {
      try {
        const content = editorView.state.doc.toString();
        await ideStore.saveFile($ideStore.activeTab, content);
      } catch (error) {
        console.error('Save failed:', error);
      }
    }
  }

  function handleKeydown(event) {
    if (event.ctrlKey && event.key === 's') {
      event.preventDefault();
      saveCurrentFile();
    }
  }

  onMount(() => {
    createEditor();
    document.addEventListener('keydown', handleKeydown);
  });

  onDestroy(() => {
    if (editorView) {
      editorView.destroy();
    }
    document.removeEventListener('keydown', handleKeydown);
  });

  // Recreate editor when active tab changes
  $: if (editorElement && $ideStore.activeTab) {
    const tab = $ideStore.openTabs.get($ideStore.activeTab);
    if (tab) {
      createEditor(tab.content, tab.language);
    }
  }
</script>

<div class="editor-wrapper">
  {#if $ideStore.activeTab}
    <div bind:this={editorElement} class="editor"></div>
  {:else}
    <div class="empty-editor">
      <div>
        <h3>Welcome to OSA IDE</h3>
        <p>Select a file from the explorer to start editing</p>
        <div class="shortcuts">
          <p><strong>Shortcuts:</strong></p>
          <ul>
            <li><kbd>Ctrl+S</kbd> - Save file</li>
            <li><kbd>Ctrl+F</kbd> - Find in file</li>
            <li><kbd>Ctrl+H</kbd> - Replace in file</li>
          </ul>
        </div>
      </div>
    </div>
  {/if}
</div>

<style>
  .editor-wrapper {
    height: 100%;
    width: 100%;
  }

  .editor {
    height: 100%;
    width: 100%;
  }

  .empty-editor {
    display: flex;
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

  .shortcuts {
    margin-top: 2rem;
    text-align: left;
  }

  .shortcuts ul {
    list-style: none;
    padding: 0;
  }

  .shortcuts li {
    margin: 0.5rem 0;
  }

  kbd {
    background: #333;
    border: 1px solid #555;
    border-radius: 3px;
    padding: 2px 6px;
    font-size: 0.8em;
    color: #fff;
  }
</style>
