<script>
  export let sidePages = [];
  export let activeTab = 'main'; // 'main' or page ID
  export let onSelectTab;
  export let onCreateSidePage;
  export let mainPageTitle = 'Main';
  export let mainPageEmoji = '📝';

  function handleTabClick(tabId) {
    if (onSelectTab) onSelectTab(tabId);
  }
</script>

<div class="tabs-bar">
  <button
    class="tab-item"
    class:active={activeTab === 'main'}
    on:click={() => handleTabClick('main')}
  >
    <span class="tab-emoji">{mainPageEmoji}</span>
    <span class="tab-label">{mainPageTitle || 'Main'}</span>
  </button>

  {#each sidePages as sp}
    <button
      class="tab-item"
      class:active={activeTab === sp.id}
      on:click={() => handleTabClick(sp.id)}
    >
      <span class="tab-emoji">{sp.emoji}</span>
      <span class="tab-label">{sp.title || 'Untitled'}</span>
    </button>
  {/each}

  <button class="tab-add" title="Add side page" on:click={onCreateSidePage}>+</button>
</div>

<style>
  .tabs-bar {
    display: flex;
    align-items: center;
    gap: 2px;
    padding: 0 16px;
    background: #1a1a1a;
    border-bottom: 1px solid #2e2e2e;
    min-height: 36px;
    overflow-x: auto;
    scrollbar-width: none;
  }
  .tabs-bar::-webkit-scrollbar { display: none; }

  .tab-item {
    display: flex;
    align-items: center;
    gap: 6px;
    background: transparent;
    border: none;
    color: #64748b;
    font-size: 12px;
    font-weight: 500;
    padding: 8px 14px;
    cursor: pointer;
    white-space: nowrap;
    border-bottom: 2px solid transparent;
    transition: color 0.15s, border-color 0.15s;
    margin-bottom: -1px;
  }
  .tab-item:hover {
    color: #94a3b8;
  }
  .tab-item.active {
    color: #e2e8f0;
    border-bottom-color: #818cf8;
  }

  .tab-emoji {
    font-size: 13px;
  }

  .tab-label {
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 120px;
  }

  .tab-add {
    background: transparent;
    border: 1px dashed #3e3e3e;
    color: #4a4a4a;
    font-size: 14px;
    width: 26px;
    height: 26px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 6px;
    cursor: pointer;
    flex-shrink: 0;
    margin-left: 4px;
    transition: color 0.15s, border-color 0.15s;
  }
  .tab-add:hover {
    color: #818cf8;
    border-color: #818cf8;
  }
</style>
