<script>
  import { onMount } from 'svelte';
  import { EventsOn } from '../wailsjs/runtime/runtime.js';
  import { 
    GetConnectionStatus, GetDbPages, GetDiscoveredDevices, ConnectToDevice, Disconnect, 
    AddPage, UpdatePage, DeletePage, RestorePage, HardDeletePage, MovePage,
    GetCards, FetchCards, AddCard, UpdateCard, DeleteCard, ReorderCards
  } from '../wailsjs/go/main/App.js';

  let connectionStatus = 'disconnected';
  let discoveredDevices = [];
  let connectedDevice = null;
  let manualIp = '';
  let manualPort = 9090;
  let manualPin = '';
  let connectionError = '';

  let activeWorkspace = '';

  let allPages = [];
  let selectedPage = null;
  let navigationHistory = [];

  let pageCards = [];
  let selectedCardId = null;

  let editorTitle = '';
  let editorEmoji = '📓';
  let saveTimeout;

  let expandedPageIds = {};

  // Custom Modal States
  let showBlockSelectorModal = false;
  let blockInsertIndex = null;

  let showEmojiPickerModal = false;
  let selectedCategory = 'smileys';

  // Rich Keyboard-Matching Emojis Catalog grouped by category
  const emojiCategories = [
    { id: 'smileys', label: '😀 Smileys & People', emojis: ['😀', '😃', '😄', '😁', '😆', '😅', '😂', '🤣', '😊', '😇', '🙂', '🙃', '😉', '😌', '😍', '🥰', '😘', '😗', '😙', '😚', '😋', '😛', '😝', '😜', '🤪', '🤨', '🧐', '🤓', '😎', '🥸', '🤩', '🥳', '😏', '😒', '😞', '😔', '😟', '😕', '🙁', '☹️', '😣', '😖', '😫', '😩', '🥺', '😢', '😭', '😤', '😠', '😡', '🤬', '🤯', '😳', '🥵', '🥶', '😱', '😨', '😰', '😥', '😓', '🤔', '🫣', '🤭', '🫢', '🫡', '🤫', '🫠', '🤥', '😶', '😐', '😑', '😬', '🫨', '😴', '🤤', '😪', '😵', '😵💫', '🤐', '🥴', '🤢', '🤮', '🤧', '😷', '🤒', '🤕', '😈', '👿', '👹', '👺', '💀', '☠️', '👻', '👽', '👾', '🤖', '🎃', '😺', '😸', '😹', '😻', '😼', '😽', '😾', '😿', '🙀', '👋', '🤚', '🖐️', '✋', '🖖', '👌', '🤌', '🤏', '✌️', '🤞', '🫰', '🤟', '🤘', '🤙', '👈', '👉', '👆', '🖕', '👇', '☝️', '👍', '👎', '✊', '👊', '🤛', '🤜', '👏', '🙌', '🫶', '👐', '🤲', '🤝', '🙏', '✍️', '💅', '🤳', '💪', '🦾', '🦿', '🦵', '🦶', '👂', '🦻', '👃', '🧠', '🫀', '🫁', '🦷', '🦴', '👀', '👁️', '👅', '👄', '💋', '🩸', '👤', '👥', '🫂'] },
    { id: 'animals', label: '🐱 Animals & Nature', emojis: ['🐶', '🐱', '🐭', '🐹', '🐰', '🦊', '🐻', '🐼', '🐨', '🐯', '🦁', '🐮', '🐷', '🐽', '🐸', '🐵', '🙈', '🙉', '🙊', '🐒', '🐔', '🐧', '🐦', '🐤', '🐣', '🐥', '🦆', '🦅', '🦉', '🪱', '🐛', '🦋', '🐌', '🐞', '🐜', '🪰', '🪲', '🪳', '🦗', '🕷️', '🕸️', 'Scorpion', '🐢', '🐍', '🦎', '🐙', '🦑', '🦞', '🦀', '🐡', '🐠', '🐟', '🐬', '🐳', '🐋', '🦈', '🐊', '🐅', '🐆', '🦓', '🦍', '🦧', '🦣', '🐘', '🦛', '🦏', '🐪', '🐫', '🦒', '🦘', '🦬', '🐃', '🐂', '🐄', '🐎', '🐖', '🐏', '🐑', '🦙', '🐐', '🦌', '🐕', '🐩', '🦮', '🐈', '🐓', '🦃', '🦚', '🦜', '🕊️', '🐇', '🦝', '🦨', '🦡', '🦦', '🦥', '🐿️', '🦔', '🐾', '🐉', '🐲', '🌵', '🎄', '🌲', '🌳', '🌴', '🪵', '🌱', '🌿', '☘️', '🍀', '🍁', '🍂', '🍃', '🍄', '🐚', '🪸', '🪨', '🌾', '💐', '🌷', '🌹', '🥀', '🌺', '🌸', '🌼', '🌻', '🌞', '🌝', '🌛', '🌜', '🌚', '🌕', '🌖', '🌗', '🌘', '🌑', '🌒', '🌓', '🌔', '🌙', '🌎', '🌍', '🌏', '🪐', '💫', '⭐️', '🌟', '✨', '⚡️', '☄️', '💥', '🔥', '🌪️', '🌈', '☀️', '🌤️', '⛅️', '🌥️', '🌦️', '☁️', '🌧️', '⛈️', '🌩️', '🌨️', '❄️', '☃️', '⛄️', '🌬️', '💨', '💧', '💦', '🫧', '☔️', '🌊', '🌫️'] },
    { id: 'food', label: '🍏 Food & Drink', emojis: ['🍏', '🍎', '🍐', '🍊', '🍋', '🍌', '🍉', '🍇', '🍓', '🫐', '🍈', '🍒', '🍑', '🥭', '🍍', '🥥', '🥝', '🍅', '🍆', '🥑', '🥦', '🥬', '🥒', '🌶️', '🫑', '🌽', '🥕', '🫒', '🧄', '🧅', '🥔', '🍠', '🥐', '🍞', '🥖', '🥨', '🧀', '🍳', '🥞', '🥓', '🥩', '🍗', '🍖', '🌭', '🍔', '🍟', '🍕', '🥪', '🥙', '🫓', '🌮', '🌯', '🫔', '🥗', '🥘', '🍲', '🫕', '🥫', '🍝', '🍜', '🍛', '🍣', '🍱', '🥟', '🍤', '🍙', '🍘', '🍥', '🥠', '🥮', '🍢', '🍡', '🍧', '🍨', '🍦', '🥧', '🍰', '🎂', '🧁', '🍮', '🍭', '🍬', '🍫', '🍿', '🍩', '🍪', '🌰', '🥜', '🫘', '🍯', '🥛', '🍼', '☕️', '🍵', '🧃', '🥤', '🧋', '🍶', '🍺', '🍻', '🥂', '🍷', '🥃', '🍸', '🍹', '🧉', '🍾', '🧊', '🥢', '🍽️', '🍴', '🥄'] },
    { id: 'activity', label: '⚽️ Activity & Travel', emojis: ['⚽️', '🏀', '🏈', '⚾️', '🥎', '🎾', '🏐', '🏉', '🥏', '🎱', '🪀', '🏸', '🏒', '🥍', '🏹', '🤿', '🥊', '🥋', '🥅', '⛳️', '⛸️', '🎽', '🎿', '🛷', '🥌', '🎯', '🪗', '🪘', '🎮', '🕹️', '🎰', '🎲', '🧩', '🧸', '🪅', '🪩', '🎨', '🖼️', '🧵', '🪡', '🧶', '🎸', '🎹', '🎺', '🎻', '🥁', '🪕', '🎧', '🎤', '🎬', '🎟️', '🎫', '🎭', '🎪', '🧗', '🏋️', '🚴', '🏃', '🚶', '🚗', '🚕', '🚙', '🚌', '🚎', '🏎️', '🚓', '🚑', '🚒', '🚐', '🛻', '🚚', '🚛', '🚜', '🛵', '🏍️', '🛺', '🚲', '🛴', 'skateboard', '🛼', '🚏', '🛣️', '🛤️', '🚢', '⛵️', '🚤', '🛥️', '🛳️', '⛴️', '🛶', '🛸', '🚁', '🛩️', '✈️', '🛫', '🛬', '🚀', '🛰️', '⚓️', '🗺️', '🧭', '🏔️', '⛰️', '🌋', '🗻', '🏕️', '🏖️', '🏜️', '🏝️', '🏞️', '🏛️', '🏗️', '🧱', '🏘️', '🏚️', '🏠', '🏡', '🏢', '🏣', '🏤', '🏥', '🏦', '🏨', '🏩', '🏪', '🏫', '🏬', '🏭', '🏯', '🏰', '💒', '🗼', '🗽', '🕌', '⛪️', '🛕', '🕍', '⛩️', '🕋', '⛲️', '⛺️', '🌁', '🌃', '🏙️', '🌅', '🌄', '🌇', '🌆', '🌉', '🎠', '🎡', '🎢', '💈'] },
    { id: 'objects', label: '💡 Objects & Symbols', emojis: ['⌚️', '📱', '📲', '💻', '⌨️', '🖥️', '🖨️', '🖱️', '🖲️', '💽', '💾', '💿', '📀', '📼', '📷', '📸', '📹', '🎥', '📽️', '🎞️', '📞', '☎️', '📟', '📠', '📺', '📻', '🎙️', '🎚️', '🎛️', '🧭', '⏰', '⌛️', '⏳', '🔋', '🔌', '💡', '🕯️', '🪔', '🗑️', '🛢️', '💸', '💵', '💴', '💶', '💷', '🪙', '💰', '💳', '💎', '⚖️', '🪜', '🔧', '🔨', '⚒️', '🛠️', '⛏️', '🪚', '🔩', '⚙️', '🧱', '⛓️', '🧲', '🧯', '🔫', '💣', '🧨', '🪓', '🔪', '🗡️', '⚔️', '🛡️', '🚬', '⚰️', '⚱️', '🏺', '🔮', '📿', '🧿', '💈', '🧫', '🧪', '🔬', '🔭', '📡', '💉', '💊', '🩹', '🩺', '🚪', '🛗', '🪞', '🪟', '🛏️', '🛋️', '🪑', '🚽', '🪠', '🚿', '🛁', '🧼', '🪥', '🪮', '🧴', '🧹', '🧺', '🧻', '🪣', '🪟', '🗝️', '🔑', '🪤', '📦', '🏷️', '✉️', '📩', '📨', '📧', '📤', '📥', '📪', '📫', '📬', '📭', '📮', '🗳️', '✏️', '✒️', '🖋️', '🖊️', '🖌️', '🖍️', '📝', '📁', '📂', '🗂️', '📅', '📆', '🗒️', '🗓️', '🪪', '🗃️', '🗄️', '📋', '📌', '📍', '📎', '🖇️', '📏', '📐', '🧮', '🔐', '🔏', '🔒', '🔓', '❤️', '🧡', '💛', '💚', '💙', '💜', '🖤', '🤍', '🤎', '💔', '❣️', '💕', '💞', '💓', '💗', '💖', '💘', '💝', '💟', '☮️', '✝️', '☪️', '🕉️', '☸️', '✡️', '🔯', '🕎', '☯️', '☦️', '🛐', '♈️', '♉️', '♊️', '♋️', '♌️', '♍️', '♎️', '♏️', '♐️', '♑️', '♒️', '♓️', '🆔', '📯', '🔔', '🔕', '📣', '📢', '💬', '💭', '🗯️', '🏁', '🚩', '🎌', '🏴', '🏳️', '🏳️🌈', '🏴☠️'] }
  ];

  // Sidebar widths (pixels)
  let leftSidebarWidth = 260;
  let rightSidebarWidth = 260;
  let dragging = null; // 'left' | 'right' | null
  let dragStartX = 0;
  let dragStartWidth = 0;

  $: rootPages = allPages.filter(p => !p.parent_id && p.relation_type !== 'sidepage');
  $: sidePages = allPages.filter(p => p.parent_id === selectedPage?.id && p.relation_type === 'sidepage');

  $: currentCategoryEmojis = (() => {
    const cat = emojiCategories.find(c => c.id === selectedCategory);
    return cat ? cat.emojis : [];
  })();

  onMount(async () => {
    try {
      connectionStatus = await GetConnectionStatus();
      discoveredDevices = await GetDiscoveredDevices();
      allPages = await GetDbPages();
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
    EventsOn('workspace-status', (data) => { activeWorkspace = data.activeWorkspace; });
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

  // We should declare selectPage as active since we use TreeRender component
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

  // Update page details immediately or debounced
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

  async function handleAddCardType(type) {
    showBlockSelectorModal = false;
    if (!selectedPage) return;
    try {
      let sortOrder = 0;
      if (blockInsertIndex !== null) {
        sortOrder = blockInsertIndex;
      } else {
        sortOrder = pageCards.length > 0 ? Math.max(...pageCards.map(c => c.sort_order)) + 1 : 0;
      }
      await AddCard(selectedPage.id, type, '', sortOrder);
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

    <!-- Read-only Active Workspace Details -->
    <div class="sidebar-section active-workspace-card">
      <div class="section-label">Active Workspace</div>
      <div class="active-ws-display">
        <span class="ws-icon">🗄️</span>
        <span class="ws-name">{activeWorkspace || 'Personal'}</span>
      </div>
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

    <!-- Improved Manual Link Section -->
    {#if connectionStatus !== 'connected'}
      <div class="sidebar-section manual-connect-card">
        <div class="section-label">Manual Link</div>
        <div class="manual-form">
          <div class="form-row">
            <div class="form-group flex-2">
              <label for="manual-ip">IP Address</label>
              <input type="text" id="manual-ip" bind:value={manualIp} placeholder="192.168.1.X" />
            </div>
            <div class="form-group flex-1">
              <label for="manual-port">Port</label>
              <input type="number" id="manual-port" bind:value={manualPort} placeholder="9090" />
            </div>
          </div>
          <div class="form-group">
            <label for="manual-pin">Auth PIN (from mobile)</label>
            <input type="text" id="manual-pin" bind:value={manualPin} placeholder="••••" maxlength="4" class="pin-input" />
          </div>
          <button class="connect-btn" on:click={handleConnectManually}>
            <span class="btn-icon">🔗</span> Link Device
          </button>
          {#if connectionError}
            <div class="connection-error-box">
              <span class="err-icon">⚠️</span>
              <span class="err-msg">{connectionError}</span>
            </div>
          {/if}
        </div>
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

      <div class="title-bar">
        <!-- Interactive Page Emoji Picker Trigger -->
        <button class="emoji-input-btn" on:click={() => showEmojiPickerModal = true}>
          {editorEmoji || '📓'}
        </button>
        <input id="editor-title-input" class="title-input" type="text" bind:value={editorTitle}
               on:input={savePageDebounced} placeholder="Untitled" />
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
            <button class="insert-btn" on:click={() => { blockInsertIndex = index + 1; showBlockSelectorModal = true; }}>+</button>
          </div>
        {/each}

        {#if pageCards.length === 0}
          <div class="empty-cards">
            <span class="empty-icon">📂</span>
            <h3>This page is empty</h3>
            <p>Add blocks to start writing.</p>
            <div class="empty-actions">
              <button class="btn secondary" on:click={() => handleAddCardType('markdown')}>📝 Markdown</button>
              <button class="btn secondary" on:click={() => handleAddCardType('image')}>🖼️ Image</button>
              <button class="btn secondary" on:click={() => handleAddCardType('subpage_link')}>🔗 Link</button>
            </div>
          </div>
        {/if}

        <button class="add-card-btn" on:click={() => { blockInsertIndex = null; showBlockSelectorModal = true; }}>+ Add Card</button>
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

<!-- Block Selector Modal -->
{#if showBlockSelectorModal}
  <div class="modal-backdrop" on:click|self={() => showBlockSelectorModal = false} role="button" tabindex="-1">
    <div class="modal-container block-selector-modal">
      <div class="modal-header">
        <h3>Add Block</h3>
        <button class="close-btn" on:click={() => showBlockSelectorModal = false}>&times;</button>
      </div>
      <div class="modal-body block-options-grid">
        <button class="block-option-row" on:click={() => handleAddCardType('markdown')}>
          <span class="block-icon">📝</span>
          <div class="block-desc">
            <span class="block-title">Markdown</span>
            <span class="block-subtitle">Write formatted text, headers, checklist items, or code blocks.</span>
          </div>
        </button>
        <button class="block-option-row" on:click={() => handleAddCardType('image')}>
          <span class="block-icon">🖼️</span>
          <div class="block-desc">
            <span class="block-title">Image</span>
            <span class="block-subtitle">Embed an image from a web URL or upload directly from your computer.</span>
          </div>
        </button>
        <button class="block-option-row" on:click={() => handleAddCardType('subpage_link')}>
          <span class="block-icon">🔗</span>
          <div class="block-desc">
            <span class="block-title">Subpage Link</span>
            <span class="block-subtitle">Link directly to another nested page in your journal hierarchy.</span>
          </div>
        </button>
      </div>
    </div>
  </div>
{/if}

<!-- Keyboard Emoji Picker Modal -->
{#if showEmojiPickerModal}
  <div class="modal-backdrop" on:click|self={() => showEmojiPickerModal = false} role="button" tabindex="-1">
    <div class="modal-container emoji-picker-modal">
      <div class="modal-header">
        <h3>Select Icon</h3>
        <button class="close-btn" on:click={() => showEmojiPickerModal = false}>&times;</button>
      </div>
      <div class="emoji-picker-tabs">
        {#each emojiCategories as category}
          <button class="emoji-tab-btn" class:active={selectedCategory === category.id} on:click={() => selectedCategory = category.id}>
            {category.label.split(' ')[0]}
          </button>
        {/each}
      </div>
      <div class="modal-body emoji-picker-body">
        <div class="emoji-grid">
          {#each currentCategoryEmojis as emoji}
            <button class="emoji-select-btn" on:click={() => { editorEmoji = emoji; showEmojiPickerModal = false; savePageImmediate(); }}>
              {emoji}
            </button>
          {/each}
        </div>
      </div>
    </div>
  </div>
{/if}

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
  .sidebar-section.grow { flex: 1; overflow-y: auto; display: flex; flex-direction: column; }

  .section-label {
    font-size: 10px; font-weight: 700; text-transform: uppercase;
    letter-spacing: 0.8px; color: #8e8e8e; margin-bottom: 8px;
    display: flex; justify-content: space-between; align-items: center;
  }

  /* Read-Only Active Workspace Display */
  .active-workspace-card {
    background: transparent;
    padding: 10px 14px;
    border-bottom: 1px solid #2e2e2e;
  }
  .active-ws-display {
    display: flex;
    align-items: center;
    gap: 8px;
    background: #121212;
    border: 1px solid #2e2e2e;
    border-radius: 6px;
    padding: 6px 10px;
    color: #cbd5e1;
    font-size: 12px;
    font-weight: 600;
  }
  .ws-icon {
    font-size: 14px;
  }
  .ws-name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

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

  /* New Manual Connect Form UI styling */
  .manual-connect-card {
    background: #151515;
    border-radius: 8px;
    border: 1px solid #252525;
    margin: 12px 14px;
    padding: 12px;
  }
  .manual-form {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }
  .form-row {
    display: flex;
    gap: 8px;
  }
  .form-group {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .form-group.flex-2 { flex: 2; }
  .form-group.flex-1 { flex: 1; }
  .form-group label {
    font-size: 9px;
    font-weight: 700;
    color: #64748b;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }
  .form-group input {
    background: #1e1e1e !important;
    border: 1px solid #2e2e2e !important;
    border-radius: 6px !important;
    color: #e2e8f0 !important;
    font-size: 12px !important;
    padding: 6px 10px !important;
    outline: none !important;
    margin-bottom: 0 !important;
    transition: border-color 0.15s;
  }
  .form-group input:focus {
    border-color: #818cf8 !important;
  }
  .pin-input {
    letter-spacing: 6px !important;
    text-align: center !important;
    font-weight: 700 !important;
    font-size: 14px !important;
  }
  .connect-btn {
    width: 100%;
    background: #818cf8;
    color: white;
    border: none;
    border-radius: 6px;
    padding: 8px;
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    transition: background 0.15s;
  }
  .connect-btn:hover {
    background: #6366f1;
  }
  .connection-error-box {
    display: flex;
    align-items: center;
    gap: 6px;
    background: rgba(239, 68, 68, 0.1);
    border: 1px solid rgba(239, 68, 68, 0.2);
    border-radius: 6px;
    padding: 6px 10px;
  }
  .err-icon {
    font-size: 12px;
  }
  .err-msg {
    color: #f87171;
    font-size: 10px;
    font-weight: 500;
    line-height: 1.3;
  }

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

  .title-bar {
    display: flex; align-items: center; gap: 10px;
    padding: 12px 24px 0; flex-shrink: 0;
  }
  .emoji-input-btn {
    width: 38px;
    height: 38px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 20px;
    background: #1e1e1e;
    border: 1px solid #2e2e2e;
    border-radius: 8px;
    color: #e2e8f0;
    cursor: pointer;
    flex-shrink: 0;
    transition: all 0.12s;
  }
  .emoji-input-btn:hover {
    border-color: #818cf8;
    background: rgba(129, 140, 248, 0.05);
  }
  .title-input {
    flex: 1; background: transparent; border: none; outline: none;
    font-size: 22px; font-weight: 700; color: #e2e8f0;
    padding: 6px 0; border-bottom: 2px solid transparent;
  }
  .title-input:focus { border-bottom-color: #818cf8; }
  .title-input::placeholder { color: #4a4a4a; }

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

  /* Notion-style Block Selector Modal Styles */
  .block-selector-modal {
    width: 440px;
    max-width: 90%;
  }
  .block-options-grid {
    display: flex;
    flex-direction: column;
    gap: 10px;
    padding: 16px !important;
  }
  .block-option-row {
    display: flex;
    align-items: center;
    gap: 14px;
    background: #1e1e1e;
    border: 1px solid #2e2e2e;
    border-radius: 8px;
    padding: 12px 16px;
    text-align: left;
    cursor: pointer;
    color: #e2e8f0;
    transition: all 0.15s ease;
    width: 100%;
  }
  .block-option-row:hover {
    background: rgba(129, 140, 248, 0.08);
    border-color: #818cf8;
    transform: translateY(-1px);
  }
  .block-icon {
    font-size: 24px;
    flex-shrink: 0;
  }
  .block-desc {
    display: flex;
    flex-direction: column;
    gap: 3px;
  }
  .block-title {
    font-size: 13px;
    font-weight: 700;
    color: #e2e8f0;
  }
  .block-subtitle {
    font-size: 11px;
    color: #64748b;
    line-height: 1.3;
  }

  /* Emoji Picker Modal Styles */
  .emoji-picker-modal {
    width: 480px;
    max-width: 95%;
    height: 450px;
    max-height: 85vh;
  }
  .emoji-picker-tabs {
    display: flex;
    gap: 4px;
    background: #121212;
    border-bottom: 1px solid #2e2e2e;
    padding: 6px 12px;
    overflow-x: auto;
    scrollbar-width: none;
  }
  .emoji-picker-tabs::-webkit-scrollbar {
    display: none;
  }
  .emoji-tab-btn {
    background: transparent;
    border: none;
    color: #64748b;
    font-size: 12px;
    font-weight: 600;
    padding: 6px 12px;
    border-radius: 4px;
    cursor: pointer;
    white-space: nowrap;
    transition: all 0.12s;
  }
  .emoji-tab-btn:hover {
    color: #94a3b8;
    background: #1a1a1a;
  }
  .emoji-tab-btn.active {
    color: #818cf8;
    background: rgba(129, 140, 248, 0.1);
  }
  .emoji-picker-body {
    padding: 12px !important;
    overflow-y: auto !important;
  }
  .emoji-grid {
    display: grid;
    grid-template-columns: repeat(8, 1fr);
    gap: 6px;
  }
  .emoji-select-btn {
    background: transparent;
    border: none;
    font-size: 24px;
    aspect-ratio: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 6px;
    cursor: pointer;
    transition: background 0.12s;
  }
  .emoji-select-btn:hover {
    background: rgba(255, 255, 255, 0.05);
  }

  /* Modal Base Backdrop / Container Shared Styles */
  .modal-backdrop {
    position: fixed;
    top: 0;
    left: 0;
    width: 100vw;
    height: 100vh;
    background: rgba(0, 0, 0, 0.6);
    backdrop-filter: blur(4px);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
    cursor: default;
  }
  .modal-container {
    background: #1a1a1a;
    border: 1px solid #2e2e2e;
    border-radius: 12px;
    display: flex;
    flex-direction: column;
    box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.5), 0 8px 10px -6px rgba(0, 0, 0, 0.5);
    overflow: hidden;
    animation: modal-fade-in 0.2s cubic-bezier(0.16, 1, 0.3, 1);
  }
  @keyframes modal-fade-in {
    from {
      opacity: 0;
      transform: scale(0.95) translateY(10px);
    }
    to {
      opacity: 1;
      transform: scale(1) translateY(0);
    }
  }
  .modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 14px 16px;
    border-bottom: 1px solid #2e2e2e;
  }
  .modal-header h3 {
    margin: 0;
    font-size: 15px;
    font-weight: 700;
    color: #e2e8f0;
  }
  .close-btn {
    background: transparent;
    border: none;
    color: #8e8e8e;
    font-size: 20px;
    cursor: pointer;
    line-height: 1;
    padding: 4px;
    border-radius: 4px;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.15s ease;
  }
  .close-btn:hover {
    color: #f87171;
    background: rgba(239, 68, 68, 0.1);
  }
  .modal-body {
    flex: 1;
    overflow-y: auto;
  }
</style>

<script context="module">
  import { default as TreeRender } from './TreeRender.svelte';
  import { default as CardBlock } from './CardBlock.svelte';
  import { default as RightSidebar } from './RightSidebar.svelte';
</script>
