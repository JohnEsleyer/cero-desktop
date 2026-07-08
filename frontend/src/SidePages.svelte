<script>
  export let sidePages = [];
  export let activeTab = "main"; // 'main' or page ID
  export let onSelectTab;
  export let onCreateSidePage;
  export let mainPageTitle = "Main";
  import PageIcon from "./PageIcon.svelte";
  export let mainPageEmoji = "";

  function handleTabClick(tabId) {
    if (onSelectTab) onSelectTab(tabId);
  }
</script>

<div class="tabs-bar">
  <button
    class="tab-item"
    class:active={activeTab === "main"}
    on:click={() => handleTabClick("main")}
  >
    <span class="tab-emoji"><PageIcon emoji={mainPageEmoji} size={12} /></span>
    <span class="tab-label">{mainPageTitle || "Main"}</span>
  </button>

  {#each sidePages as sp}
    <button
      class="tab-item"
      class:active={activeTab === sp.id}
      on:click={() => handleTabClick(sp.id)}
    >
      <span class="tab-emoji"><PageIcon emoji={sp.emoji} size={12} /></span>
      <span class="tab-label">{sp.title || "Untitled"}</span>
    </button>
  {/each}

  <button class="tab-add" title="Add side page" on:click={onCreateSidePage}
    >+</button
  >
</div>

<style>
  .sidebar-header {
    display: none; /* Keeps native sizing from standard header structures if present */
  }

  .tabs-bar {
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 6px 20px;
    background: #09090b;
    border-bottom: 1px solid rgba(255, 255, 255, 0.05);
    min-height: 44px;
    overflow-x: auto;
    scrollbar-width: none;
  }
  .tabs-bar::-webkit-scrollbar {
    display: none;
  }

  .tab-item {
    display: flex;
    align-items: center;
    gap: 6px;
    background: transparent;
    border: 1px solid transparent;
    color: #71717a;
    font-size: 11px;
    font-weight: 600;
    padding: 6px 12px;
    cursor: pointer;
    white-space: nowrap;
    border-radius: 6px;
    transition: all 0.15s ease;
  }
  .tab-item:hover {
    color: #a1a1aa;
    background: rgba(255, 255, 255, 0.02);
  }
  .tab-item.active {
    color: #ffffff;
    background: rgba(255, 255, 255, 0.04);
    border-color: rgba(255, 255, 255, 0.05);
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
  }

  .tab-emoji {
    font-size: 12px;
  }

  .tab-label {
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 110px;
  }

  .tab-add {
    background: transparent;
    border: 1px dashed rgba(255, 255, 255, 0.05);
    color: #52525b;
    font-size: 12px;
    width: 24px;
    height: 24px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 6px;
    cursor: pointer;
    flex-shrink: 0;
    margin-left: 4px;
    transition: all 0.15s ease;
  }
  .tab-add:hover {
    color: #818cf8;
    border-color: rgba(129, 140, 248, 0.3);
    background: rgba(129, 140, 248, 0.04);
  }
</style>
