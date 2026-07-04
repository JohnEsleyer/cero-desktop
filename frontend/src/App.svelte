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
    GetCards,
    FetchCards,
    AddCard,
    UpdateCard,
    DeleteCard,
    ReorderCards,
    ListWorkspaces,
    CreateWorkspace,
    SwitchWorkspace,
    GetActiveWorkspace
  } from '../wailsjs/go/main/App.js';

  // Network State
  let connectionStatus = 'disconnected';
  let discoveredDevices = [];
  let connectedDevice = null;
  let manualIp = '';
  let manualPort = 9090;
  let manualPin = '';
  let connectionError = '';
  let pairingDeviceIp = '';

  // Workspace State
  let workspaces = [];
  let activeWorkspace = '';
  let showWorkspaceDropdown = false;
  let newWorkspaceName = '';

  // Pages State
  let allPages = [];
  let selectedPage = null;
  let navigationHistory = [];

  // Cards State
  let pageCards = [];
  let selectedCardId = null;

  // Tab State
  let activeTab = 'main';

  // Editor State
  let editorTitle = '';
  let editorEmoji = '📓';
  let showEmojiPicker = false;
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
      await loadWorkspaces();
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
          selectedPage = updated;
          const isTitleFocused = document.activeElement?.id === 'editor-title-input';
          if (!isTitleFocused) {
            editorTitle = updated.title;
          }
          editorEmoji = updated.emoji;
        }
      }
    });

    EventsOn('workspace-status', (data) => {
      activeWorkspace = data.activeWorkspace;
      workspaces = data.availableWorkspaces || [];
    });

    EventsOn('cards-update', (data) => {
      const targetPageId = activeTab === 'main' ? selectedPage?.id : activeTab;
      if (targetPageId && data.pageId === targetPageId) {
        pageCards = data.cards || [];
        if (selectedCardId && !pageCards.find(c => c.id === selectedCardId)) {
          selectedCardId = null;
        }
      }
    });

    EventsOn('pairing-required', (data) => {
      pairingDeviceIp = data.remoteAddress || 'Unknown';
    });
  });

  async function loadWorkspaces() {
    try {
      workspaces = await ListWorkspaces();
      activeWorkspace = await GetActiveWorkspace();
    } catch (e) {
      console.error("Failed to load workspaces:", e);
    }
  }

  async function handleSwitchWorkspace(name) {
    try {
      await SwitchWorkspace(name);
      activeWorkspace = name;
      allPages = await GetDbPages();
      selectedPage = null;
      navigationHistory = [];
      showWorkspaceDropdown = false;
    } catch (e) {
      alert("Failed to switch workspace: " + e);
    }
  }

  async function handleCreateWorkspace() {
    const name = newWorkspaceName.trim();
    if (!name) return;
    try {
      await CreateWorkspace(name);
      newWorkspaceName = '';
      await loadWorkspaces();
      await handleSwitchWorkspace(name);
    } catch (e) {
      alert("Failed to create workspace: " + e);
    }
  }

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
    if (saveTimeout) {
      clearTimeout(saveTimeout);
      saveCard();
    }

    if (pushToHistory && selectedPage && selectedPage.id !== page.id) {
      navigationHistory = [...navigationHistory, selectedPage.id];
    }

    selectedPage = page;
    editorTitle = page.title;
    editorEmoji = page.emoji;
    showEmojiPicker = false;
    selectedCardId = null;
    pageCards = [];
    activeTab = 'main';
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

  function selectTab(tabId) {
    activeTab = tabId;
    selectedCardId = null;
    pageCards = [];
    if (tabId === 'main' && selectedPage) {
      FetchCards(selectedPage.id);
    } else if (tabId !== 'main') {
      FetchCards(tabId);
    }
  }

  function savePage() {
    if (!selectedPage) return;
    UpdatePage(
      selectedPage.id,
      selectedPage.parent_id || '',
      selectedPage.relation_type || 'subpage',
      editorTitle.trim() || 'Untitled',
      editorEmoji
    ).catch(err => {
      console.error("Save error:", err);
    });
  }

  function savePageDebounced() {
    if (saveTimeout) clearTimeout(saveTimeout);
    saveTimeout = setTimeout(() => {
      savePage();
    }, 500);
  }

  function saveCard() {
    if (!selectedCardId) return;
    UpdateCard(selectedCardId, cardEditorContent).catch(err => {
      console.error("Card save error:", err);
    });
  }

  function saveCardDebounced() {
    if (saveTimeout) clearTimeout(saveTimeout);
    saveTimeout = setTimeout(() => {
      saveCard();
    }, 500);
  }

  function selectCard(card) {
    selectedCardId = card.id;
  }

  function createPage(parentId = '', relationType = 'subpage') {
    AddPage(parentId, 'New Page', '📝', relationType).then(() => {
      if (parentId) {
        expandedPageIds[parentId] = true;
      }
    }).catch(err => {
      alert("Failed to create page: " + err);
    });
  }

  function createSidePage() {
    if (!selectedPage) return;
    createPage(selectedPage.id, 'sidepage');
  }

  async function addCard(type = 'markdown', insertIndex = -1) {
    const targetPageId = activeTab === 'main' ? selectedPage?.id : activeTab;
    if (!targetPageId) return;
    try {
      const sortOrders = pageCards.map(c => c.sort_order);
      const nextOrder = sortOrders.length > 0 ? Math.max(...sortOrders) + 1 : 0;
      await AddCard(targetPageId, type, '', nextOrder);
    } catch (e) {
      alert("Failed to add card: " + e);
    }
  }

  function handleCardDrop(e, toIndex) {
    const fromIndex = parseInt(e.dataTransfer.getData('text/plain'));
    if (isNaN(fromIndex) || fromIndex === toIndex) return;
    const targetPageId = activeTab === 'main' ? selectedPage?.id : activeTab;
    if (!targetPageId) return;
    const reordered = [...pageCards];
    const [moved] = reordered.splice(fromIndex, 1);
    reordered.splice(toIndex > fromIndex ? toIndex - 1 : toIndex, 0, moved);
    const ids = reordered.map(c => c.id);
    ReorderCards(targetPageId, ids).then(() => {
      pageCards = reordered;
    }).catch(err => {
      console.error("Reorder failed:", err);
    });
  }

  function showInsertMenu(index) {
    const types = ['markdown', 'image', 'subpage_link'];
    const labels = ['📝 Markdown', '🖼️ Image', '🔗 Subpage Link'];
    const choice = prompt('Insert card type:\n1: 📝 Markdown\n2: 🖼️ Image\n3: 🔗 Subpage Link');
    if (choice == null) return;
    const idx = parseInt(choice);
    if (idx >= 1 && idx <= types.length) {
      addCard(types[idx - 1]);
    }
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
    const pages = allPages.filter(p => p.id !== id && p.relation_type !== 'sidepage');
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

  function getPagePath(page) {
    if (!page) return [];
    let path = [page];
    let parentId = page.parent_id;
    let depth = 0;
    while (parentId && depth < 20) {
      let parent = allPages.find(p => p.id === parentId);
      if (parent) {
        path.unshift(parent);
        parentId = parent.parent_id;
      } else {
        break;
      }
      depth++;
    }
    return path;
  }

  $: currentPath = getPagePath(selectedPage);
  $: rootPages = allPages.filter(p => !p.parent_id && p.relation_type !== 'sidepage');
  $: getChildrenOf = (parentId) => allPages.filter(p => p.parent_id === parentId && p.relation_type !== 'sidepage');
  $: sidePages = allPages.filter(p => p.parent_id === selectedPage?.id && p.relation_type === 'sidepage');
</script>

<main class="app-layout">
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

    <div class="sidebar-section workspace-section">
      <div class="section-title">
        <span>Workspace</span>
      </div>
      <div class="workspace-current">
        <button class="workspace-selector" on:click={() => showWorkspaceDropdown = !showWorkspaceDropdown}>
          <span class="workspace-name">{activeWorkspace || 'None'}</span>
          <span class="dropdown-arrow">{showWorkspaceDropdown ? '▲' : '▼'}</span>
        </button>
      </div>
      {#if showWorkspaceDropdown}
        <div class="workspace-dropdown">
          {#each workspaces as ws}
            <button class="workspace-option {ws === activeWorkspace ? 'active' : ''}" on:click={() => handleSwitchWorkspace(ws)}>
              {ws}
            </button>
          {/each}
          <div class="workspace-create">
            <input type="text" bind:value={newWorkspaceName} placeholder="New workspace name..." on:keydown={(e) => e.key === 'Enter' && handleCreateWorkspace()} />
            <button class="btn btn-primary" on:click={handleCreateWorkspace}>Create</button>
          </div>
        </div>
      {/if}
    </div>

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
          <div class="tree-list">
            {#each rootPages as page}
              <div class="tree-node-wrapper">
                <svelte:component this={TreeRender} {page} {allPages} {selectedPage} {expandedPageIds} {selectPage} {createPage} {deletePage} {toggleExpand} {getChildrenOf} />
              </div>
            {/each}
          </div>
        {/if}
      </div>
    </div>

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

  <section class="editor-workspace">
    {#if connectionStatus !== 'connected'}
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
      <div class="welcome-container">
        <div class="connected-placeholder">
          <span class="placeholder-emoji">📝</span>
          <h2>Select or Create a Page</h2>
          <p>Choose a journal entry from the sidebar tree, or create a new root note to start writing.</p>
          <button class="btn btn-primary" on:click={() => createPage('')}>Create New Page</button>
        </div>
      </div>
    {:else}
      <div class="editor-header">
        <div class="breadcrumbs">
          {#if navigationHistory.length > 0}
            <button class="btn-back" on:click={goBack} title="Go Back">← Back</button>
            <span class="divider">|</span>
          {/if}
          <span class="breadcrumb-root">{activeWorkspace}</span>
          {#each currentPath as step, i}
            <span class="divider">/</span>
            <span class="breadcrumb-page" class:active-step={i === currentPath.length - 1}>
              {step.emoji} {step.title || 'Untitled'}
            </span>
          {/each}
        </div>

        <div class="header-controls">
          <button class="btn btn-secondary" on:click={() => movePage(selectedPage.id)}>Move To...</button>
          <button class="btn btn-danger" on:click={() => deletePage(selectedPage.id)}>Archive</button>
        </div>
      </div>

      <SidePages
        {sidePages}
        {activeTab}
        onSelectTab={selectTab}
        onCreateSidePage={createSidePage}
        mainPageTitle={editorTitle}
        mainPageEmoji={editorEmoji}
      />

      <div class="card-column">
        {#each pageCards as card, index (card.id)}
          <div class="card-slot"
               draggable="true"
               on:dragstart={(e) => { e.dataTransfer.setData('text/plain', index.toString()); e.dataTransfer.effectAllowed = 'move'; }}
               on:dragover|preventDefault={(e) => { e.dataTransfer.dropEffect = 'move'; }}
               on:drop|preventDefault={(e) => { handleCardDrop(e, index); }}>
            <CardBlock
              {card}
              isSelected={selectedCardId === card.id}
              allPages={allPages}
              onSelect={selectCard}
              onNavigate={selectPage}
              onDeleted={(id) => { pageCards = pageCards.filter(c => c.id !== id); }}
            />
          </div>
          <div class="insert-slot"
               on:dragover|preventDefault={(e) => { e.dataTransfer.dropEffect = 'move'; }}
               on:drop|preventDefault={(e) => { handleCardDrop(e, index + 1); }}>
            <button class="insert-btn" on:click={() => showInsertMenu(index)}>+</button>
          </div>
        {/each}

        {#if pageCards.length === 0}
          <div class="empty-cards-container">
            <span class="empty-icon">📂</span>
            <h3>This page is empty</h3>
            <p>Add block elements to start drafting content, embeds, or linking other pages.</p>
            <div class="empty-actions">
              <button class="btn btn-secondary" on:click={() => addCard('markdown')}>
                📝 Add Markdown Block
              </button>
              <button class="btn btn-secondary" on:click={() => addCard('image')}>
                🖼️ Add Image Block
              </button>
              <button class="btn btn-secondary" on:click={() => addCard('subpage_link')}>
                🔗 Add Subpage Link
              </button>
            </div>
          </div>
        {/if}

        <button class="add-card-btn" on:click={() => addCard('markdown')}>+ Add Card</button>
      </div>
    {/if}
  </section>
</main>

<script context="module">
  import { default as TreeRender } from './TreeRender.svelte';
  import { default as CardBlock } from './CardBlock.svelte';
  import { default as SidePages } from './SidePages.svelte';
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

  .logo-icon { font-size: 20px; }

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

  .add-btn:hover { color: white; }

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

  .manual-fields .row input { width: 60px; }

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

  .editor-workspace {
    flex-grow: 1;
    display: flex;
    flex-direction: column;
    background-color: #121212;
    height: 100%;
  }

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

  .pulse-icon { font-size: 32px; }

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

  .connected-placeholder { text-align: center; color: #6c6c6c; }

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
  .breadcrumb-page { color: #94a3b8; font-weight: 500; }
  .breadcrumb-page.active-step { color: #818cf8; font-weight: 700; }

  .header-controls {
    display: flex;
    align-items: center;
    gap: 16px;
  }

  .empty-cards-container {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 60px 40px;
    text-align: center;
    background-color: #1a1a1a;
    border: 1px dashed #2e2e2e;
    border-radius: 12px;
    margin: 20px 24px;
  }

  .empty-icon {
    font-size: 48px;
    margin-bottom: 16px;
  }

  .empty-cards-container h3 {
    font-size: 18px;
    font-weight: 700;
    margin: 0 0 8px 0;
    color: white;
  }

  .empty-cards-container p {
    font-size: 13px;
    color: #8e8e8e;
    margin: 0 0 24px 0;
    max-width: 380px;
    line-height: 1.5;
  }

  .empty-actions {
    display: flex;
    gap: 12px;
    flex-wrap: wrap;
    justify-content: center;
  }

  .empty-actions .btn {
    font-size: 12px;
    padding: 8px 16px;
  }

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

  .workspace-section { border-bottom: 1px solid #2e2e2e; }
  .workspace-current { padding: 0 12px 8px; }
  .workspace-selector {
    width: 100%;
    display: flex;
    justify-content: space-between;
    align-items: center;
    background: #2a2a2a;
    border: 1px solid #3e3e3e;
    border-radius: 6px;
    padding: 6px 10px;
    color: #e2e8f0;
    cursor: pointer;
    font-size: 12px;
  }
  .workspace-selector:hover { background: #333; }
  .workspace-dropdown { padding: 0 12px 8px; }
  .workspace-option {
    display: block;
    width: 100%;
    text-align: left;
    background: transparent;
    border: none;
    color: #94a3b8;
    padding: 5px 10px;
    font-size: 12px;
    border-radius: 4px;
    cursor: pointer;
  }
  .workspace-option:hover { background: #2a2a2a; color: #e2e8f0; }
  .workspace-option.active { background: #333; color: #818cf8; font-weight: 600; }
  .workspace-create {
    display: flex;
    gap: 4px;
    margin-top: 6px;
  }
  .workspace-create input {
    flex: 1;
    background: #2a2a2a;
    border: 1px solid #3e3e3e;
    border-radius: 4px;
    padding: 4px 8px;
    color: #e2e8f0;
    font-size: 11px;
  }
  .workspace-create .btn { padding: 4px 8px; font-size: 11px; }

  .card-column {
    flex: 1;
    overflow-y: auto;
    padding: 16px 24px;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .add-card-btn {
    background: transparent;
    border: 1px dashed #3e3e3e;
    border-radius: 8px;
    padding: 10px;
    color: #64748b;
    cursor: pointer;
    font-size: 12px;
    transition: all 0.15s ease;
  }
  .add-card-btn:hover { border-color: #818cf8; color: #818cf8; }

  .card-slot { transition: opacity 0.15s ease; }
  .card-slot[draggable="true"] { cursor: grab; }
  .card-slot[draggable="true"]:active { cursor: grabbing; }

  .insert-slot {
    height: 4px;
    margin: 0 24px;
    border-radius: 2px;
    transition: all 0.15s ease;
    display: flex;
    align-items: center;
    justify-content: center;
    position: relative;
  }
  .insert-slot:hover { height: 24px; background: rgba(129,140,248,0.05); }

  .insert-btn {
    opacity: 0;
    background: #2a2a2a;
    border: 1px solid #3e3e3e;
    color: #64748b;
    font-size: 12px;
    width: 20px;
    height: 20px;
    border-radius: 4px;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.15s ease;
  }
  .insert-slot:hover .insert-btn { opacity: 1; }
  .insert-btn:hover { border-color: #818cf8; color: #818cf8; }
</style>
