<script>
  export var page;
  export var allPages;
  export var selectedPage;
  export var expandedPageIds;
  export var selectPage;
  export var createPage;
  export var deletePage;
  export var toggleExpand;
  export var getChildrenOf;

  import { MovePage } from '../wailsjs/go/main/App.js';

  $: children = getChildrenOf(page.id);
  $: hasChildren = children.length > 0;
  $: isExpanded = expandedPageIds[page.id] || false;
  $: isSelected = selectedPage && selectedPage.id === page.id;

  async function handleMoveTo() {
    const pages = allPages.filter(p => p.id !== page.id);
    const options = pages.map(p => `${p.emoji} ${p.title || 'Untitled'} (${p.id})`);
    options.unshift('0: 📂 Root Level (No Parent)');
    
    const choice = prompt('Enter the number for destination:\n' + options.join('\n'));
    if (choice == null) return;
    
    const idx = parseInt(choice);
    if (idx === 0) {
      await MovePage(page.id, '');
    } else if (idx > 0 && idx <= pages.length) {
      await MovePage(page.id, pages[idx - 1].id);
    }
  }
</script>

<div class="tree-node">
  <div class="node-row {isSelected ? 'active' : ''}">
    <button class="expand-arrow" on:click|stopPropagation={() => toggleExpand(page.id)}>
      {#if hasChildren}
        <span class="arrow {isExpanded ? 'open' : ''}">▸</span>
      {:else}
        <span class="dot">•</span>
      {/if}
    </button>

    <button class="node-content" on:click={() => selectPage(page)}>
      <span class="emoji">{page.emoji}</span>
      <span class="title" title={page.title}>{page.title || 'Untitled'}</span>
    </button>

    <div class="node-actions">
      <button class="action-btn" title="Add subpage" on:click|stopPropagation={() => createPage(page.id)}>+</button>
      <button class="action-btn" title="Move to..." on:click|stopPropagation={handleMoveTo}>↗</button>
      <button class="action-btn delete" title="Archive page" on:click|stopPropagation={() => deletePage(page.id)}>×</button>
    </div>
  </div>

  {#if hasChildren && isExpanded}
    <div class="children-list">
      {#each children as child}
        <svelte:self 
          page={child} 
          {allPages} 
          {selectedPage} 
          {expandedPageIds} 
          {selectPage} 
          {createPage} 
          {deletePage} 
          {toggleExpand} 
          {getChildrenOf}
        />
      {/each}
    </div>
  {/if}
</div>

<style>
  .tree-node {
    display: flex;
    flex-direction: column;
    width: 100%;
  }

  .node-row {
    display: flex;
    align-items: center;
    padding: 2px 4px;
    border-radius: 4px;
    cursor: pointer;
    transition: background-color 0.15s ease;
    position: relative;
  }

  .node-row:hover {
    background-color: #242424;
  }

  .node-row.active {
    background-color: rgba(129, 140, 248, 0.08);
    color: #a5b4fc;
  }

  .expand-arrow {
    background: transparent;
    border: none;
    color: #6c6c6c;
    cursor: pointer;
    width: 18px;
    height: 18px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 10px;
    padding: 0;
  }

  .expand-arrow:hover {
    color: white;
  }

  .arrow {
    display: inline-block;
    transition: transform 0.15s ease;
  }

  .arrow.open {
    transform: rotate(90deg);
  }

  .dot {
    font-size: 8px;
    color: #3e3e3e;
  }

  .node-content {
    background: transparent;
    border: none;
    display: flex;
    align-items: center;
    gap: 6px;
    flex-grow: 1;
    text-align: left;
    color: inherit;
    cursor: pointer;
    padding: 4px 2px;
    overflow: hidden;
  }

  .emoji {
    font-size: 14px;
  }

  .title {
    font-size: 13px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-weight: 500;
  }

  .node-actions {
    display: none;
    align-items: center;
    gap: 4px;
    position: absolute;
    right: 6px;
    background-color: #1a1a1a;
    padding-left: 8px;
  }

  .node-row:hover .node-actions {
    display: flex;
    background-color: #242424;
  }

  .action-btn {
    background: transparent;
    border: none;
    color: #8e8e8e;
    cursor: pointer;
    font-size: 12px;
    width: 16px;
    height: 16px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 3px;
  }

  .action-btn:hover {
    background-color: #3e3e3e;
    color: white;
  }

  .action-btn.delete:hover {
    color: #f87171;
    background-color: rgba(239, 68, 68, 0.1);
  }

  .children-list {
    margin-left: 12px;
    border-left: 1px solid #242424;
    padding-left: 6px;
  }
</style>
