<script>
  import { onMount } from 'svelte';
  import { EventsOn } from '../wailsjs/runtime/runtime.js';
  import { 
    GetConnectionStatus, GetDbPages, GetDiscoveredDevices, ConnectToDevice, Disconnect, 
    AddPage, UpdatePage, DeletePage, RestorePage, HardDeletePage, MovePage,
    GetCards, FetchCards, AddCard, UpdateCard, DeleteCard, ReorderCards,
    ListWorkspaces, CreateWorkspace, SwitchWorkspace, GetActiveWorkspace
  } from '../wailsjs/go/main/App.js';

  let connectionStatus = 'disconnected';
  let discoveredDevices = [];
  let connectedDevice = null;
  let manualIp = '';
  let manualPort = 9090;
  let manualPin = '';
  let connectionError = '';

  let workspaces = [];
  let activeWorkspace = '';
  let showWorkspaceDropdown = false;
  let newWorkspaceName = '';

  let allPages = [];
  let selectedPage = null;
  let navigationHistory = [];

  let pageCards = [];
  let selectedCardId = null;

  let editorTitle = '';
  let editorEmoji = '📓';
  let saveTimeout;

  let expandedPageIds = {};

  // Sidebar widths (pixels)
  let leftSidebarWidth = 260;
  let rightSidebarWidth = 260;
  let dragging = null; // 'left' | 'right' | null
  let dragStartX = 0;
  let dragStartWidth = 0;

  $: rootPages = allPages.filter(p => !p.parent_id && p.relation_type !== 'sidepage');
  $: sidePages = allPages.filter(p => p.parent_id === selectedPage?.id && p.relation_type === 'sidepage');

  onMount(async () => {
    try {
      connectionStatus = await GetConnectionStatus();
      discoveredDevices = await GetDiscoveredDevices();
      allPages = await GetDbPages();
      await loadWorkspaces();
    } catch (e) { console.error("Init failed:", e); }

    EventsOn('connection-status', (status) => {
      connectionStatus = status;
      if (status === 'disconnected') { connectedDevice = null; allPages = []; selectedPage = null; navigationHistory = []; }
      connectionError = '';
    });
    EventsOn('discovered-devices', (d) => { discoveredDevices = d; });
    EventsOn('db-update', (pages) => {
      allPages = pages;
      if (selectedPage) {
        const updated = pages.find(p => p.id === selectedPage.id);
        if (!updated) { selectedPage = null; navigationHistory = []; }
        else if (updated.updated_at !== selectedPage.updated_at) {
          selectedPage = updated;
          const focused = document.activeElement?.id === 'editor-title-input';
          if (!focused) editorTitle = updated.title;
          editorEmoji = updated.emoji;
        }
      }
    });
    EventsOn('workspace-status', (data) => { activeWorkspace = data.activeWorkspace; workspaces = data.availableWorkspaces || []; });
    EventsOn('cards-update', (data) => {
      const pid = data.pageId || data.page_id;
      if (selectedPage && pid === selectedPage.id) {
        pageCards = data.cards || [];
        if (selectedCardId && !pageCards.find(c => c.id === selectedCardId)) selectedCardId = null;
      }
    });
    EventsOn('pairing-required', () => {});

    window.addEventListener('mousemove', onDragMove);
    window.addEventListener('mouseup', onDragEnd);
    return () => {
      window.removeEventListener('mousemove', onDragMove);
      window.removeEventListener('mouseup', onDragEnd);
    };
  });

  function onDragStart(panel, e) {
    dragging = panel;
    dragStartX = e.clientX;
    dragStartWidth = panel === 'left' ? leftSidebarWidth : rightSidebarWidth;
    e.preventDefault();
  }

  function onDragMove(e) {
    if (!dragging) return;
    const delta = e.clientX - dragStartX;
    if (dragging === 'left') {
      leftSidebarWidth = Math.max(180, Math.min(450, dragStartWidth + delta));
    } else if (dragging === 'right') {
      rightSidebarWidth = Math.max(180, Math.min(450, dragStartWidth - delta));
    }
  }

  function onDragEnd() { dragging = null; }

  async function loadWorkspaces() {
    try { workspaces = await ListWorkspaces(); activeWorkspace = await GetActiveWorkspace(); } catch (e) {}
  }

  async function handleSwitchWorkspace(name) {
    try { await SwitchWorkspace(name); activeWorkspace = name; allPages = await GetDbPages(); selectedPage = null; navigationHistory = []; showWorkspaceDropdown = false; } catch (e) { alert("Failed: " + e); }
  }

  async function handleCreateWorkspace() {
    const name = newWorkspaceName.trim(); if (!name) return;
    try { await CreateWorkspace(name); newWorkspaceName = ''; await loadWorkspaces(); await handleSwitchWorkspace(name); } catch (e) { alert("Failed: " + e); }
  }

  async function handleConnect(device) {
    connectionError = '';
    try {
      connectedDevice = device;
      const pin = prompt('Enter auth PIN from mobile:') || '';
      if (!pin) { connectedDevice = null; return; }
      await ConnectToDevice(device.ip, device.port, pin);
    } catch (err) { connectionError = err.toString(); connectedDevice = null; }
  }

  async function handleConnectManually() {
    if (!manualIp || !manualPin) { connectionError = 'IP and PIN required'; return; }
    connectionError = '';
    try { connectedDevice = { ip: manualIp.trim(), port: parseInt(manualPort) || 9090, deviceName: 'Manual' }; await ConnectToDevice(manualIp.trim(), parseInt(manualPort) || 9090, manualPin.trim()); } catch (err) { connectionError = err.toString(); connectedDevice = null; }
  }

  async function handleDisconnect() {
    try { await Disconnect(); connectedDevice = null; allPages = []; selectedPage = null; navigationHistory = []; } catch (err) {}
  }

  function toggleExpand(pageId) { expandedPageIds[pageId] = !expandedPageIds[pageId]; }

  function selectPage(page, pushToHistory = true) {
    if (saveTimeout) { clearTimeout(saveTimeout); savePageImmediate(); }
    if (pushToHistory && selectedPage && selectedPage.id !== page.id) navigationHistory = [...navigationHistory, selectedPage.id];
    selectedPage = page; editorTitle = page.title; editorEmoji = page.emoji; selectedCardId = null; pageCards = [];
    FetchCards(page.id);
  }

  function goBack() {
    if (navigationHistory.length === 0) return;
    const prevId = navigationHistory[navigationHistory.length - 1];
    navigationHistory = navigationHistory.slice(0, -1);
    const prev = allPages.find(p => p.id === prevId);
    if (prev) selectPage(prev, false);
  }

  function savePageImmediate() {
    if (!selectedPage) return;
    const t = editorTitle.trim() || 'Untitled';
    UpdatePage(selectedPage.id, t, editorEmoji).catch(() => {});
  }

  function savePageDebounced() {
    if (saveTimeout) clearTimeout(saveTimeout);
    saveTimeout = setTimeout(savePageImmediate, 500);
  }

  function selectCard(card) { selectedCardId = card.id; }

  function createPage(parentId = '', relationType = 'subpage') {
    AddPage(parentId, relationType, 'New Page', '📝').then(() => {
      if (parentId) expandedPageIds[parentId] = true;
    }).catch(err => alert("Failed: " + err));
  }

  function createSidePage() {
    if (!selectedPage) return;
    createPage(selectedPage.id, 'sidepage');
  }

  async function addCard(type = 'markdown') {
    if (!selectedPage) return;
    try {
      const nextOrder = pageCards.length > 0 ? Math.max(...pageCards.map(c => c.sort_order)) + 1 : 0;
      await AddCard(selectedPage.id, type, '', nextOrder);
    } catch (e) { alert("Failed: " + e); }
  }

  function handleCardDrop(e, toIndex) {
    const fromIndex = parseInt(e.dataTransfer.getData('text/plain'));
    if (isNaN(fromIndex) || fromIndex === toIndex || !selectedPage) return;
    const reordered = [...pageCards];
    const [moved] = reordered.splice(fromIndex, 1);
    reordered.splice(toIndex > fromIndex ? toIndex - 1 : toIndex, 0, moved);
    ReorderCards(selectedPage.id, reordered.map(c => c.id)).then(() => { pageCards = reordered; }).catch(() => {});
  }

  function showInsertMenu() {
    const choice = prompt('Insert card type:\n1: 📝 Markdown\n2: 🖼️ Image\n3: 🔗 Subpage Link');
    if (!choice) return;
    const types = ['markdown', 'image', 'subpage_link'];
    const idx = parseInt(choice);
    if (idx >= 1 && idx <= 3) addCard(types[idx - 1]);
  }

  function deletePage(id) {
    if (!confirm("Archive this page and all its subpages?")) return;
    DeletePage(id).then(() => { if (selectedPage?.id === id) { selectedPage = null; navigationHistory = []; } }).catch(() => {});
  }

  function movePage(id) {
    const pages = allPages.filter(p => p.id !== id && p.relation_type !== 'sidepage');
    const options = pages.map((p, i) => `${i + 1}: ${p.emoji} ${p.title || 'Untitled'}`);
    options.unshift('0: 📂 Root Level');
    const choice = prompt('Move to:\n' + options.join('\n'));
    if (choice == null) return;
    const idx = parseInt(choice);
    if (idx === 0) MovePage(id, '');
    else if (idx > 0 && idx <= pages.length) MovePage(id, pages[idx - 1].id);
  }

  function getChildrenOf(parentId) { return allPages.filter(p => p.parent_id === parentId && p.relation_type !== 'sidepage'); }
