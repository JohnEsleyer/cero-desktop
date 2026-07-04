<script>
  import { onMount } from 'svelte';
  import { EventsOn } from '../wailsjs/runtime/runtime.js';
  import { 
    GetConnectionStatus, 
    GetDbPages, 
    GetDiscoveredDevices, 
    ConnectToDevice, 
    Disconnect, 
    AddPage, 
    UpdatePage, 
    DeletePage,
    RestorePage,
    HardDeletePage,
    MovePage,
    FetchPageContent
  } from '../wailsjs/go/main/App.js';
  import { marked } from 'marked';

  // Network State
  let connectionStatus = 'disconnected';
  let discoveredDevices = [];
  let connectedDevice = null;
  let manualIp = '';
  let manualPort = 9090;
  let manualPin = '';
  let connectionError = '';
  let pairingDeviceIp = '';

  // Pages State
  let allPages = [];
  let selectedPage = null;
  let navigationHistory = []; // Stack to track page selection history

  // Editor State
  let editorTitle = '';
  let editorContent = '';
  let editorEmoji = '📓';
  let viewMode = 'split'; // 'edit', 'preview', 'split'
  let showEmojiPicker = false;
  let pendingContentFetch = false;
  let saveTimeout;

  // Tree UI Expansion State
  let expandedPageIds = {};

  const curatedEmojis = [
    '📓', '📝', '📅', '💭', '💡', '🏷️', '✈️', '🏃', '💻', '🏠', 
    '🎨', '🎵', '📚', '✍️', '❤️', '🌟', '🍀', '☀️', '🌧️', '☕', 
    '🧠', '🔋', '🏡', '🎯'
  ];

  onMount(async () => {
    try {
      connectionStatus = await GetConnectionStatus();
      discoveredDevices = await GetDiscoveredDevices();
      allPages = await GetDbPages();
    } catch (e) {
      console.error("Initial load failed:", e);
    }

    EventsOn('connection-status', (status) => {
      connectionStatus = status;
      if (status === 'disconnected') {
        connectedDevice = null;
        allPages = [];
        selectedPage = null;
        navigationHistory = [];
      }
      connectionError = '';
    });

    EventsOn('discovered-devices', (devices) => {
      discoveredDevices = devices;
    });

    EventsOn('db-update', (pages) => {
      allPages = pages;
      if (selectedPage) {
        const updated = pages.find(p => p.id === selectedPage.id);
        if (!updated) {
          selectedPage = null;
          navigationHistory = [];
        } else if (updated.updated_at !== selectedPage.updated_at) {
          // Preserve content if incoming update has no content (metadata-only sync)
          const hasContent = updated.content && updated.content.length > 0;
          
          // Update selected page copy if changed remotely
          if (hasContent) {
            selectedPage = updated;
          } else {
            selectedPage = {...updated, content: selectedPage.content || ''};
          }
          
          // Only update edit text fields if the user is not actively editing them to prevent cursor jumping
          const isContentFocused = document.activeElement?.id === 'editor-textarea';
          const isTitleFocused = document.activeElement?.id === 'editor-title-input';

          if (hasContent && !isContentFocused) {
            editorContent = updated.content;
          }
          if (!isTitleFocused) {
            editorTitle = updated.title;
          }
          editorEmoji = updated.emoji;
        }
      }
    });

    EventsOn('pairing-required', (data) => {
      pairingDeviceIp = data.remoteAddress || 'Unknown';
    });

    EventsOn('page-content', (data) => {
      if (selectedPage && selectedPage.id === data.id) {
        // Only apply if content hasn't been modified by user since fetch was requested
        if (pendingContentFetch) {
          editorContent = data.content;
          selectedPage.content = data.content;
          pendingContentFetch = false;
        }
      }
    });
  });

  async function handleConnect(device) {
    connectionError = '';
    try {
      connectedDevice = device;
      const pin = prompt('Enter the auth PIN displayed on your mobile device:') || '';
      if (!pin) {
        connectedDevice = null;
        return;
      }
      await ConnectToDevice(device.ip, device.port, pin);
    } catch (err) {
      connectionError = err.toString();
      connectedDevice = null;
    }
  }

  async function handleConnectManually() {
    if (!manualIp) {
      connectionError = 'IP Address is required';
      return;
    }
    if (!manualPin) {
      connectionError = 'Auth PIN is required';
      return;
    }
    connectionError = '';
    const device = {
      ip: manualIp.trim(),
      port: parseInt(manualPort) || 9090,
      deviceName: 'Manual Entry'
    };
    
    try {
      connectedDevice = device;
      await ConnectToDevice(device.ip, device.port, manualPin.trim());
    } catch (err) {
      connectionError = err.toString();
      connectedDevice = null;
    }
  }

  async function handleDisconnect() {
    try {
      await Disconnect();
      connectedDevice = null;
      allPages = [];
      selectedPage = null;
      navigationHistory = [];
    } catch (err) {
      console.error(err);
    }
  }

  function toggleExpand(pageId) {
    expandedPageIds[pageId] = !expandedPageIds[pageId];
  }

  function selectPage(page, pushToHistory = true) {
    // If there is a pending debounced save, execute it immediately before switching
    if (saveTimeout) {
      clearTimeout(saveTimeout);
      savePage();
    }

    if (pushToHistory && selectedPage && selectedPage.id !== page.id) {
      navigationHistory = [...navigationHistory, selectedPage.id];
    }

    selectedPage = page;
    editorTitle = page.title;
    editorEmoji = page.emoji;
    showEmojiPicker = false;

    // Lazy load content if it's empty (metadata-only sync)
    if (!page.content) {
      pendingContentFetch = true;
      editorContent = '...';
      FetchPageContent(page.id).catch(err => {
        console.error("Failed to fetch content:", err);
        editorContent = '';
        pendingContentFetch = false;
      });
    } else {
      pendingContentFetch = false;
      editorContent = page.content;
    }
  }

  function goBack() {
    if (navigationHistory.length === 0) return;
    const prevId = navigationHistory[navigationHistory.length - 1];
    navigationHistory = navigationHistory.slice(0, -1);
    const prevPage = allPages.find(p => p.id === prevId);
    if (prevPage) {
      selectPage(prevPage, false);
    }
  }

  function savePage() {
    if (!selectedPage) return;
    UpdatePage(
      selectedPage.id,
      selectedPage.parent_id || '',
      editorTitle.trim() || 'Untitled',
      editorContent,
      editorEmoji
    ).catch(err => {
      console.error("Save error:", err);
    });
  }

  function savePageDebounced() {
    if (saveTimeout) clearTimeout(saveTimeout);
    saveTimeout = setTimeout(() => {
      savePage();
    }, 500); // 500ms debounce
  }

  function createPage(parentId = '') {
    AddPage(
      parentId,
      'New Page',
      '# New Page\n\nStart writing markdown here...',
      '📝'
    ).then(() => {
      if (parentId) {
        expandedPageIds[parentId] = true;
      }
    }).catch(err => {
      alert("Failed to create page: " + err);
    });
  }

  function deletePage(id) {
    if (confirm("Archive this page and all its subpages? You can restore from trash later.")) {
      DeletePage(id).then(() => {
        if (selectedPage && selectedPage.id === id) {
          selectedPage = null;
          navigationHistory = [];
        }
      }).catch(err => {
        alert("Failed to archive page: " + err);
      });
    }
  }

  function movePage(id) {
    const pages = allPages.filter(p => p.id !== id);
    const options = pages.map((p, i) => `${i + 1}: ${p.emoji} ${p.title || 'Untitled'}`);
    options.unshift('0: 📂 Root Level');
    
    const choice = prompt('Move page to:\n' + options.join('\n'));
    if (choice == null) return;
    
    const idx = parseInt(choice);
    if (idx === 0) {
      MovePage(id, '');
    } else if (idx > 0 && idx <= pages.length) {
      MovePage(id, pages[idx - 1].id);
    }
  }

  function selectEmoji(emoji) {
    editorEmoji = emoji;
    showEmojiPicker = false;
    savePage();
  }

  function insertMarkdown(prefix, suffix) {
    const textarea = document.getElementById('editor-textarea');
    if (!textarea) return;

    const start = textarea.selectionStart;
    const end = textarea.selectionEnd;
    const text = textarea.value;
    const selected = text.substring(start, end);
    
    editorContent = text.substring(0, start) + prefix + selected + suffix + text.substring(end);
    
    // Focus back and set selection
    setTimeout(() => {
      textarea.focus();
      const newCursorPos = start + prefix.length + selected.length;
      textarea.setSelectionRange(newCursorPos, newCursorPos);
      savePage();
    }, 0);
  }

  // Reactive markdown parser
  $: parsedMarkdown = marked.parse(editorContent || '');

  // Recursive page nodes finder
  $: rootPages = allPages.filter(p => !p.parent_id);
  $: getChildrenOf = (parentId) => allPages.filter(p => p.parent_id === parentId);
