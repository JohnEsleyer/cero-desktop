<script>
  import PageIcon from "./PageIcon.svelte";

  let { sidePages = [], selectedPage = null, onSelectPage, onCreateSidePage, onDeletePage } = $props();
</script>

<div class="right-sidebar">
  <div class="sidebar-header">
    <span class="header-title">Context</span>
    {#if selectedPage}
      <button
        class="add-btn"
        title="Add context page"
        onclick={onCreateSidePage}>+</button
      >
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
      <span class="empty-hint"
        >Side pages provide supplementary info about the current page.</span
      >
      <button class="add-btn-label" onclick={onCreateSidePage}
        >+ Add Context Page</button
      >
    </div>
  {:else}
    <div class="pages-list">
      {#each sidePages as sp}
        <div class="page-card" class:selected={false}>
          <button class="page-btn" onclick={() => onSelectPage(sp)}>
            <span class="page-emoji"><PageIcon emoji={sp.emoji} size={14} /></span>
            <div class="page-info">
              <span class="page-title">{sp.title || "Untitled"}</span>
            </div>
          </button>
          <button
            class="delete-btn"
            title="Remove context page"
            onclick={(e) => { e.stopPropagation(); onDeletePage(sp.id); }}
          >
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
    background-color: #121215;
    overflow: hidden;
  }

  .sidebar-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 14px 20px;
    border-bottom: 1px solid rgba(255, 255, 255, 0.05);
    flex-shrink: 0;
  }

  .header-title {
    font-size: 9px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 1px;
    color: #71717a;
  }

  .add-btn {
    background: transparent;
    border: 1px solid rgba(255, 255, 255, 0.05);
    color: #a1a1aa;
    cursor: pointer;
    font-size: 13px;
    width: 20px;
    height: 20px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 4px;
    transition: all 0.15s ease;
  }
  .add-btn:hover {
    border-color: rgba(129, 140, 248, 0.3);
    color: #818cf8;
    background: rgba(129, 140, 248, 0.04);
  }

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
    font-size: 24px;
    margin-bottom: 12px;
    opacity: 0.6;
  }

  .empty-state p {
    font-size: 11px;
    color: #71717a;
    margin: 0 0 6px 0;
    font-weight: 500;
  }

  .empty-hint {
    font-size: 10px;
    color: #52525b;
    line-height: 1.4;
    margin-bottom: 16px;
    max-width: 180px;
  }

  .add-btn-label {
    background: rgba(255, 255, 255, 0.01);
    border: 1px dashed rgba(255, 255, 255, 0.06);
    color: #818cf8;
    cursor: pointer;
    font-size: 11px;
    font-weight: 600;
    padding: 8px 16px;
    border-radius: 6px;
    transition: all 0.15s ease;
  }
  .add-btn-label:hover {
    border-color: rgba(129, 140, 248, 0.3);
    background: rgba(129, 140, 248, 0.04);
  }

  .pages-list {
    flex: 1;
    overflow-y: auto;
    padding: 10px;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .page-card {
    display: flex;
    align-items: center;
    border-radius: 6px;
    border: 1px solid transparent;
    transition: all 0.12s ease;
  }
  .page-card:hover {
    background: rgba(255, 255, 255, 0.01);
    border-color: rgba(255, 255, 255, 0.02);
  }

  .page-btn {
    flex: 1;
    display: flex;
    align-items: center;
    gap: 8px;
    background: transparent;
    border: none;
    text-align: left;
    color: #cbd5e1;
    padding: 6px 8px;
    cursor: pointer;
    border-radius: 6px;
    min-width: 0;
  }

  .page-emoji {
    font-size: 14px;
    flex-shrink: 0;
  }

  .page-info {
    flex: 1;
    min-width: 0;
  }

  .page-title {
    font-size: 12px;
    font-weight: 500;
    display: block;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .delete-btn {
    background: transparent;
    border: none;
    color: #52525b;
    cursor: pointer;
    font-size: 14px;
    padding: 4px 8px;
    opacity: 0;
    transition: all 0.12s ease;
    flex-shrink: 0;
  }
  .page-card:hover .delete-btn {
    opacity: 1;
  }
  .delete-btn:hover {
    color: #f87171;
  }
</style>
