<script>
  import { onMount } from 'svelte';
  import { EventsOn } from '../wailsjs/runtime/runtime.js';
  import { 
    GetConnectionStatus, 
    GetDbItems, 
    GetDiscoveredDevices, 
    ConnectToDevice, 
    Disconnect, 
    AddItem, 
    UpdateItem, 
    DeleteItem
  } from '../wailsjs/go/main/App.js';

  // State
  let connectionStatus = 'disconnected';
  let discoveredDevices = [];
  let dbItems = [];
  let connectedDevice = null; // { ip, port, name }
  
  // Form State
  let showModal = false;
  let modalTitle = 'Create Record';
  let currentItemId = '';
  let titleInput = '';
  let contentInput = '';

  // Manual IP input
  let manualIp = '';
  let manualPort = 9090;
  let connectionError = '';

  onMount(async () => {
    // 1. Initial State Load
    try {
      connectionStatus = await GetConnectionStatus();
      discoveredDevices = await GetDiscoveredDevices();
      dbItems = await GetDbItems();
    } catch (e) {
      console.error("Failed to load initial state:", e);
    }

    // 2. Subscribe to Wails Events
    EventsOn('connection-status', (status) => {
      connectionStatus = status;
      if (status === 'disconnected') {
        connectedDevice = null;
        dbItems = [];
      }
      connectionError = '';
    });

    EventsOn('discovered-devices', (devices) => {
      discoveredDevices = devices;
    });

    EventsOn('db-update', (items) => {
      dbItems = items;
    });
  });

  async function handleConnect(device) {
    connectionError = '';
    try {
      connectedDevice = device;
      await ConnectToDevice(device.ip, device.port);
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
    connectionError = '';
    const device = {
      ip: manualIp.trim(),
      port: parseInt(manualPort) || 9090,
      deviceName: 'Manual Entry'
    };
    
    try {
      connectedDevice = device;
      await ConnectToDevice(device.ip, device.port);
    } catch (err) {
      connectionError = err.toString();
      connectedDevice = null;
    }
  }

  async function handleDisconnect() {
    try {
      await Disconnect();
      connectedDevice = null;
      dbItems = [];
    } catch (err) {
      console.error(err);
    }
  }

  function openCreateModal() {
    modalTitle = 'Create Database Record';
    currentItemId = '';
    titleInput = '';
    contentInput = '';
    showModal = true;
  }

  function openEditModal(item) {
    modalTitle = 'Edit Database Record';
    currentItemId = item.id;
    titleInput = item.title;
    contentInput = item.content;
    showModal = true;
  }

  async function saveRecord() {
    if (!titleInput.trim() || !contentInput.trim()) return;

    try {
      if (currentItemId) {
        // Edit Mode
        await UpdateItem(currentItemId, titleInput.trim(), contentInput.trim());
      } else {
        // Create Mode
        await AddItem(titleInput.trim(), contentInput.trim());
      }
      showModal = false;
    } catch (err) {
      alert("Failed to save: " + err);
    }
  }

  async function handleDelete(id) {
    if (confirm("Are you sure you want to delete this record?")) {
      try {
        await DeleteItem(id);
      } catch (err) {
        alert("Failed to delete: " + err);
      }
    }
  }

  function formatTime(isoString) {
    if (!isoString) return '';
    try {
      const date = new Date(isoString);
      return date.toLocaleTimeString();
    } catch (e) {
      return isoString;
    }
  }
</script>

<main class="app-layout">
  <!-- LEFT COLUMN: Sidebar Connections -->
  <aside class="sidebar">
    <div class="sidebar-header">
      <div class="logo-container">
        <svg xmlns="http://www.w3.org/2000/svg" class="logo-icon" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 11c0 3.517-1.009 6.799-2.753 9.571m-3.44-2.04l.054-.09A13.916 13.916 0 008 11a4 4 0 118 0c0 1.017-.07 2.019-.203 3m-2.118 6.844A21.88 21.88 0 0015.171 17m3.839 1.132c.645-2.266.99-4.659.99-7.132A8 8 0 008 4.07M3 15.364c.64-1.319 1-2.8 1-4.364 0-1.457.39-2.823 1.07-4" />
        </svg>
        <span class="logo-text">PocketDB Link</span>
      </div>
      <div class="sync-badge {connectionStatus}">
        <span class="pulse-dot"></span>
        <span class="badge-text">{connectionStatus}</span>
      </div>
    </div>

    <!-- Section 1: Discovered Mobile Servers -->
    <div class="section-container">
      <div class="section-title">
        <span>Discovered Devices</span>
        <div class="scanning-indicator">
          <div class="sonar-wave"></div>
          <span>Scanning</span>
        </div>
      </div>

      <div class="devices-list">
        {#if discoveredDevices.length === 0}
          <div class="empty-list-message">
            <svg xmlns="http://www.w3.org/2000/svg" class="empty-icon" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M8.111 16.404a5.5 5.5 0 017.778 0M12 20h.01m-7.08-7.071a10 10 0 0114.142 0M1.398 6.541A16 16 0 0122.6 6.538" />
            </svg>
            <p>Searching for mobile database servers on the local WiFi network...</p>
          </div>
        {:else}
          {#each discoveredDevices as device}
            <div class="device-card {connectedDevice && connectedDevice.ip === device.ip ? 'active' : ''}">
              <div class="device-info">
                <span class="device-name">{device.deviceName}</span>
                <span class="device-address">{device.ip}:{device.port}</span>
              </div>
              {#if connectionStatus === 'connected' && connectedDevice && connectedDevice.ip === device.ip}
                <button class="btn btn-secondary disconnect-btn" on:click={handleDisconnect}>Disconnect</button>
              {:else if connectionStatus === 'connecting' && connectedDevice && connectedDevice.ip === device.ip}
                <button class="btn btn-primary connect-btn loading" disabled>Connecting...</button>
              {:else}
                <button class="btn btn-primary connect-btn" on:click={() => handleConnect(device)}>Connect</button>
              {/if}
            </div>
          {/each}
        {/if}
      </div>
    </div>

    <!-- Section 2: Manual Link -->
    <div class="section-container manual-section">
      <div class="section-title">Manual Connection</div>
      <div class="manual-form">
        <div class="form-group">
          <label for="manual-ip">Server IP Address</label>
          <input type="text" id="manual-ip" bind:value={manualIp} placeholder="e.g. 192.168.1.50" disabled={connectionStatus !== 'disconnected'} />
        </div>
        <div class="form-row">
          <div class="form-group">
            <label for="manual-port">Port</label>
            <input type="number" id="manual-port" bind:value={manualPort} placeholder="9090" disabled={connectionStatus !== 'disconnected'} />
          </div>
          <div class="form-group action-group">
            {#if connectionStatus === 'connected'}
              <button class="btn btn-secondary w-full" on:click={handleDisconnect}>Disconnect</button>
            {:else if connectionStatus === 'connecting'}
              <button class="btn btn-primary w-full loading" disabled>Connecting...</button>
            {:else}
              <button class="btn btn-primary w-full" on:click={handleConnectManually}>Connect</button>
            {/if}
          </div>
        </div>
        {#if connectionError}
          <div class="error-banner">{connectionError}</div>
        {/if}
      </div>
    </div>
  </aside>

  <!-- RIGHT COLUMN: Main Database Workspace -->
  <section class="main-workspace">
    {#if connectionStatus !== 'connected'}
      <!-- Disconnected Welcome State -->
      <div class="welcome-container">
        <div class="welcome-card">
          <div class="icon-pulse">
            <svg xmlns="http://www.w3.org/2000/svg" class="pulse-icon" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
            </svg>
          </div>
          <h2>Desktop Link Database Hub</h2>
          <p>
            This application allows you to view and edit data synchronised with the <strong>PocketDB Mobile</strong> app.
            Mobile is the source of truth, establishing an automatic web socket communication link.
          </p>
          <div class="guide-steps">
            <div class="step">
              <div class="step-num">1</div>
              <div class="step-desc">Open the Flutter Mobile App on the same WiFi Network.</div>
            </div>
            <div class="step">
              <div class="step-num">2</div>
              <div class="step-desc">Start the Local Sync Server on Mobile.</div>
            </div>
            <div class="step">
              <div class="step-num">3</div>
              <div class="step-desc">Click "Connect" on the discovered device or enter the IP manually.</div>
            </div>
          </div>
        </div>
      </div>
    {:else}
      <!-- Connected Database Workspace -->
      <div class="workspace-header">
        <div class="header-info">
          <h1>Synchronized Database Workspace</h1>
          <p class="connected-text">
            Active link to <strong>{connectedDevice ? connectedDevice.deviceName : 'Mobile Server'}</strong> ({connectedDevice ? connectedDevice.ip : ''})
          </p>
        </div>
        <button class="btn btn-primary add-record-btn" on:click={openCreateModal}>
          <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" class="btn-icon">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
          </svg>
          Add Record
        </button>
      </div>

      <div class="db-workspace-content">
        {#if dbItems.length === 0}
          <div class="empty-db-state">
            <svg xmlns="http://www.w3.org/2000/svg" class="empty-db-icon" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1" d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4m0 5c0 2.21-3.582 4-8 4s-8-1.79-8-4" />
            </svg>
            <h3>Database is Empty</h3>
            <p>Add records using the button above or on the mobile device. They will sync instantly.</p>
          </div>
        {:else}
          <div class="records-grid">
            {#each dbItems as item}
              <div class="record-card">
                <div class="record-header">
                  <h3>{item.title}</h3>
                  <div class="record-actions">
                    <button class="icon-btn" title="Edit" on:click={() => openEditModal(item)}>
                      <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
                      </svg>
                    </button>
                    <button class="icon-btn delete" title="Delete" on:click={() => handleDelete(item.id)}>
                      <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                      </svg>
                    </button>
                  </div>
                </div>
                <p class="record-content">{item.content}</p>
                <div class="record-footer">
                  <div class="sync-status">
                    <svg xmlns="http://www.w3.org/2000/svg" class="sync-icon" viewBox="0 0 20 20" fill="currentColor">
                      <path fill-rule="evenodd" d="M6.267 3.455a.75.75 0 00-.708.522L4.547 7.25h1.905a.75.75 0 010 1.5H3.5a.75.75 0 01-.75-.75V5.5a.75.75 0 011.5 0v1.07l1.309-3.926a2.25 2.25 0 012.124-1.544h7.634a2.25 2.25 0 012.124 1.544l1.309 3.926v-1.07a.75.75 0 011.5 0v2.5a.75.75 0 01-.75.75h-2.952a.75.75 0 010-1.5h1.905l-1.012-3.273a.75.75 0 00-.708-.522H6.267zM2.08 13.5a.75.75 0 01.75-.75h14.34a.75.75 0 01.75.75v3.25a2.25 2.25 0 01-2.25 2.25H4.25A2.25 2.25 0 012 16.75V13.5zm1.5 2.5a.75.75 0 000 1.5h12.84a.75.75 0 000-1.5H3.58z" clip-rule="evenodd" />
                    </svg>
                    <span>Real-time Synced</span>
                  </div>
                  <span class="time-stamp">Updated {formatTime(item.updatedAt)}</span>
                </div>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    {/if}
  </section>
</main>

<!-- Overlay Modal for Add/Edit -->
{#if showModal}
  <div class="modal-overlay" on:click|self={() => showModal = false}>
    <div class="modal-card">
      <div class="modal-header">
        <h2>{modalTitle}</h2>
        <button class="close-btn" on:click={() => showModal = false}>
          <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>
      <div class="modal-body">
        <div class="form-group">
          <label for="record-title">Title</label>
          <input type="text" id="record-title" bind:value={titleInput} placeholder="Enter record title..." />
        </div>
        <div class="form-group">
          <label for="record-content">Content</label>
          <textarea id="record-content" rows="4" bind:value={contentInput} placeholder="Enter record details / content..."></textarea>
        </div>
      </div>
      <div class="modal-footer">
        <button class="btn btn-secondary" on:click={() => showModal = false}>Cancel</button>
        <button class="btn btn-primary" on:click={saveRecord} disabled={!titleInput.trim() || !contentInput.trim()}>
          Save Record
        </button>
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
    background-color: #0f172a;
    color: #e2e8f0;
  }

  /* SIDEBAR STYLES */
  .sidebar {
    width: 340px;
    background-color: #1e293b;
    border-right: 1px solid #334155;
    display: flex;
    flex-direction: column;
    height: 100%;
    box-shadow: 4px 0 24px rgba(0, 0, 0, 0.15);
    z-index: 10;
  }

  .sidebar-header {
    padding: 20px;
    border-bottom: 1px solid #334155;
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .logo-container {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .logo-icon {
    width: 24px;
    height: 24px;
    color: #6366f1;
  }

  .logo-text {
    font-size: 18px;
    font-weight: 800;
    letter-spacing: -0.5px;
    background: linear-gradient(135deg, #6366f1, #a855f7);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
  }

  .sync-badge {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 4px 10px;
    border-radius: 20px;
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .sync-badge.disconnected {
    background-color: rgba(239, 68, 68, 0.1);
    color: #ef4444;
  }

  .sync-badge.connecting {
    background-color: rgba(168, 85, 247, 0.1);
    color: #a855f7;
  }

  .sync-badge.connected {
    background-color: rgba(34, 197, 94, 0.1);
    color: #22c55e;
  }

  .pulse-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
  }

  .disconnected .pulse-dot {
    background-color: #ef4444;
  }

  .connecting .pulse-dot {
    background-color: #a855f7;
    animation: pulse 1.5s infinite ease-in-out;
  }

  .connected .pulse-dot {
    background-color: #22c55e;
  }

  @keyframes pulse {
    0% { transform: scale(0.9); opacity: 0.6; }
    50% { transform: scale(1.3); opacity: 1; }
    100% { transform: scale(0.9); opacity: 0.6; }
  }

  .section-container {
    padding: 20px;
    border-bottom: 1px solid #334155;
  }

  .section-title {
    font-size: 12px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 1px;
    color: #94a3b8;
    margin-bottom: 14px;
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .scanning-indicator {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 10px;
    color: #6366f1;
  }

  .sonar-wave {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background-color: #6366f1;
    position: relative;
  }

  .sonar-wave::after {
    content: '';
    position: absolute;
    width: 100%;
    height: 100%;
    border-radius: 50%;
    background-color: #6366f1;
    animation: sonar 1.5s infinite ease-out;
    top: 0;
    left: 0;
  }

  @keyframes sonar {
    0% { transform: scale(1); opacity: 0.8; }
    100% { transform: scale(3.5); opacity: 0; }
  }

  .devices-list {
    display: flex;
    flex-direction: column;
    gap: 10px;
    max-height: 240px;
    overflow-y: auto;
  }

  .empty-list-message {
    padding: 20px 10px;
    text-align: center;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 8px;
    color: #64748b;
  }

  .empty-icon {
    width: 28px;
    height: 28px;
  }

  .empty-list-message p {
    font-size: 11px;
    line-height: 1.5;
    margin: 0;
  }

  .device-card {
    background-color: #0f172a;
    border: 1px solid #334155;
    border-radius: 12px;
    padding: 12px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    transition: all 0.25s ease;
  }

  .device-card:hover {
    border-color: #475569;
    background-color: #1e293b;
    transform: translateY(-2px);
  }

  .device-card.active {
    border-color: #6366f1;
    background-color: rgba(99, 102, 241, 0.05);
  }

  .device-info {
    display: flex;
    flex-direction: column;
    gap: 3px;
  }

  .device-name {
    font-size: 14px;
    font-weight: 700;
  }

  .device-address {
    font-size: 11px;
    color: #64748b;
    font-family: monospace;
  }

  /* BUTTON STYLES */
  .btn {
    border-radius: 8px;
    font-size: 12px;
    font-weight: 600;
    padding: 8px 14px;
    cursor: pointer;
    border: none;
    transition: all 0.2s ease;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
  }

  .btn-primary {
    background-color: #6366f1;
    color: white;
  }

  .btn-primary:hover:not(:disabled) {
    background-color: #4f46e5;
    box-shadow: 0 4px 12px rgba(99, 102, 241, 0.3);
  }

  .btn-secondary {
    background-color: #334155;
    color: #e2e8f0;
  }

  .btn-secondary:hover:not(:disabled) {
    background-color: #475569;
  }

  .btn:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .w-full {
    width: 100%;
  }

  /* MANUAL CONNECTION FORM */
  .manual-section {
    border-bottom: none;
    margin-top: auto;
    background-color: rgba(15, 23, 42, 0.4);
  }

  .manual-form {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .form-group {
    display: flex;
    flex-direction: column;
    gap: 6px;
    flex: 1;
  }

  .form-group label {
    font-size: 11px;
    font-weight: 600;
    color: #94a3b8;
  }

  .form-group input, .form-group textarea {
    background-color: #0f172a;
    border: 1px solid #334155;
    border-radius: 8px;
    color: white;
    font-size: 13px;
    padding: 8px 12px;
    outline: none;
    transition: border-color 0.2s ease;
  }

  .form-group input:focus, .form-group textarea:focus {
    border-color: #6366f1;
  }

  .form-row {
    display: flex;
    gap: 10px;
    align-items: flex-end;
  }

  .action-group {
    flex: 1.2;
  }

  .error-banner {
    background-color: rgba(239, 68, 68, 0.1);
    border: 1px solid rgba(239, 68, 68, 0.2);
    color: #ef4444;
    border-radius: 8px;
    font-size: 11px;
    padding: 8px 12px;
    word-break: break-all;
  }

  /* MAIN WORKSPACE */
  .main-workspace {
    flex: 1;
    display: flex;
    flex-direction: column;
    background-color: #0f172a;
    height: 100%;
    overflow-y: auto;
  }

  /* WELCOME SCREEN */
  .welcome-container {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 40px;
  }

  .welcome-card {
    max-width: 520px;
    text-align: center;
    background-color: #1e293b;
    border: 1px solid #334155;
    border-radius: 24px;
    padding: 40px;
    box-shadow: 0 10px 30px rgba(0, 0, 0, 0.25);
  }

  .icon-pulse {
    width: 64px;
    height: 64px;
    border-radius: 50%;
    background-color: rgba(99, 102, 241, 0.1);
    display: flex;
    align-items: center;
    justify-content: center;
    margin: 0 auto 24px;
  }

  .pulse-icon {
    width: 32px;
    height: 32px;
    color: #6366f1;
  }

  .welcome-card h2 {
    font-size: 22px;
    font-weight: 800;
    margin: 0 0 12px 0;
    color: white;
  }

  .welcome-card p {
    font-size: 14px;
    line-height: 1.6;
    color: #94a3b8;
    margin: 0 0 28px 0;
  }

  .guide-steps {
    display: flex;
    flex-direction: column;
    gap: 16px;
    text-align: left;
  }

  .step {
    display: flex;
    align-items: center;
    gap: 14px;
    background-color: #0f172a;
    padding: 12px 16px;
    border-radius: 12px;
    border: 1px solid #334155;
  }

  .step-num {
    width: 24px;
    height: 24px;
    border-radius: 50%;
    background-color: #6366f1;
    color: white;
    font-weight: 700;
    font-size: 12px;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }

  .step-desc {
    font-size: 13px;
    color: #e2e8f0;
  }

  /* WORKSPACE CONNECTED */
  .workspace-header {
    padding: 24px 40px;
    border-bottom: 1px solid #334155;
    display: flex;
    justify-content: space-between;
    align-items: center;
    background-color: #1e293b;
  }

  .header-info h1 {
    font-size: 20px;
    font-weight: 800;
    color: white;
    margin: 0 0 4px 0;
  }

  .connected-text {
    font-size: 13px;
    color: #94a3b8;
    margin: 0;
  }

  .connected-text strong {
    color: #22c55e;
  }

  .btn-icon {
    width: 16px;
    height: 16px;
  }

  .add-record-btn {
    padding: 10px 18px;
    font-size: 13px;
  }

  .db-workspace-content {
    flex: 1;
    padding: 40px;
    overflow-y: auto;
  }

  .empty-db-state {
    text-align: center;
    padding: 60px 40px;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    color: #64748b;
  }

  .empty-db-icon {
    width: 64px;
    height: 64px;
    margin-bottom: 16px;
    color: #334155;
  }

  .empty-db-state h3 {
    font-size: 18px;
    color: #cbd5e1;
    margin: 0 0 8px 0;
  }

  .empty-db-state p {
    font-size: 13px;
    max-width: 320px;
    margin: 0;
    line-height: 1.5;
  }

  .records-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
    gap: 20px;
  }

  .record-card {
    background-color: #1e293b;
    border: 1px solid #334155;
    border-radius: 16px;
    padding: 20px;
    display: flex;
    flex-direction: column;
    box-shadow: 0 4px 12px rgba(0,0,0,0.1);
    transition: all 0.2s ease;
  }

  .record-card:hover {
    border-color: #6366f1;
    transform: translateY(-2px);
    box-shadow: 0 6px 18px rgba(0,0,0,0.15);
  }

  .record-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: 12px;
    margin-bottom: 10px;
  }

  .record-header h3 {
    font-size: 16px;
    font-weight: 700;
    margin: 0;
    color: white;
    line-height: 1.4;
  }

  .record-actions {
    display: flex;
    gap: 6px;
  }

  .icon-btn {
    background: transparent;
    border: none;
    width: 28px;
    height: 28px;
    border-radius: 6px;
    display: flex;
    align-items: center;
    justify-content: center;
    color: #94a3b8;
    cursor: pointer;
    transition: all 0.2s ease;
  }

  .icon-btn svg {
    width: 16px;
    height: 16px;
  }

  .icon-btn:hover {
    background-color: #334155;
    color: white;
  }

  .icon-btn.delete:hover {
    background-color: rgba(239, 68, 68, 0.1);
    color: #ef4444;
  }

  .record-content {
    font-size: 13.5px;
    line-height: 1.6;
    color: #cbd5e1;
    margin: 0 0 16px 0;
    flex: 1;
    white-space: pre-wrap;
  }

  .record-footer {
    border-top: 1px solid #334155;
    padding-top: 12px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    font-size: 11px;
    color: #64748b;
  }

  .sync-status {
    display: flex;
    align-items: center;
    gap: 4px;
    color: #22c55e;
  }

  .sync-icon {
    width: 12px;
    height: 12px;
  }

  .time-stamp {
    font-family: monospace;
  }

  /* MODAL OVERLAY */
  .modal-overlay {
    position: fixed;
    top: 0;
    left: 0;
    width: 100vw;
    height: 100vh;
    background-color: rgba(15, 23, 42, 0.7);
    backdrop-filter: blur(4px);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 100;
  }

  .modal-card {
    background-color: #1e293b;
    border: 1px solid #334155;
    border-radius: 20px;
    width: 480px;
    max-width: 90%;
    box-shadow: 0 20px 40px rgba(0, 0, 0, 0.4);
    display: flex;
    flex-direction: column;
    overflow: hidden;
    animation: modal-enter 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
  }

  @keyframes modal-enter {
    0% { transform: scale(0.9) translateY(10px); opacity: 0; }
    100% { transform: scale(1) translateY(0); opacity: 1; }
  }

  .modal-header {
    padding: 20px 24px;
    border-bottom: 1px solid #334155;
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .modal-header h2 {
    font-size: 18px;
    font-weight: 700;
    margin: 0;
    color: white;
  }

  .close-btn {
    background: transparent;
    border: none;
    color: #64748b;
    cursor: pointer;
    width: 24px;
    height: 24px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 4px;
  }

  .close-btn:hover {
    background-color: #334155;
    color: white;
  }

  .close-btn svg {
    width: 18px;
    height: 18px;
  }

  .modal-body {
    padding: 24px;
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .modal-footer {
    padding: 16px 24px;
    border-top: 1px solid #334155;
    display: flex;
    justify-content: flex-end;
    gap: 10px;
    background-color: #161e2e;
  }
</style>