</script>

<main class="app-layout" class:dragging>
  <!-- LEFT SIDEBAR -->
  <aside class="sidebar left-sidebar" style="width: {leftSidebarWidth}px; min-width: {leftSidebarWidth}px;">
    <div class="sidebar-header">
      <div class="logo-section"><span class="logo-icon">📓</span><span class="logo-text">Cero</span></div>
      <div class="status-badge {connectionStatus}"><span class="dot"></span>{connectionStatus}</div>
    </div>

    <div class="sidebar-section">
      <div class="section-label">Workspace</div>
      <button class="workspace-btn" on:click={() => showWorkspaceDropdown = !showWorkspaceDropdown}>
        <span>{activeWorkspace || 'None'}</span>
        <span class="arrow">{showWorkspaceDropdown ? '▲' : '▼'}</span>
      </button>
      {#if showWorkspaceDropdown}
        <div class="dropdown">
          {#each workspaces as ws}
            <button class="opt" class:active={ws === activeWorkspace} on:click={() => handleSwitchWorkspace(ws)}>{ws}</button>
          {/each}
          <div class="create-row">
            <input type="text" bind:value={newWorkspaceName} placeholder="New..." on:keydown={(e) => e.key === 'Enter' && handleCreateWorkspace()} />
            <button class="btn-sm" on:click={handleCreateWorkspace}>+</button>
          </div>
        </div>
      {/if}
    </div>

    <div class="sidebar-section">
      <div class="section-label">Link Devices</div>
      <div class="devices-box">
        {#if connectionStatus === 'connected'}
          <div class="connected-info">
            <span class="conn-label">Connected to</span>
            <span class="conn-name">{connectedDevice?.deviceName || 'Mobile'}</span>
          </div>
          <button class="btn-sm full" on:click={handleDisconnect}>Disconnect</button>
        {:else}
          {#if discoveredDevices.length === 0}
            <div class="no-dev">Scanning WiFi...</div>
          {:else}
            {#each discoveredDevices as d}
              <div class="dev-row">
                <div><span class="dev-name">{d.deviceName}</span><span class="dev-ip">{d.ip}:{d.port}</span></div>
                <button class="btn-sm primary" on:click={() => handleConnect(d)}>Link</button>
              </div>
            {/each}
          {/if}
        {/if}
      </div>
    </div>

    <div class="sidebar-section grow">
      <div class="section-label">
        <span>Pages</span>
        {#if connectionStatus === 'connected'}
          <button class="icon-btn" on:click={() => createPage('')}>+</button>
        {/if}
      </div>
      <div class="tree-scroll">
        {#if connectionStatus !== 'connected'}
          <div class="tree-empty">Connect to phone to view pages.</div>
        {:else if rootPages.length === 0}
          <div class="tree-empty">No pages yet.</div>
        {:else}
          {#each rootPages as page}
            <svelte:component this={TreeRender} {page} {allPages} {selectedPage} {expandedPageIds} {selectPage} {createPage} {deletePage} {toggleExpand} {getChildrenOf} />
          {/each}
        {/if}
      </div>
    </div>

    {#if connectionStatus !== 'connected'}
      <div class="sidebar-section manual">
        <div class="section-label">Manual Link</div>
        <input type="text" bind:value={manualIp} placeholder="IP Address" />
        <input type="text" bind:value={manualPin} placeholder="Auth PIN" maxlength="4" class="pin" />
        <div class="manual-row">
          <input type="number" bind:value={manualPort} placeholder="9090" />
          <button class="btn-sm primary" on:click={handleConnectManually}>Link</button>
        </div>
        {#if connectionError}<div class="err">{connectionError}</div>{/if}
      </div>
    {/if}
  </aside>

  <!-- LEFT DRAG HANDLE -->
  <div class="drag-handle left-handle" on:mousedown={(e) => onDragStart('left', e)} class:active={dragging === 'left'}>
    <div class="handle-line"></div>
  </div>

  <!-- EDITOR WORKSPACE -->
  <section class="editor-workspace">
    {#if connectionStatus !== 'connected'}
      <div class="welcome"><div class="welcome-card">
        <div class="pulse"><span>📓</span></div>
        <h1>Cero Journal</h1>
        <p>Connect to your mobile phone to sync notes in real time.</p>
        <div class="steps">
          <div class="step"><span class="num">1</span><p>Start <strong>Cero</strong> on phone</p></div>
          <div class="step"><span class="num">2</span><p>Enable <strong>Sync Hub</strong></p></div>
          <div class="step"><span class="num">3</span><p>Link device on left</p></div>
        </div>
      </div></div>
    {:else if !selectedPage}
      <div class="welcome"><div class="empty-page">
        <span class="big-icon">📝</span>
        <h2>Select or Create a Page</h2>
        <p>Choose from sidebar or create a new page.</p>
        <button class="btn primary" on:click={() => createPage('')}>Create New Page</button>
      </div></div>
    {:else}
      <div class="editor-header">
        <div class="breadcrumbs">
          {#if navigationHistory.length > 0}
            <button class="back-btn" on:click={goBack}>← Back</button>
            <span class="sep">|</span>
          {/if}
          <span class="bc-root">{activeWorkspace}</span>
          <span class="sep">/</span>
          <span class="bc-current">{selectedPage.emoji} {selectedPage.title || 'Untitled'}</span>
        </div>
        <div class="header-actions">
          <button class="btn-sm" on:click={() => movePage(selectedPage.id)}>Move</button>
          <button class="btn-sm danger" on:click={() => deletePage(selectedPage.id)}>Archive</button>
        </div>
      </div>

      <div class="card-column">
        {#each pageCards as card, index (card.id)}
          <div class="card-slot" draggable="true"
               on:dragstart={(e) => { e.dataTransfer.setData('text/plain', index.toString()); e.dataTransfer.effectAllowed = 'move'; }}
               on:dragover|preventDefault={(e) => { e.dataTransfer.dropEffect = 'move'; }}
               on:drop|preventDefault={(e) => handleCardDrop(e, index)}>
            <CardBlock {card} isSelected={selectedCardId === card.id} allPages={allPages}
              onSelect={selectCard} onNavigate={selectPage} onDeleted={(id) => { pageCards = pageCards.filter(c => c.id !== id); }} />
          </div>
          <div class="insert-slot" on:dragover|preventDefault on:drop|preventDefault={(e) => handleCardDrop(e, index + 1)}>
            <button class="insert-btn" on:click={showInsertMenu}>+</button>
          </div>
        {/each}

        {#if pageCards.length === 0}
          <div class="empty-cards">
            <span class="empty-icon">📂</span>
            <h3>This page is empty</h3>
            <p>Add blocks to start writing.</p>
            <div class="empty-actions">
              <button class="btn secondary" on:click={() => addCard('markdown')}>📝 Markdown</button>
              <button class="btn secondary" on:click={() => addCard('image')}>🖼️ Image</button>
              <button class="btn secondary" on:click={() => addCard('subpage_link')}>🔗 Link</button>
            </div>
          </div>
        {/if}

        <button class="add-card-btn" on:click={() => addCard('markdown')}>+ Add Card</button>
      </div>
    {/if}
  </section>

  <!-- RIGHT DRAG HANDLE -->
  <div class="drag-handle right-handle" on:mousedown={(e) => onDragStart('right', e)} class:active={dragging === 'right'}>
    <div class="handle-line"></div>
  </div>

  <!-- RIGHT SIDEBAR -->
  <aside class="sidebar right-sidebar" style="width: {rightSidebarWidth}px; min-width: {rightSidebarWidth}px;">
    <RightSidebar {sidePages} {selectedPage} onSelectPage={selectPage} onCreateSidePage={createSidePage} onDeletePage={deletePage} />
  </aside>
</main>

<script context="module">
  import { default as TreeRender } from './TreeRender.svelte';
  import { default as CardBlock } from './CardBlock.svelte';
  import { default as RightSidebar } from './RightSidebar.svelte';
</script>

<style>
  .app-layout {
    display: flex;
    width: 100vw;
    height: 100vh;
    overflow: hidden;
    background-color: #121212;
    color: #e2e8f0;
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
  }
  .app-layout.dragging { cursor: col-resize; user-select: none; }

  .sidebar {
    display: flex;
    flex-direction: column;
    height: 100%;
    overflow: hidden;
    flex-shrink: 0;
  }
  .left-sidebar { background: #1a1a1a; border-right: 1px solid #2e2e2e; }
  .right-sidebar { background: #1a1a1a; border-left: 1px solid #2e2e2e; }

  /* Drag handles */
  .drag-handle {
    width: 5px;
    cursor: col-resize;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    background: transparent;
    transition: background 0.15s;
    z-index: 20;
  }
  .drag-handle:hover, .drag-handle.active { background: rgba(129,140,248,0.15); }
  .handle-line {
    width: 2px;
    height: 24px;
    border-radius: 1px;
    background: #3e3e3e;
    transition: background 0.15s;
  }
  .drag-handle:hover .handle-line, .drag-handle.active .handle-line { background: #818cf8; }

  .sidebar-header {
    padding: 14px 16px;
    border-bottom: 1px solid #2e2e2e;
    display: flex;
    justify-content: space-between;
    align-items: center;
    flex-shrink: 0;
  }
  .logo-section { display: flex; align-items: center; gap: 8px; font-weight: 700; }
  .logo-icon { font-size: 18px; }
  .logo-text { font-size: 14px; background: linear-gradient(135deg, #a5b4fc, #c084fc); -webkit-background-clip: text; -webkit-text-fill-color: transparent; }

  .status-badge {
    display: flex; align-items: center; gap: 5px;
    padding: 2px 8px; border-radius: 10px;
    font-size: 9px; font-weight: 700; text-transform: uppercase;
  }
  .status-badge.disconnected { background: rgba(239,68,68,0.1); color: #f87171; }
  .status-badge.connected { background: rgba(34,197,94,0.1); color: #4ade80; }
  .status-badge.connecting { background: rgba(251,191,36,0.1); color: #fbbf24; }
  .status-badge.reconnecting { background: rgba(251,191,36,0.1); color: #fbbf24; }
  .dot { width: 5px; height: 5px; border-radius: 50%; }
  .disconnected .dot { background: #f87171; }
  .connected .dot { background: #4ade80; }
  .connecting .dot, .reconnecting .dot { background: #fbbf24; animation: pulse 1s infinite; }
  @keyframes pulse { 0%, 100% { opacity: 1; } 50% { opacity: 0.3; } }

  .sidebar-section { padding: 12px 14px; border-bottom: 1px solid #2e2e2e; }
  .sidebar-section.grow { flex: 1; overflow: hidden; display: flex; flex-direction: column; }
  .sidebar-section.manual { margin-top: auto; }

  .section-label {
    font-size: 10px; font-weight: 700; text-transform: uppercase;
    letter-spacing: 0.8px; color: #8e8e8e; margin-bottom: 8px;
    display: flex; justify-content: space-between; align-items: center;
  }

  .workspace-btn {
    width: 100%; display: flex; justify-content: space-between; align-items: center;
    background: #2a2a2a; border: 1px solid #3e3e3e; border-radius: 6px;
    padding: 6px 10px; color: #e2e8f0; cursor: pointer; font-size: 12px;
  }
  .workspace-btn:hover { background: #333; }
  .arrow { font-size: 10px; color: #8e8e8e; }

  .dropdown { margin-top: 6px; }
  .opt {
    display: block; width: 100%; text-align: left; background: transparent;
    border: none; color: #94a3b8; padding: 5px 10px; font-size: 12px;
    border-radius: 4px; cursor: pointer;
  }
  .opt:hover { background: #2a2a2a; color: #e2e8f0; }
  .opt.active { color: #818cf8; font-weight: 600; }
  .create-row { display: flex; gap: 4px; margin-top: 6px; }
  .create-row input { flex: 1; background: #2a2a2a; border: 1px solid #3e3e3e; border-radius: 4px; padding: 4px 8px; color: #e2e8f0; font-size: 11px; }

  .devices-box {
    background: #121212; border: 1px solid #2e2e2e; border-radius: 8px; padding: 10px;
  }
  .connected-info { display: flex; justify-content: space-between; font-size: 11px; margin-bottom: 6px; }
  .conn-label { color: #8e8e8e; }
  .conn-name { color: #4ade80; font-weight: 700; }
  .no-dev { font-size: 11px; color: #6c6c6c; text-align: center; }
  .dev-row { display: flex; justify-content: space-between; align-items: center; margin-bottom: 6px; }
  .dev-name { font-size: 12px; font-weight: 700; display: block; }
  .dev-ip { font-size: 10px; color: #6c6c6c; font-family: monospace; }

  .tree-scroll { flex: 1; overflow-y: auto; }
  .tree-empty { font-size: 12px; color: #6c6c6c; padding: 8px 0; }

  .icon-btn { background: transparent; border: none; color: #8e8e8e; cursor: pointer; font-size: 16px; padding: 0 4px; }
  .icon-btn:hover { color: white; }

  .manual input { width: 100%; background: #121212; border: 1px solid #2e2e2e; border-radius: 6px; color: white; font-size: 11px; padding: 6px 10px; outline: none; margin-bottom: 6px; }
  .pin { letter-spacing: 4px; text-align: center; font-weight: bold; }
  .manual-row { display: flex; gap: 6px; }
  .manual-row input { width: 60px; margin-bottom: 0; }
  .manual-row button { flex: 1; }
  .err { color: #f87171; font-size: 10px; margin-top: 4px; }

  /* Buttons */
  .btn-sm {
    border-radius: 6px; font-size: 11px; font-weight: 600; padding: 5px 10px;
    cursor: pointer; border: none; background: #2e2e2e; color: #cbd5e1;
    transition: all 0.15s;
  }
  .btn-sm:hover { background: #3e3e3e; }
  .btn-sm.primary { background: #818cf8; color: white; }
  .btn-sm.primary:hover { background: #6366f1; }
  .btn-sm.danger { background: rgba(239,68,68,0.1); color: #f87171; }
  .btn-sm.full { width: 100%; }

  .btn {
    border-radius: 6px; font-size: 12px; font-weight: 600; padding: 8px 16px;
    cursor: pointer; border: none; transition: all 0.15s;
  }
  .btn.primary { background: #818cf8; color: white; }
  .btn.primary:hover { background: #6366f1; }
  .btn.secondary { background: #2e2e2e; color: #cbd5e1; }
  .btn.secondary:hover { background: #3e3e3e; }

  /* Editor */
  .editor-workspace { flex: 1; display: flex; flex-direction: column; height: 100%; overflow: hidden; background: #121212; }

  .welcome { flex: 1; display: flex; align-items: center; justify-content: center; padding: 40px; }
  .welcome-card { max-width: 500px; text-align: center; background: #1a1a1a; border: 1px solid #2e2e2e; border-radius: 20px; padding: 40px; }
  .pulse { width: 64px; height: 64px; border-radius: 50%; background: rgba(165,180,252,0.05); border: 1px solid rgba(165,180,252,0.15); display: flex; align-items: center; justify-content: center; margin: 0 auto 16px; font-size: 28px; }
  .welcome-card h1 { font-size: 22px; font-weight: 800; margin: 0 0 8px; }
  .welcome-card p { font-size: 13px; color: #8e8e8e; margin: 0 0 24px; }
  .steps { display: grid; grid-template-columns: repeat(3, 1fr); gap: 12px; text-align: left; }
  .step { background: #121212; border: 1px solid #2e2e2e; border-radius: 10px; padding: 14px; }
  .step .num { width: 20px; height: 20px; border-radius: 50%; background: #818cf8; color: white; font-weight: 700; font-size: 10px; display: inline-flex; align-items: center; justify-content: center; margin-bottom: 6px; }
  .step p { font-size: 11px; color: #d1d5db; margin: 0; line-height: 1.4; }

  .empty-page { text-align: center; }
  .big-icon { font-size: 48px; display: block; margin-bottom: 12px; }
  .empty-page h2 { font-size: 18px; margin: 0 0 6px; }
  .empty-page p { font-size: 13px; color: #8e8e8e; margin: 0 0 20px; }

  .editor-header {
    background: #1a1a1a; border-bottom: 1px solid #2e2e2e;
    padding: 10px 24px; display: flex; justify-content: space-between; align-items: center;
    flex-shrink: 0;
  }
  .breadcrumbs { display: flex; align-items: center; gap: 6px; font-size: 12px; }
  .back-btn { background: transparent; border: none; color: #8e8e8e; cursor: pointer; font-size: 11px; font-weight: 600; padding: 3px 6px; border-radius: 4px; }
  .back-btn:hover { background: #2e2e2e; color: white; }
  .sep { color: #444; }
  .bc-root { color: #8e8e8e; }
  .bc-current { color: #818cf8; font-weight: 600; }
  .header-actions { display: flex; gap: 8px; }

  .card-column { flex: 1; overflow-y: auto; padding: 16px 24px; display: flex; flex-direction: column; gap: 10px; }

  .empty-cards {
    display: flex; flex-direction: column; align-items: center; justify-content: center;
    padding: 60px 40px; text-align: center;
    background: #1a1a1a; border: 1px dashed #2e2e2e; border-radius: 12px; margin: 16px 0;
  }
  .empty-cards .empty-icon { font-size: 48px; margin-bottom: 12px; }
  .empty-cards h3 { font-size: 16px; font-weight: 700; margin: 0 0 6px; }
  .empty-cards p { font-size: 13px; color: #8e8e8e; margin: 0 0 20px; }
  .empty-actions { display: flex; gap: 10px; }

  .add-card-btn {
    background: transparent; border: 1px dashed #3e3e3e; border-radius: 8px;
    padding: 10px; color: #64748b; cursor: pointer; font-size: 12px;
    transition: all 0.15s;
  }
  .add-card-btn:hover { border-color: #818cf8; color: #818cf8; }

  .card-slot { transition: opacity 0.15s; }
  .card-slot[draggable="true"] { cursor: grab; }
  .card-slot[draggable="true"]:active { cursor: grabbing; }

  .insert-slot { height: 4px; margin: 0 24px; display: flex; align-items: center; justify-content: center; transition: all 0.15s; }
  .insert-slot:hover { height: 24px; background: rgba(129,140,248,0.05); }
  .insert-btn {
    opacity: 0; background: #2a2a2a; border: 1px solid #3e3e3e; color: #64748b;
    font-size: 12px; width: 20px; height: 20px; border-radius: 4px; cursor: pointer;
    display: flex; align-items: center; justify-content: center; transition: all 0.15s;
  }
  .insert-slot:hover .insert-btn { opacity: 1; }
  .insert-btn:hover { border-color: #818cf8; color: #818cf8; }
</style>
