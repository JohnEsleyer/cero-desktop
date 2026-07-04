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

  import { MovePage } from "../wailsjs/go/main/App.js";

  $: children = getChildrenOf(page.id);
  $: hasChildren = children.length > 0;
  $: isExpanded = expandedPageIds[page.id] || false;
  $: isSelected = selectedPage && selectedPage.id === page.id;

  async function handleMoveTo() {
    const pages = allPages.filter((p) => p.id !== page.id);
    const options = pages.map(
      (p) => `${p.emoji} ${p.title || "Untitled"} (${p.id})`,
    );
    options.unshift("0: 📂 Root Level (No Parent)");

    const choice = prompt(
      "Enter the number for destination:\n" + options.join("\n"),
    );
    if (choice == null) return;

    const idx = parseInt(choice);
    if (idx === 0) {
      await MovePage(page.id, "");
    } else if (idx > 0 && idx <= pages.length) {
      await MovePage(page.id, pages[idx - 1].id);
    }
  }
</script>

<div class="tree-node">
  <div class="node-row {isSelected ? 'active' : ''}">
    <button
      class="expand-arrow"
      on:click|stopPropagation={() => toggleExpand(page.id)}
    >
      {#if hasChildren}
        <span class="arrow {isExpanded ? 'open' : ''}">▸</span>
      {:else}
        <span class="dot">•</span>
      {/if}
    </button>

    <button class="node-content" on:click={() => selectPage(page)}>
      <span class="emoji">{page.emoji}</span>
      <span class="title" title={page.title}>{page.title || "Untitled"}</span>
    </button>

    <div class="node-actions">
      <button
        class="action-btn"
        title="Add subpage"
        on:click|stopPropagation={() => createPage(page.id)}>+</button
      >
      <button
        class="action-btn"
        title="Move to..."
        on:click|stopPropagation={handleMoveTo}>↗</button
      >
      <button
        class="action-btn delete"
        title="Archive page"
        on:click|stopPropagation={() => deletePage(page.id)}>×</button
      >
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
    padding: 3px 6px;
    border-radius: 6px;
    cursor: pointer;
    transition: all 0.15s ease;
    position: relative;
  }

  .node-row:hover {
    background-color: rgba(255, 255, 255, 0.02);
  }

  .node-row.active {
    background-color: rgba(129, 140, 248, 0.06);
    color: #818cf8;
  }

  .expand-arrow {
    background: transparent;
    border: none;
    color: #52525b;
    cursor: pointer;
    width: 16px;
    height: 16px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 9px;
    padding: 0;
    transition: color 0.12s;
  }

  .expand-arrow:hover {
    color: #cbd5e1;
  }

  .arrow {
    display: inline-block;
    transition: transform 0.12s ease;
  }

  .arrow.open {
    transform: rotate(90deg);
  }

  .dot {
    font-size: 7px;
    color: #27272a;
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
    padding: 2px;
    overflow: hidden;
  }

  .emoji {
    font-size: 13px;
  }

  .title {
    font-size: 12px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-weight: 500;
  }

  .node-actions {
    display: none;
    align-items: center;
    gap: 2px;
    position: absolute;
    right: 4px;
    background-color: #121215;
    padding-left: 6px;
  }

  .node-row:hover .node-actions {
    display: flex;
    background-color: rgba(24, 24, 27, 0.98);
    border-radius: 4px;
    padding: 1px 3px;
    border: 1px solid rgba(255, 255, 255, 0.05);
  }

  .action-btn {
    background: transparent;
    border: none;
    color: #71717a;
    cursor: pointer;
    font-size: 11px;
    width: 15px;
    height: 15px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 3px;
    transition: all 0.12s;
  }

  .action-btn:hover {
    background-color: rgba(255, 255, 255, 0.05);
    color: #f4f4f5;
  }

  .action-btn.delete:hover {
    color: #f87171;
    background-color: rgba(239, 68, 68, 0.08);
  }

  /* Thin micro vertical guide-line tracing nesting paths */
  .children-list {
    margin-left: 11px;
    border-left: 1px solid rgba(255, 255, 255, 0.03);
    padding-left: 4px;
  }
</style>
