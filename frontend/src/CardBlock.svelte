<script>
  import { marked } from 'marked';
  import { UpdateCard, DeleteCard, SaveImage, GetImage } from '../wailsjs/go/main/App.js';

  export let card;
  export let isSelected = false;
  export let allPages = [];
  export let onSelect;
  export let onDeleted;
  export let onNavigate;

  let editing = false;
  let editContent = card.content || '';

  // Sync editContent when card changes externally
  $: if (card.id && !editing) {
    editContent = card.content || '';
  }

  function handleClick() {
    if (onSelect) onSelect(card);
  }

  function startEdit() {
    if (card.type !== 'markdown') return;
    editing = true;
    editContent = card.content || '';
  }

  function stopEdit() {
    editing = false;
    if (editContent !== card.content) {
      UpdateCard(card.id, card.page_id, editContent).then(() => {
        card.content = editContent;
      }).catch(err => {
        console.error("Card save failed:", err);
      });
    }
  }

  function handleKeydown(e) {
    if (e.key === 'Escape') stopEdit();
  }

  function deleteCard(e) {
    e.stopPropagation();
    if (confirm('Delete this card?')) {
      DeleteCard(card.id, card.page_id).then(() => {
        if (onDeleted) onDeleted(card.id);
      }).catch(err => {
        console.error("Card delete failed:", err);
      });
    }
  }

  let imageInput;

  function setImage() {
    // Show options: URL or file upload
    const choice = prompt('Image source:\n1: Enter URL\n2: Upload file');
    if (choice === '1') {
      const url = prompt('Enter image URL:', card.content || '');
      if (url !== null) {
        UpdateCard(card.id, card.page_id, url).then(() => {
          card.content = url;
        }).catch(err => {
          console.error("Card save failed:", err);
        });
      }
    } else if (choice === '2') {
      imageInput.click();
    }
  }

  function handleFileUpload(e) {
    const file = e.target.files[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = () => {
      const base64 = reader.result;
      SaveImage(base64, file.name).then((filename) => {
        UpdateCard(card.id, card.page_id, filename).then(() => {
          card.content = filename;
        }).catch(err => {
          console.error("Card save failed:", err);
        });
      }).catch(err => {
        console.error("Image save failed:", err);
        alert("Failed to save image: " + err);
      });
    };
    reader.readAsDataURL(file);
    e.target.value = '';
  }

  function getImageSrc(content) {
    if (!content) return '';
    // If it's a URL, return as-is
    if (content.startsWith('http://') || content.startsWith('https://') || content.startsWith('data:')) {
      return content;
    }
    // If it's a local file path, return as-is
    if (content.startsWith('/') || content.startsWith('file://')) {
      return content;
    }
    // Otherwise it's a workspace image filename — use GetImage
    // For now, we'll try to construct the path
    return content;
  }

  function unlinkImage(e) {
    e.stopPropagation();
    if (confirm('Remove image?')) {
      UpdateCard(card.id, card.page_id, '').then(() => {
        card.content = '';
      }).catch(err => {
        console.error("Card save failed:", err);
      });
    }
  }

  function selectSubpage(e) {
    e.stopPropagation();
    const candidates = allPages.filter(p => p.id !== card.page_id);
    if (candidates.length === 0) {
      alert('No other pages to link to.');
      return;
    }
    // Build a simple selection prompt
    const lines = candidates.map((p, i) => `${i + 1}: ${p.emoji} ${p.title || 'Untitled'}`);
    const choice = prompt('Select a page to link:\n' + lines.join('\n'));
    if (choice == null) return;
    const idx = parseInt(choice);
    if (idx > 0 && idx <= candidates.length) {
      const target = candidates[idx - 1];
      UpdateCard(card.id, card.page_id, target.id).then(() => {
        card.content = target.id;
      }).catch(err => {
        console.error("Card save failed:", err);
      });
    }
  }

  function navigateToSubpage(e) {
    e.stopPropagation();
    if (!card.content || !onNavigate) return;
    const target = allPages.find(p => p.id === card.content);
    if (target) onNavigate(target);
  }

  $: linkedPage = (() => {
    if (card.type !== 'subpage_link' || !card.content) return null;
    return allPages.find(p => p.id === card.content) || null;
  })();

  $: renderedMarkdown = marked.parse(editing ? editContent : (card.content || ''));
</script>

<div class="card-block {card.type}" class:selected={isSelected} on:click={handleClick} role="button" tabindex="0">
  <!-- Markdown Card -->
  {#if card.type === 'markdown'}
    {#if editing}
      <div class="card-edit-area">
        <div class="edit-toolbar">
          <button class="tool-btn" title="Bold" on:click|stopPropagation={() => { editContent += '**bold**'; }}>B</button>
          <button class="tool-btn" title="Italic" on:click|stopPropagation={() => { editContent += '*italic*'; }}>I</button>
          <button class="tool-btn" title="Heading" on:click|stopPropagation={() => { editContent += '\n## '; }}>H</button>
          <button class="tool-btn" title="List" on:click|stopPropagation={() => { editContent += '\n- '; }}>List</button>
          <button class="tool-btn" title="Code" on:click|stopPropagation={() => { editContent += '\n```\n'; }}>Code</button>
          <span class="toolbar-spacer"></span>
          <button class="tool-btn done-btn" title="Done" on:click|stopPropagation={stopEdit}>Done</button>
        </div>
        <textarea
          class="card-textarea"
          bind:value={editContent}
          on:blur={stopEdit}
          on:keydown={handleKeydown}
          placeholder="Write something..."
        ></textarea>
      </div>
    {:else}
      <div class="card-preview markdown-rendered" on:dblclick={startEdit} role="button" tabindex="0">
        {#if card.content}
          {@html renderedMarkdown}
        {:else}
          <span class="empty-hint">Double-click to write...</span>
        {/if}
      </div>
    {/if}

  <!-- Image Card -->
  {:else if card.type === 'image'}
    <div class="image-card-content">
      <input type="file" bind:this={imageInput} accept="image/*" style="display:none" on:change={handleFileUpload} />
      {#if card.content}
        <div class="image-wrapper">
          <img src={getImageSrc(card.content)} alt="Card image" class="card-image"
               on:error={(e) => e.target.style.display='none'} />
        </div>
        <div class="image-actions">
          <button class="img-action-btn" title="Change image" on:click|stopPropagation={setImage}>Change</button>
          <button class="img-action-btn danger" title="Remove image" on:click|stopPropagation={unlinkImage}>Remove</button>
        </div>
      {:else}
        <div class="image-placeholder" on:click|stopPropagation={setImage} role="button" tabindex="0">
          <span class="placeholder-icon">🖼️</span>
          <span class="placeholder-text">Click to add image (URL or file)</span>
        </div>
      {/if}
    </div>

  <!-- Subpage Link Card -->
  {:else if card.type === 'subpage_link'}
    {#if linkedPage}
      <div class="subpage-link-card" on:click|stopPropagation={navigateToSubpage} role="button" tabindex="0">
        <span class="link-emoji">{linkedPage.emoji}</span>
        <div class="link-info">
          <span class="link-title">{linkedPage.title || 'Untitled'}</span>
          <span class="link-hint">Open subpage →</span>
        </div>
        <button class="link-action-btn" title="Change link" on:click|stopPropagation={selectSubpage}>Change</button>
      </div>
    {:else}
      <div class="subpage-empty" on:click|stopPropagation={selectSubpage} role="button" tabindex="0">
        <span>🔗</span>
        <span>Click to link a subpage...</span>
      </div>
    {/if}

  <!-- File Card -->
  {:else}
    <div class="file-card">
      <span>📎 {card.content || '(attach file)'}</span>
    </div>
  {/if}

  <!-- Actions (visible when selected) -->
  {#if isSelected}
    <div class="card-actions">
      <button class="action-btn" title="Delete card" on:click={deleteCard}>×</button>
    </div>
  {/if}
</div>

<style>
  .card-block {
    position: relative;
    background: #1e1e1e;
    border: 1px solid #2e2e2e;
    border-radius: 8px;
    padding: 12px;
    cursor: pointer;
    transition: border-color 0.15s ease;
  }
  .card-block:hover { border-color: #4a4a4a; }
  .card-block.selected { border-color: #818cf8; }

  /* Markdown */
  .card-preview {
    min-height: 20px;
    font-size: 13px;
    line-height: 1.6;
    color: #cbd5e1;
  }
  .card-preview:hover { background: rgba(255,255,255,0.02); border-radius: 4px; }
  .empty-hint { color: #4a4a4a; font-style: italic; }

  .card-edit-area { display: flex; flex-direction: column; }

  .edit-toolbar {
    display: flex;
    align-items: center;
    gap: 2px;
    padding: 4px 0;
    margin-bottom: 6px;
    border-bottom: 1px solid #2a2a2a;
  }
  .tool-btn {
    background: transparent;
    border: none;
    color: #94a3b8;
    font-size: 12px;
    padding: 3px 7px;
    border-radius: 3px;
    cursor: pointer;
    font-weight: 600;
  }
  .tool-btn:hover { background: #2a2a2a; color: #e2e8f0; }
  .done-btn { color: #818cf8; margin-left: auto; }
  .done-btn:hover { background: rgba(129,140,248,0.1); }
  .toolbar-spacer { flex: 1; }

  .card-textarea {
    width: 100%;
    min-height: 80px;
    background: #121212;
    border: 1px solid #333;
    border-radius: 6px;
    color: #e2e8f0;
    font-family: "Fira Code", monospace;
    font-size: 13px;
    padding: 8px;
    resize: vertical;
  }
  .card-textarea:focus { outline: none; border-color: #818cf8; }

  /* Image */
  .image-card-content { display: flex; flex-direction: column; gap: 8px; }
  .image-wrapper { display: flex; justify-content: center; }
  .card-image { max-width: 100%; max-height: 400px; border-radius: 4px; object-fit: contain; }
  .image-placeholder {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 24px;
    border: 2px dashed #3e3e3e;
    border-radius: 6px;
    cursor: pointer;
    transition: border-color 0.15s ease;
  }
  .image-placeholder:hover { border-color: #818cf8; }
  .placeholder-icon { font-size: 28px; margin-bottom: 6px; }
  .placeholder-text { color: #64748b; font-size: 12px; }
  .image-actions {
    display: flex;
    gap: 6px;
    justify-content: flex-end;
  }
  .img-action-btn {
    background: #2a2a2a;
    border: 1px solid #3e3e3e;
    color: #94a3b8;
    font-size: 11px;
    padding: 3px 8px;
    border-radius: 4px;
    cursor: pointer;
  }
  .img-action-btn:hover { background: #3e3e3e; color: #e2e8f0; }
  .img-action-btn.danger:hover { background: rgba(239,68,68,0.15); color: #f87171; }

  /* Subpage Link */
  .subpage-link-card {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 8px;
    border-radius: 6px;
    cursor: pointer;
    transition: background 0.12s ease;
  }
  .subpage-link-card:hover { background: #252525; }
  .link-emoji { font-size: 24px; }
  .link-info { flex: 1; display: flex; flex-direction: column; }
  .link-title { color: #e2e8f0; font-weight: 600; font-size: 14px; }
  .link-hint { color: #64748b; font-size: 11px; margin-top: 2px; }
  .link-action-btn {
    background: #2a2a2a;
    border: 1px solid #3e3e3e;
    color: #94a3b8;
    font-size: 11px;
    padding: 3px 8px;
    border-radius: 4px;
    cursor: pointer;
  }
  .link-action-btn:hover { background: #3e3e3e; color: #e2e8f0; }

  .subpage-empty {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 10px;
    border: 2px dashed #3e3e3e;
    border-radius: 6px;
    cursor: pointer;
    color: #64748b;
    font-size: 13px;
  }
  .subpage-empty:hover { border-color: #818cf8; color: #818cf8; }

  /* File */
  .file-card { color: #94a3b8; font-size: 13px; }

  /* Actions */
  .card-actions {
    position: absolute;
    top: 6px;
    right: 6px;
    display: flex;
    gap: 4px;
  }
  .action-btn {
    background: #2a2a2a;
    border: 1px solid #3e3e3e;
    color: #8e8e8e;
    cursor: pointer;
    font-size: 12px;
    width: 20px;
    height: 20px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 4px;
  }
  .action-btn:hover { background: #3e3e3e; color: white; }

  /* Markdown rendered */
  .markdown-rendered :global(h1) { font-size: 18px; margin: 16px 0 8px 0; color: white; }
  .markdown-rendered :global(h2) { font-size: 16px; margin: 14px 0 6px 0; color: white; }
  .markdown-rendered :global(h3) { font-size: 14px; margin: 12px 0 4px 0; color: white; }
  .markdown-rendered :global(code) { background: #1a1a1a; padding: 2px 4px; border-radius: 4px; font-size: 12px; }
  .markdown-rendered :global(pre) { background: #1a1a1a; padding: 10px; border-radius: 6px; overflow-x: auto; }
  .markdown-rendered :global(ul), .markdown-rendered :global(ol) { padding-left: 18px; }
  .markdown-rendered :global(li) { margin-bottom: 3px; }
</style>
