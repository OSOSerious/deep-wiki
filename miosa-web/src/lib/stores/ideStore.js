import { writable } from 'svelte/store';

// API base URL - adjust based on your backend
const API_BASE = 'http://localhost:8080/api/ide';

// Store for IDE state
function createIDEStore() {
  const { subscribe, set, update } = writable({
    fileTree: null,
    recentFiles: [],
    openTabs: new Map(),
    activeTab: null,
    currentFile: null,
    status: 'Ready'
  });

  return {
    subscribe,
    
    // Load file tree from backend
    async loadFileTree() {
      try {
        const response = await fetch(`${API_BASE}/tree`);
        if (!response.ok) throw new Error('Failed to load file tree');
        const tree = await response.json();
        
        update(state => ({
          ...state,
          fileTree: tree,
          status: 'File tree loaded'
        }));
      } catch (error) {
        update(state => ({
          ...state,
          status: `Error: ${error.message}`
        }));
        throw error;
      }
    },

    // Load recent files
    async loadRecentFiles() {
      try {
        const response = await fetch(`${API_BASE}/recent`);
        if (!response.ok) throw new Error('Failed to load recent files');
        const files = await response.json();
        
        update(state => ({
          ...state,
          recentFiles: files
        }));
      } catch (error) {
        console.error('Failed to load recent files:', error);
      }
    },

    // Search files
    async searchFiles(query) {
      try {
        const response = await fetch(`${API_BASE}/search?q=${encodeURIComponent(query)}&type=name`);
        if (!response.ok) throw new Error('Search failed');
        const results = await response.json();
        
        // Convert search results to tree-like structure for display
        const searchTree = {
          name: 'Search Results',
          isDir: true,
          children: results.map(file => ({
            ...file,
            children: []
          }))
        };
        
        update(state => ({
          ...state,
          fileTree: searchTree,
          status: `Found ${results.length} results`
        }));
      } catch (error) {
        update(state => ({
          ...state,
          status: `Search error: ${error.message}`
        }));
      }
    },

    // Open a file
    async openFile(path) {
      try {
        update(state => ({
          ...state,
          status: 'Loading file...'
        }));

        const response = await fetch(`${API_BASE}/file?path=${encodeURIComponent(path)}`);
        if (!response.ok) throw new Error('Failed to open file');
        const file = await response.json();
        
        update(state => {
          const newTabs = new Map(state.openTabs);
          newTabs.set(path, {
            content: file.content,
            modified: false,
            language: file.language,
            lines: file.lines
          });
          
          return {
            ...state,
            openTabs: newTabs,
            activeTab: path,
            currentFile: path,
            status: `Opened ${path.split(/[/\\]/).pop()} (${file.lines} lines)`
          };
        });
      } catch (error) {
        update(state => ({
          ...state,
          status: `Error opening file: ${error.message}`
        }));
        throw error;
      }
    },

    // Save current file
    async saveFile(path, content) {
      try {
        update(state => ({
          ...state,
          status: 'Saving...'
        }));

        const response = await fetch(`${API_BASE}/file`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json'
          },
          body: JSON.stringify({
            path: path,
            content: content
          })
        });

        if (!response.ok) throw new Error('Failed to save file');
        
        update(state => {
          const newTabs = new Map(state.openTabs);
          const tab = newTabs.get(path);
          if (tab) {
            tab.content = content;
            tab.modified = false;
          }
          
          return {
            ...state,
            openTabs: newTabs,
            status: 'File saved successfully'
          };
        });
      } catch (error) {
        update(state => ({
          ...state,
          status: `Save error: ${error.message}`
        }));
        throw error;
      }
    },

    // Mark tab as modified
    markTabModified(path) {
      update(state => {
        const newTabs = new Map(state.openTabs);
        const tab = newTabs.get(path);
        if (tab && !tab.modified) {
          tab.modified = true;
        }
        
        return {
          ...state,
          openTabs: newTabs
        };
      });
    },

    // Set active tab
    setActiveTab(path) {
      update(state => ({
        ...state,
        activeTab: path,
        currentFile: path
      }));
    },

    // Close tab
    closeTab(path) {
      update(state => {
        const newTabs = new Map(state.openTabs);
        newTabs.delete(path);
        
        let newActiveTab = state.activeTab;
        let newCurrentFile = state.currentFile;
        
        if (state.activeTab === path) {
          const remainingTabs = Array.from(newTabs.keys());
          if (remainingTabs.length > 0) {
            newActiveTab = remainingTabs[0];
            newCurrentFile = remainingTabs[0];
          } else {
            newActiveTab = null;
            newCurrentFile = null;
          }
        }
        
        return {
          ...state,
          openTabs: newTabs,
          activeTab: newActiveTab,
          currentFile: newCurrentFile
        };
      });
    },

    // Update status
    setStatus(message) {
      update(state => ({
        ...state,
        status: message
      }));
    }
  };
}

export const ideStore = createIDEStore();
