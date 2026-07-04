<script>
  export let sidePages = [];
  export let selectedPage = null;
  export let onSelectPage;
  export let onCreateSidePage;
  export let onDeletePage;
</script>

<div class="right-sidebar">
  <div class="sidebar-header">
    <span class="header-title">Context</span>
    {#if selectedPage}
      <button class="add-btn" title="Add context page" on:click={onCreateSidePage}>+</button>
    {/if}
  </div>

  {#if !selectedPage}
    <div class="empty-state">
      <span class="empty-icon">📋</span>
      <p>Select a page to view context pages</p>
    </div>
  {:else if sidePages.length === 0}
    <div class="empty-state">
      <span class="empty-icon">📎</span>
      <p>No context pages yet</p>
      <span class="empty-hint">Side pages provide supplementary info about the current page.</span>
      <button class="add-btn-label" on:click={onCreateSidePage}>+ Add Context Page</button>
    </div>
  {:else}
    <div class="pages-list">
      {#each sidePages as sp}
        <div class="page-card" class:selected={false}>
          <button class="page-btn" on:click={() => onSelectPage(sp)}>
            <span class="page-emoji">{sp.emoji}</span>
            <div class="page-info">
              <span class="page-title">{sp.title || 'Untitled'}</span>
            </div>
          </button>
          <button class="delete-btn" title="Remove context page" on:click|stopPropagation={() => onDeletePage(sp.id)}>
            ×
          </button>
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .right-sidebar {
    width: 100%;
    height: 100%;
    display: flex;
    flex-direction: column;
    background-color: #1a1a1a;
    overflow: hidden;
  }

  .sidebar-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 12px 14px;
    border-bottom: 1px solid #2e2e2e;
    flex-shrink: 0;
  }

  .header-title {
    font-size: 10px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 1px;
    color: #8e8e8e;
  }

  .add-btn {
    background: transparent;
    border: 1px solid #3e3e3e;
    color: #64748b;
    cursor: pointer;
    font-size: 14px;
    width: 22px;
    height: 22px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 4px;
    transition: all 0.15s ease;
  }
  .add-btn:hover { border-color: #818cf8; color: #818cf8; }

  .empty-state {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 24px;
    text-align: center;
  }

  .empty-icon {
    font-size: 32px;
    margin-bottom: 12px;
    opacity: 0.6;
  }

  .empty-state p {
    font-size: 12px;
    color: #6c6c6c;
    margin: 0 0 8px 0;
  }

  .empty-hint {
    font-size: 11px;
    color: #4a4a4a;
    line-height: 1.4;
    margin-bottom: 16px;
  }

  .add-btn-label {
    background: transparent;
    border: 1px dashed #3e3e3e;
    color: #818cf8;
    cursor: pointer;
    font-size: 12px;
    padding: 8px 16px;
    border-radius: 6px;
    transition: all 0.15s ease;
  }
  .add-btn-label:hover { border-color: #818cf8; background: rgba(129,140,248,0.05); }

  .pages-list {
    flex: 1;
    overflow-y: auto;
    padding: 8px;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .page-card {
    display: flex;
    align-items: center;
    border-radius: 6px;
    transition: background 0.12s ease;
  }
  .page-card:hover { background: #222; }

  .page-btn {
    flex: 1;
    display: flex;
    align-items: center;
    gap: 10px;
    background: transparent;
    border: none;
    text-align: left;
    color: #e2e8f0;
    padding: 8px 10px;
    cursor: pointer;
    border-radius: 6px;
    min-width: 0;
  }
  .page-btn:hover { background: rgba(255,255,255,0.03); }

  .page-emoji { font-size: 18px; flex-shrink: 0; }

  .page-info {
    flex: 1;
    min-width: 0;
  }

  .page-title {
    font-size: 13px;
    font-weight: 500;
    display: block;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .delete-btn {
    background: transparent;
    border: none;
    color: #4a4a4a;
    cursor: pointer;
    font-size: 16px;
    padding: 4px 8px;
    opacity: 0;
    transition: all 0.12s ease;
    flex-shrink: 0;
  }
  .page-card:hover .delete-btn { opacity: 1; }
  .delete-btn:hover { color: #f87171; }
</style>
