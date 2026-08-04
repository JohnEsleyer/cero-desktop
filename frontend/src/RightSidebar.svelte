<script>
  let {
    activeCard = null,
    onAddContextNote,
    onOpenContextNote,
    onDeleteContextNote,
    onClose,
    dailyBookmarks = [],
    ontoggleDailyBookmark
  } = $props();

  function parseMetadata(commentField) {
    if (!commentField) return { color: "default", comments: [], contextNotes: [] };
    try {
      const data = JSON.parse(commentField);
      return {
        color: data.color || "default",
        comments: Array.isArray(data.comments) ? data.comments : [],
        contextNotes: Array.isArray(data.contextNotes) ? data.contextNotes : []
      };
    } catch (_) {
      return { color: "default", comments: [], contextNotes: [] };
    }
  }

  let contextNotes = $derived(activeCard ? parseMetadata(activeCard.comment).contextNotes : []);
  let activeCardLabel = $derived(activeCard ? (activeCard.type || 'BLOCK').toUpperCase() : '');
</script>

<div class="right-sidebar">
  <div class="sidebar-header">
    <div style="display: flex; flex-direction: column; gap: 2px;">
      <span class="header-title">Card Context Notes</span>
      {#if activeCard}
        <span style="font-size: 9px; color: #71717a; font-weight: 600;">
          Target: {activeCardLabel}
        </span>
      {/if}
    </div>
    <div style="display: flex; align-items: center; gap: 4px;">
      {#if activeCard}
        <button class="add-btn" title="Add context note" onclick={onAddContextNote}>+</button>
      {/if}
      {#if onClose}
        <button class="close-btn" title="Close context panel" onclick={onClose}>×</button>
      {/if}
    </div>
  </div>

  {#if !activeCard}
    <div class="empty-state">
      <span class="empty-icon">📝</span>
      <p>Select or view a block to manage context notes</p>
    </div>
  {:else if contextNotes.length === 0}
    <div class="empty-state">
      <span class="empty-icon">📂</span>
      <p>No context notes for this block</p>
      <span class="empty-hint">Add Markdown or HTML context notes to attach supplementary details.</span>
      <button class="add-btn-label" onclick={onAddContextNote}>+ Add Context Note</button>
    </div>
  {:else}
    <div class="pages-list">
      {#each contextNotes as note, i (note.id || i)}
        <div class="page-card">
          <button class="page-btn" onclick={() => onOpenContextNote(note, i)}>
            <span class="page-emoji">{note.type === "html" ? "🌐" : "📝"}</span>
            <div class="page-info">
              <span class="page-title">{note.title || "Context Note"}</span>
              <span style="font-size: 8.5px; font-weight: 800; color: {note.type === 'html' ? '#2dd4bf' : '#818cf8'};">{note.type.toUpperCase()}</span>
            </div>
          </button>
          <button
            class="delete-btn"
            title="Bookmark context note"
            style="color: {dailyBookmarks.some((b) => b.targetId === note.id || b.id === note.id) ? '#818cf8' : '#71717a'}; margin-right: 4px;"
            onclick={(e) => {
              e.stopPropagation();
              ontoggleDailyBookmark && ontoggleDailyBookmark({
                id: note.id,
                targetId: note.id,
                targetType: 'context_note',
                title: note.title || 'Context Note',
                content: note.content || '',
                contentType: note.type || 'markdown'
              });
            }}
          >
            {dailyBookmarks.some((b) => b.targetId === note.id || b.id === note.id) ? "🔖" : "🏷️"}
          </button>
          <button class="delete-btn" title="Delete context note" onclick={(e) => { e.stopPropagation(); onDeleteContextNote(i); }}>×</button>
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
    color: #818cf8;
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

  .close-btn {
    background: transparent;
    border: none;
    color: #71717a;
    font-size: 16px;
    cursor: pointer;
    padding: 2px 6px;
    border-radius: 4px;
    display: flex;
    align-items: center;
    justify-content: center;
    line-height: 1;
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
    border: 1px solid rgba(255,255,255,0.03);
    background: rgba(255,255,255,0.01);
    transition: all 0.12s ease;
  }
  .page-card:hover {
    background: rgba(255, 255, 255, 0.03);
    border-color: rgba(129, 140, 248, 0.2);
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
    padding: 8px 10px;
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
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .page-title {
    font-size: 12px;
    font-weight: 600;
    display: block;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: #ffffff;
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