</script>

<main class="app-layout">
  <!-- LEFT COLUMN: Sidebar -->
  <aside class="sidebar">
    <div class="sidebar-header">
      <div class="logo-section">
        <span class="logo-icon">📓</span>
        <span class="logo-text">Cero Desktop</span>
      </div>
      <div class="status-indicator {connectionStatus}">
        <span class="indicator-dot"></span>
        <span class="indicator-text">{connectionStatus}</span>
      </div>
    </div>

    <!-- Sync Server Discovered list -->
    <div class="sidebar-section">
      <div class="section-title">
        <span>Link Devices</span>
        {#if connectionStatus !== 'connected'}
          <span class="scanning-pulse">Scanning...</span>
        {/if}
      </div>
      <div class="discovered-container">
        {#if connectionStatus === 'connected'}
          <div class="active-connection-card">
            <div class="active-info">
              <span class="active-label">Connected to</span>
              <span class="active-val">{connectedDevice ? connectedDevice.deviceName : 'Mobile Server'}</span>
            </div>
            <button class="btn btn-secondary w-full" on:click={handleDisconnect}>Disconnect Link</button>
          </div>
        {:else}
          {#if discoveredDevices.length === 0}
            <div class="no-devices">
              Searching for mobile server on WiFi...
            </div>
          {:else}
            <div class="devices-list">
              {#each discoveredDevices as device}
                <div class="device-row">
                  <div class="device-meta">
                    <span class="name">{device.deviceName}</span>
                    <span class="ip">{device.ip}:{device.port}</span>
                  </div>
                  <button class="btn btn-primary connect-btn" on:click={() => handleConnect(device)}>Link</button>
                </div>
              {/each}
            </div>
          {/if}
        {/if}
      </div>
    </div>

    <!-- Page Tree Navigation -->
    <div class="sidebar-section flex-grow">
      <div class="section-title">
        <span>Journal Pages</span>
        {#if connectionStatus === 'connected'}
          <button class="add-btn" title="New root note" on:click={() => createPage('')}>+</button>
        {/if}
      </div>
      
      <div class="tree-container">
        {#if connectionStatus !== 'connected'}
          <div class="tree-placeholder">Connect to your phone to view and edit pages.</div>
        {:else if rootPages.length === 0}
          <div class="tree-placeholder">No pages yet. Create one!</div>
        {:else}
          <!-- Render root tree nodes -->
          <div class="tree-list">
            {#each rootPages as page}
              <div class="tree-node-wrapper">
                <!-- Recursive tree render via svelte:self -->
                <svelte:component this={TreeRender} {page} {allPages} {selectedPage} {expandedPageIds} {selectPage} {createPage} {deletePage} {toggleExpand} {getChildrenOf} />
              </div>
            {/each}
          </div>
        {/if}
      </div>
    </div>

    <!-- Manual Connection Fallback -->
    {#if connectionStatus !== 'connected'}
      <div class="sidebar-section manual-section">
        <span class="section-title">Manual Link</span>
        <div class="manual-fields">
          <input type="text" bind:value={manualIp} placeholder="IP Address (e.g. 192.168.1.5)" />
          <input type="text" bind:value={manualPin} placeholder="Auth PIN (from mobile device)" maxlength="4" class="pin-input" />
          <div class="row">
            <input type="number" bind:value={manualPort} placeholder="9090" />
            <button class="btn btn-primary" on:click={handleConnectManually}>Link</button>
          </div>
          {#if connectionError}
            <div class="error-msg">{connectionError}</div>
          {/if}
        </div>
      </div>
    {/if}
  </aside>

  <!-- RIGHT COLUMN: Main Editor -->
  <section class="editor-workspace">
    {#if connectionStatus !== 'connected'}
      <!-- Disconnected Welcome Screen -->
      <div class="welcome-container">
        <div class="welcome-card">
          <div class="pulse-ring">
            <span class="pulse-icon">📓</span>
          </div>
          <h1>Cero Journal Workspace</h1>
          <p>
            Connect to your mobile phone (Central database) to write and synchronize your notes in real time. 
            All modifications are stored offline on your phone first.
          </p>
          <div class="steps-grid">
            <div class="step-card">
              <span class="num">1</span>
              <p>Start <strong>Cero</strong> on your mobile phone.</p>
            </div>
            <div class="step-card">
              <span class="num">2</span>
              <p>Turn on <strong>Cero Sync Hub</strong> switch in the app drawer.</p>
            </div>
            <div class="step-card">
              <span class="num">3</span>
              <p>Select your device in the <strong>Link Devices</strong> list on the left.</p>
            </div>
          </div>
        </div>
      </div>
    {:else if !selectedPage}
      <!-- Connected Empty State -->
      <div class="welcome-container">
        <div class="connected-placeholder">
          <span class="placeholder-emoji">📝</span>
          <h2>Select or Create a Page</h2>
          <p>Choose a journal entry from the sidebar tree, or create a new root note to start writing markdown.</p>
          <button class="btn btn-primary" on:click={() => createPage('')}>Create New Page</button>
        </div>
      </div>
    {:else}
      <!-- Connected Editor Workspace -->
      <div class="editor-header">
        <div class="breadcrumbs">
          {#if navigationHistory.length > 0}
            <button class="btn-back" on:click={goBack} title="Go Back">← Back</button>
            <span class="divider">|</span>
          {/if}
          <span class="breadcrumb-root">Journal</span>
          <span class="divider">/</span>
          <span class="breadcrumb-page">{editorTitle || 'Untitled'}</span>
        </div>

        <div class="header-controls">
          <!-- View Mode Toggle -->
          <div class="view-mode-tabs">
            <button class="tab-btn {viewMode === 'edit' ? 'active' : ''}" on:click={() => viewMode = 'edit'}>Edit</button>
            <button class="tab-btn {viewMode === 'preview' ? 'active' : ''}" on:click={() => viewMode = 'preview'}>Preview</button>
            <button class="tab-btn {viewMode === 'split' ? 'active' : ''}" on:click={() => viewMode = 'split'}>Split Screen</button>
          </div>

          <button class="btn btn-secondary" on:click={() => movePage(selectedPage.id)}>Move To...</button>
          <button class="btn btn-danger" on:click={() => deletePage(selectedPage.id)}>Archive</button>
        </div>
      </div>

      <div class="editor-body {viewMode}">
        <!-- LEFT HALF: Writing Area -->
        {#if viewMode === 'edit' || viewMode === 'split'}
          <div class="editor-pane">
            <!-- Markdown Format Toolbar -->
            <div class="format-toolbar">
              <button class="tool-btn" title="Bold" on:click={() => insertMarkdown('**', '**')}>B</button>
              <button class="tool-btn" title="Italic" on:click={() => insertMarkdown('*', '*')}>I</button>
              <button class="tool-btn" title="Header" on:click={() => insertMarkdown('# ', '')}>H</button>
              <button class="tool-btn" title="List" on:click={() => insertMarkdown('- ', '')}>List</button>
              <button class="tool-btn" title="Checkbox" on:click={() => insertMarkdown('- [ ] ', '')}>Todo</button>
              <button class="tool-btn" title="Code" on:click={() => insertMarkdown('`', '`')}>Code</button>
            </div>

            <div class="pane-content-wrapper">
              <!-- Emoji Picker -->
              <div class="emoji-container">
                <button class="emoji-trigger" on:click={() => showEmojiPicker = !showEmojiPicker}>
                  {editorEmoji}
                </button>
                {#if showEmojiPicker}
                  <div class="emoji-dropdown">
                    {#each curatedEmojis as emoji}
                      <button class="emoji-picker-btn" on:click={() => selectEmoji(emoji)}>{emoji}</button>
                    {/each}
                  </div>
                {/if}
              </div>

              <!-- Title Input -->
              <input 
                id="editor-title-input"
                type="text" 
                class="title-input" 
                bind:value={editorTitle} 
                on:input={savePageDebounced} 
                placeholder="Untitled" 
              />

              <!-- Monospace Editor Textarea -->
              <textarea 
                id="editor-textarea"
                class="textarea-editor" 
                bind:value={editorContent} 
                on:input={savePageDebounced} 
                placeholder="Start writing notes..."
              ></textarea>
            </div>
          </div>
        {/if}

        <!-- RIGHT HALF: Live Preview -->
        {#if viewMode === 'preview' || viewMode === 'split'}
          <div class="preview-pane">
            <div class="pane-content-wrapper">
              <div class="preview-emoji-header">{editorEmoji}</div>
              <h1 class="preview-title">{editorTitle || 'Untitled'}</h1>
              <div class="markdown-rendered">
                {@html parsedMarkdown}
              </div>
            </div>
          </div>
        {/if}
      </div>
    {/if}
  </section>
</main>

<!-- HELPER RECURSIVE COMPONENT FOR SIDEBAR TREE (Rendered Inline as a helper subcomponent) -->
<script context="module">
  // We define the TreeRender component structure to enable infinite nested listings in Wails
  import { default as TreeRender } from './TreeRender.svelte';
</script>

<style>
  .app-layout {
    display: flex;
    width: 100vw;
    height: 100vh;
    overflow: hidden;
    background-color: #121212;
    color: #e2e8f0;
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
  }

  /* SIDEBAR STYLES */
  .sidebar {
    width: 290px;
    background-color: #1a1a1a;
    border-right: 1px solid #2e2e2e;
    display: flex;
    flex-direction: column;
    height: 100%;
    z-index: 10;
  }

  .sidebar-header {
    padding: 16px 20px;
    border-bottom: 1px solid #2e2e2e;
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .logo-section {
    display: flex;
    align-items: center;
    gap: 8px;
    font-weight: 700;
  }

  .logo-icon {
    font-size: 20px;
  }

  .logo-text {
    font-size: 15px;
    background: linear-gradient(135deg, #a5b4fc, #c084fc);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
  }

  .status-indicator {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 3px 8px;
    border-radius: 12px;
    font-size: 9px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .status-indicator.disconnected { background: rgba(239, 68, 68, 0.1); color: #f87171; }
  .status-indicator.connecting { background: rgba(168, 85, 247, 0.1); color: #c084fc; }
  .status-indicator.connected { background: rgba(34, 197, 94, 0.1); color: #4ade80; }

  .indicator-dot {
    width: 5px;
    height: 5px;
    border-radius: 50%;
  }
  .disconnected .indicator-dot { background-color: #f87171; }
  .connecting .indicator-dot { background-color: #c084fc; animation: pulse 1.5s infinite; }
  .connected .indicator-dot { background-color: #4ade80; }

  @keyframes pulse {
    0% { transform: scale(0.9); opacity: 0.6; }
    50% { transform: scale(1.3); opacity: 1; }
    100% { transform: scale(0.9); opacity: 0.6; }
  }

  .sidebar-section {
    padding: 16px 20px;
    border-bottom: 1px solid #2e2e2e;
    display: flex;
    flex-direction: column;
  }

  .flex-grow {
    flex-grow: 1;
    overflow-y: auto;
  }

  .section-title {
    font-size: 10px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 1px;
    color: #8e8e8e;
    margin-bottom: 10px;
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .scanning-pulse {
    color: #818cf8;
    font-size: 9px;
    text-transform: none;
    letter-spacing: 0;
  }

  .discovered-container {
    background-color: #121212;
    border: 1px solid #2e2e2e;
    border-radius: 8px;
    padding: 10px;
  }

  .active-connection-card {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .active-info {
    display: flex;
    justify-content: space-between;
    font-size: 11px;
  }

  .active-label { color: #8e8e8e; }
  .active-val { color: #4ade80; font-weight: 700; }

  .no-devices {
    font-size: 11px;
    color: #6c6c6c;
    text-align: center;
    padding: 4px 0;
  }

  .devices-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .device-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .device-meta {
    display: flex;
    flex-direction: column;
  }

  .device-meta .name {
    font-size: 12px;
    font-weight: 700;
  }

  .device-meta .ip {
    font-size: 10px;
    color: #6c6c6c;
    font-family: monospace;
  }

  .tree-container {
    flex-grow: 1;
    overflow-y: auto;
  }

  .tree-placeholder {
    font-size: 12px;
    color: #6c6c6c;
    line-height: 1.5;
    padding: 8px 0;
  }

  .add-btn {
    background: transparent;
    border: none;
    color: #8e8e8e;
    cursor: pointer;
    font-size: 16px;
    line-height: 1;
    padding: 0 4px;
  }

  .add-btn:hover {
    color: white;
  }

  /* MANUAL CONNECTION */
  .manual-section {
    border-bottom: none;
    background-color: #151515;
    margin-top: auto;
  }

  .manual-fields {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .manual-fields input {
    background-color: #121212;
    border: 1px solid #2e2e2e;
    border-radius: 6px;
    color: white;
    font-size: 11px;
    padding: 6px 10px;
    outline: none;
  }

  .manual-fields .row {
    display: flex;
    gap: 6px;
  }

  .manual-fields .row input {
    width: 60px;
  }

  .pin-input {
    letter-spacing: 4px;
    font-size: 16px !important;
    text-align: center;
    font-weight: bold;
  }

  .manual-fields .row button {
    flex-grow: 1;
    padding: 4px;
    font-size: 11px;
  }

  .error-msg {
    color: #f87171;
    font-size: 10px;
    margin-top: 4px;
    word-break: break-all;
  }

  /* MAIN WORKSPACE */
  .editor-workspace {
    flex-grow: 1;
    display: flex;
    flex-direction: column;
    background-color: #121212;
    height: 100%;
  }

  /* WELCOME INTERFACE */
  .welcome-container {
    flex-grow: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 40px;
  }

  .welcome-card {
    max-width: 580px;
    text-align: center;
    background-color: #1a1a1a;
    border: 1px solid #2e2e2e;
    border-radius: 20px;
    padding: 40px;
    box-shadow: 0 12px 36px rgba(0,0,0,0.4);
  }

  .pulse-ring {
    width: 72px;
    height: 72px;
    border-radius: 50%;
    background-color: rgba(165, 180, 252, 0.05);
    border: 1px solid rgba(165, 180, 252, 0.15);
    display: flex;
    align-items: center;
    justify-content: center;
    margin: 0 auto 20px;
  }

  .pulse-icon {
    font-size: 32px;
  }

  .welcome-card h1 {
    font-size: 24px;
    font-weight: 800;
    margin: 0 0 10px 0;
  }

  .welcome-card p {
    font-size: 13.5px;
    color: #8e8e8e;
    line-height: 1.6;
    margin: 0 0 30px 0;
  }

  .steps-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 16px;
    text-align: left;
  }

  .step-card {
    background-color: #121212;
    border: 1px solid #2e2e2e;
    border-radius: 12px;
    padding: 16px;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .step-card .num {
    width: 20px;
    height: 20px;
    border-radius: 50%;
    background-color: #818cf8;
    color: white;
    font-weight: 700;
    font-size: 10px;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .step-card p {
    font-size: 11px;
    line-height: 1.4;
    color: #d1d5db;
    margin: 0;
  }

  .connected-placeholder {
    text-align: center;
    color: #6c6c6c;
  }

  .placeholder-emoji {
    font-size: 48px;
    display: block;
    margin-bottom: 12px;
  }

  .connected-placeholder h2 {
    color: white;
    font-size: 18px;
    margin: 0 0 6px 0;
  }

  .connected-placeholder p {
    font-size: 13px;
    margin: 0 0 20px 0;
    max-width: 320px;
  }

  /* EDITOR WORKSPACE ACTIVE */
  .editor-header {
    background-color: #1a1a1a;
    border-bottom: 1px solid #2e2e2e;
    padding: 14px 30px;
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .breadcrumbs {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 12px;
  }

  .breadcrumb-root { color: #8e8e8e; }
  .divider { color: #444; }
  .breadcrumb-page { color: white; font-weight: 600; }

  .header-controls {
    display: flex;
    align-items: center;
    gap: 16px;
  }

  .view-mode-tabs {
    background-color: #121212;
    border: 1px solid #2e2e2e;
    border-radius: 6px;
    padding: 2px;
    display: flex;
  }

  .tab-btn {
    background: transparent;
    border: none;
    color: #8e8e8e;
    font-size: 11px;
    font-weight: 600;
    padding: 4px 12px;
    cursor: pointer;
    border-radius: 4px;
    transition: all 0.2s ease;
  }

  .tab-btn.active {
    background-color: #2e2e2e;
    color: white;
  }

  .editor-body {
    flex-grow: 1;
    display: flex;
    height: calc(100% - 60px);
    overflow: hidden;
  }

  /* Split screen layout options */
  .editor-body.edit .editor-pane { width: 100%; }
  .editor-body.preview .preview-pane { width: 100%; }
  .editor-body.split .editor-pane { width: 50%; border-right: 1px solid #2e2e2e; }
  .editor-body.split .preview-pane { width: 50%; }

  .editor-pane, .preview-pane {
    height: 100%;
    display: flex;
    flex-direction: column;
    overflow-y: auto;
  }

  .pane-content-wrapper {
    padding: 40px;
    max-width: 720px;
    width: 100%;
    margin: 0 auto;
    display: flex;
    flex-direction: column;
    min-height: 100%;
    box-sizing: border-box;
  }

  .format-toolbar {
    background-color: #151515;
    border-bottom: 1px solid #2e2e2e;
    padding: 4px 20px;
    display: flex;
    gap: 4px;
  }

  .tool-btn {
    background: transparent;
    border: none;
    color: #8e8e8e;
    font-size: 11px;
    padding: 4px 8px;
    cursor: pointer;
    border-radius: 4px;
  }

  .tool-btn:hover {
    background-color: #2e2e2e;
    color: white;
  }

  .emoji-container {
    position: relative;
    margin-bottom: 16px;
    align-self: flex-start;
  }

  .emoji-trigger {
    background: transparent;
    border: none;
    font-size: 48px;
    cursor: pointer;
    padding: 6px;
    border-radius: 12px;
    transition: background-color 0.2s ease;
  }

  .emoji-trigger:hover {
    background-color: #222;
  }

  .emoji-dropdown {
    position: absolute;
    top: 60px;
    left: 0;
    background-color: #1a1a1a;
    border: 1px solid #2e2e2e;
    border-radius: 8px;
    padding: 10px;
    display: grid;
    grid-template-columns: repeat(6, 1fr);
    gap: 6px;
    width: 220px;
    box-shadow: 0 10px 25px rgba(0,0,0,0.5);
    z-index: 20;
  }

  .emoji-picker-btn {
    background: transparent;
    border: none;
    font-size: 20px;
    cursor: pointer;
    padding: 4px;
    border-radius: 4px;
  }

  .emoji-picker-btn:hover {
    background-color: #2e2e2e;
  }

  .title-input {
    background: transparent;
    border: none;
    font-size: 28px;
    font-weight: 800;
    color: white;
    width: 100%;
    outline: none;
    margin-bottom: 20px;
  }

  .textarea-editor {
    background: transparent;
    border: none;
    resize: none;
    flex-grow: 1;
    width: 100%;
    outline: none;
    color: #e2e8f0;
    font-size: 14.5px;
    line-height: 1.6;
    font-family: "Fira Code", monospace;
  }

  /* PREVIEW PANE STYLES */
  .preview-pane {
    background-color: #0d0d0d;
  }

  .preview-emoji-header {
    font-size: 48px;
    margin-bottom: 20px;
  }

  .preview-title {
    font-size: 28px;
    font-weight: 800;
    margin: 0 0 20px 0;
  }

  .markdown-rendered {
    font-size: 14.5px;
    line-height: 1.65;
    color: #cbd5e1;
  }

  .markdown-rendered :global(h1) { font-size: 20px; border-bottom: 1px solid #2e2e2e; padding-bottom: 6px; margin: 24px 0 12px 0; color: white; }
  .markdown-rendered :global(h2) { font-size: 17px; margin: 20px 0 10px 0; color: white; }
  .markdown-rendered :global(h3) { font-size: 15px; margin: 16px 0 8px 0; color: white; }
  .markdown-rendered :global(code) { background-color: #1a1a1a; padding: 2px 4px; border-radius: 4px; font-family: monospace; font-size: 12px; }
  .markdown-rendered :global(pre) { background-color: #1a1a1a; padding: 12px; border-radius: 8px; overflow-x: auto; }
  .markdown-rendered :global(pre code) { background-color: transparent; padding: 0; }
  .markdown-rendered :global(ul), .markdown-rendered :global(ol) { padding-left: 20px; }
  .markdown-rendered :global(li) { margin-bottom: 4px; }

  /* FETCH CONTENT PROMPT */
  /* UNIVERSAL BUTTONS */
  .btn {
    border-radius: 6px;
    font-size: 12px;
    font-weight: 600;
    padding: 6px 12px;
    cursor: pointer;
    border: none;
    transition: all 0.2s ease;
    text-align: center;
  }

  .btn-primary { background-color: #818cf8; color: white; }
  .btn-primary:hover { background-color: #6366f1; }
  
  .btn-secondary { background-color: #2e2e2e; color: #cbd5e1; }
  .btn-secondary:hover { background-color: #3e3e3e; }

  .btn-danger { background-color: rgba(239,68,68,0.1); color: #f87171; }
  .btn-danger:hover { background-color: rgba(239,68,68,0.25); }

  .btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-back {
    background: transparent;
    border: none;
    color: #8e8e8e;
    cursor: pointer;
    font-size: 11px;
    font-weight: 600;
    padding: 4px 8px;
    border-radius: 4px;
    transition: all 0.2s ease;
  }

  .btn-back:hover {
    background-color: #2e2e2e;
    color: white;
  }

  .w-full { width: 100%; }
</style>
