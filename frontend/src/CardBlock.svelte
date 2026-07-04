<script>
  import { marked } from 'marked';
  import { UpdateCard, DeleteCard, SaveImage, GetImage, AddPage, StartHTMLServer, StopHTMLServer } from '../wailsjs/go/main/App.js';

  export let card;
  export let isSelected = false;
  export let allPages = [];
  export let onSelect;
  export let onDeleted;
  export let onNavigate;

  let editing = false;
  let editContent = card.content || '';
  let showLinkModal = false;
  let pageSearchQuery = '';

  // Code Block variables
  let editingCode = false;
  let codeLang = 'javascript';
  let codeContent = '';

  // Sites Card variables
  let showSitesModal = false;
  let sitesTab = 'edit'; // 'edit' | 'preview'
  let sitesName = 'My HTML Site';
  let sitesDesc = 'Renders customized HTML template preview';
  let sitesHtml = '<h1>Sample Site</h1>\n<p>Edit HTML and watch it render live in the preview tab.</p>';
  let sitesLocalUrl = '';

  // Sync state properties on external card change
  $: if (card.id && !editing) {
    editContent = card.content || '';
  }

  $: if (card.type === 'code') {
    const raw = card.content || '';
    if (raw.includes('\n')) {
      const firstLineIdx = raw.indexOf('\n');
      codeLang = raw.substring(0, firstLineIdx).trim().toLowerCase();
      codeContent = raw.substring(firstLineIdx + 1);
    } else {
      codeLang = 'javascript';
      codeContent = raw;
    }
  }

  $: if (card.type === 'sites') {
    try {
      const parsed = JSON.parse(card.content || '{}');
      sitesName = parsed.name || 'My HTML Site';
      sitesDesc = parsed.description || 'Renders customized HTML template preview';
      sitesHtml = parsed.html || '<h1>Sample Site</h1>\n<p>Edit HTML and watch it render live in the preview tab.</p>';
    } catch (_) {
      sitesName = 'My HTML Site';
      sitesDesc = 'Renders customized HTML template preview';
      sitesHtml = card.content || '';
    }
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
    if (content.startsWith('http://') || content.startsWith('https://') || content.startsWith('data:')) {
      return content;
    }
    if (content.startsWith('/') || content.startsWith('file://')) {
      return content;
    }
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
    pageSearchQuery = '';
    showLinkModal = true;
  }

  function selectPageToLink(target) {
    showLinkModal = false;
    UpdateCard(card.id, card.page_id, target.id).then(() => {
      card.content = target.id;
    }).catch(err => {
      console.error("Card save failed:", err);
    });
  }

  function navigateToSubpage(e) {
    e.stopPropagation();
    if (!card.content || !onNavigate) return;
    const target = allPages.find(p => p.id === card.content);
    if (target) onNavigate(target);
  }

  function handleSaveCode() {
    const combined = `${codeLang}\n${codeContent}`;
    UpdateCard(card.id, card.page_id, combined).then(() => {
      card.content = combined;
    }).catch(err => console.error(err));
  }

  function getCodeHighlightHtml(code, lang) {
    if (!code) return '<span style="color: #64748b; font-style: italic;">// Empty code block...</span>';
    const escaped = code.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
    const keywords = ['function', 'return', 'if', 'else', 'for', 'while', 'const', 'let', 'var', 'import', 'class', 'void', 'final', 'def', 'package', 'func', 'interface'];
    const keywordsRegex = new RegExp(`\\b(${keywords.join('|')})\\b`, 'g');
    let highlighted = escaped
      .replace(keywordsRegex, '<span style="color: #f472b6; font-weight: bold;">$1</span>')
      .replace(/("(.*?)"|\'([^\']*)\')/g, '<span style="color: #34d399;">$1</span>')
      .replace(/(\/\/[^\n]*)/g, '<span style="color: #94a3b8; font-style: italic;">$1</span>');
    return highlighted;
  }

  async function openSitesModal() {
    showSitesModal = true;
    sitesTab = 'edit';
    sitesLocalUrl = '';
    try {
      if (typeof StartHTMLServer === 'function') {
        sitesLocalUrl = await StartHTMLServer(sitesHtml);
      } else {
        sitesLocalUrl = 'http://localhost:9095/fallback/' + card.id;
      }
    } catch (err) {
      console.error("Failed to start html server:", err);
    }
  }

  async function closeSitesModal() {
    handleSaveSites();
    showSitesModal = false;
    try {
      if (typeof StopHTMLServer === 'function') {
        await StopHTMLServer();
      }
    } catch (err) {
      console.error("Failed to stop html server:", err);
    }
  }

  async function handleSaveSites() {
    const combined = JSON.stringify({
      name: sitesName,
      description: sitesDesc,
      html: sitesHtml
    });
    UpdateCard(card.id, card.page_id, combined).then(() => {
      card.content = combined;
    }).catch(err => console.error(err));

    if (showSitesModal && typeof StartHTMLServer === 'function') {
      try {
        sitesLocalUrl = await StartHTMLServer(sitesHtml);
      } catch (err) {
        console.error(err);
      }
    }
  }

  async function handleCreateNewPageAndLink() {
    showLinkModal = false;
    try {
      const res = await AddPage(card.page_id, 'subpage', 'New Subpage Link', '📝');
      if (res && res.id) {
        await UpdateCard(card.id, card.page_id, res.id);
        card.content = res.id;
      }
    } catch (err) {
      console.error("Failed creating linked page in modal:", err);
    }
  }

  $: linkedPage = (() => {
    if (card.type !== 'subpage_link' || !card.content) return null;
    return allPages.find(p => p.id === card.content) || null;
  })();

  $: renderedMarkdown = marked.parse(editing ? editContent : (card.content || ''));

  $: filteredCandidates = allPages
    .filter(p => p.id !== card.page_id && p.relation_type !== 'sidepage')
    .filter(p => {
      if (!pageSearchQuery) return true;
      const query = pageSearchQuery.toLowerCase();
      return (p.title || '').toLowerCase().includes(query) || (p.emoji || '').includes(query);
    });
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

  <!-- Syntax Code Block Card -->
  {:else if card.type === 'code'}
    <div class="code-card-content">
      {#if editingCode}
        <div class="code-toolbar" style="display: flex; gap: 8px; margin-bottom: 8px; align-items: center;">
          <select bind:value={codeLang} on:change={handleSaveCode} style="background: #2a2a2a; color: #818cf8; border: 1px solid #3e3e3e; padding: 4px; border-radius: 4px; font-size: 11px; font-weight: bold;">
            <option value="javascript">JAVASCRIPT</option>
            <option value="python">PYTHON</option>
            <option value="html">HTML</option>
            <option value="css">CSS</option>
            <option value="go">GO</option>
            <option value="dart">DART</option>
            <option value="json">JSON</option>
            <option value="bash">BASH</option>
          </select>
          <span style="flex: 1;"></span>
          <button class="tool-btn done-btn" on:click|stopPropagation={() => { editingCode = false; handleSaveCode(); }}>Done</button>
        </div>
        <textarea
          class="card-textarea"
          bind:value={codeContent}
          on:input={handleSaveCode}
          on:blur={() => { editingCode = false; handleSaveCode(); }}
          placeholder="Write code snippet..."
        ></textarea>
      {:else}
        <div class="code-preview-container" on:dblclick|stopPropagation={() => editingCode = true} style="position: relative; background: #131313; border: 1px solid #2a2a2a; border-radius: 6px; padding: 14px; font-family: monospace; cursor: pointer;">
          <div class="code-lang-tag" style="position: absolute; top: 6px; right: 10px; font-size: 9px; color: #818cf8; font-weight: bold; text-transform: uppercase;">{codeLang}</div>
          <pre style="margin: 0; color: #cbd5e1; font-size: 12px; line-height: 1.5; overflow-x: auto;">{@html getCodeHighlightHtml(codeContent, codeLang)}</pre>
          <span class="empty-hint" style="font-size: 10px; color: #4a4a4a; display: block; margin-top: 6px;">Double-click code block to edit...</span>
        </div>
      {/if}
    </div>

  <!-- HTML Sites Sandbox Card -->
  {:else if card.type === 'sites'}
    <div class="sites-card-preview" on:click|stopPropagation={openSitesModal} style="display: flex; flex-direction: column; gap: 6px; border: 1px dashed #3e3e3e; border-radius: 8px; padding: 12px; cursor: pointer; transition: background 0.15s, border-color 0.15s;">
      <div style="display: flex; align-items: center; gap: 10px;">
        <span style="font-size: 24px;">🌐</span>
        <div style="display: flex; flex-direction: column; flex: 1;">
          <span style="font-weight: bold; color: white; font-size: 14px;">{sitesName}</span>
          <span style="color: #64748b; font-size: 12px;">{sitesDesc}</span>
        </div>
      </div>
      <div style="color: #818cf8; font-size: 11px; font-weight: 600; display: flex; align-items: center; gap: 4px; margin-top: 6px;">
        <span>🔍</span> Click block to open sandboxed workspace HTML local server
      </div>
    </div>

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

{#if showLinkModal}
  <div class="modal-backdrop" on:click|self={() => showLinkModal = false} role="button" tabindex="-1">
    <div class="modal-container">
      <div class="modal-header">
        <h3>Link a Subpage</h3>
        <button class="close-btn" on:click={() => showLinkModal = false}>&times;</button>
      </div>

      <div class="modal-search">
        <span class="search-icon">🔍</span>
        <input
          type="text"
          bind:value={pageSearchQuery}
          placeholder="Search pages..."
          autofocus
        />
        {#if pageSearchQuery}
          <button class="clear-btn" on:click={() => pageSearchQuery = ''}>&times;</button>
        {/if}
      </div>

      <div class="modal-create-action-row" style="padding: 10px 16px; border-bottom: 1px solid #2e2e2e; display: flex;">
        <button class="btn primary" style="width: 100%; display: flex; align-items: center; justify-content: center; gap: 6px; font-size: 11px;" on:click|stopPropagation={handleCreateNewPageAndLink}>
          <span>+</span> Create New Page & Link
        </button>
      </div>

      <div class="modal-body">
        {#if filteredCandidates.length === 0}
          <div class="empty-results">
            <span>📭</span>
            <p>{pageSearchQuery ? 'No matching pages' : 'No pages available to link'}</p>
          </div>
        {:else}
          <div class="candidates-list">
            {#each filteredCandidates as p}
              <button class="candidate-row" on:click={() => selectPageToLink(p)}>
                <span class="cand-emoji">{p.emoji}</span>
                <span class="cand-title">{p.title || 'Untitled'}</span>
              </button>
            {/each}
          </div>
        {/if}
      </div>
    </div>
  </div>
{/if}

<!-- Modal: HTML Site Sandbox Workspace -->
{#if showSitesModal}
  <div class="modal-backdrop" on:click|self={closeSitesModal} role="button" tabindex="-1">
    <div class="modal-container" style="width: 750px; max-width: 95%; height: 600px; max-height: 90vh; display: flex; flex-direction: column;">
      <div class="modal-header">
        <div style="display: flex; flex-direction: column; gap: 2px;">
          <h3 style="margin:0;">🌐 HTML Site block Sandbox</h3>
          <span style="font-size: 11px; color: #64748b;">Renders templates seamlessly using sandbox environment</span>
        </div>
        <button class="close-btn" on:click={closeSitesModal}>&times;</button>
      </div>

      <div class="sites-tabs" style="display: flex; gap: 4px; background: #121212; border-bottom: 1px solid #2e2e2e; padding: 6px 12px;">
        <button class="emoji-tab-btn" class:active={sitesTab === 'edit'} on:click={() => sitesTab = 'edit'}>
          🛠️ Code Editor
        </button>
        <button class="emoji-tab-btn" class:active={sitesTab === 'preview'} on:click={() => sitesTab = 'preview'}>
          📡 Local Server
        </button>
      </div>

      <div class="modal-body" style="flex:1; overflow-y:auto; padding: 18px;">
        {#if sitesTab === 'edit'}
          <div style="display: flex; flex-direction: column; gap: 14px;">
            <div style="display: flex; flex-direction: column; gap: 4px;">
              <label style="font-size: 10px; font-weight: 700; color: #818cf8; letter-spacing: 0.5px;">SITE NAME</label>
              <input type="text" bind:value={sitesName} on:input={handleSaveSites} placeholder="Enter name of site widget" style="background:#1e1e1e; border: 1px solid #2e2e2e; border-radius: 6px; padding: 8px 12px; color: white;" />
            </div>
            <div style="display: flex; flex-direction: column; gap: 4px;">
              <label style="font-size: 10px; font-weight: 700; color: #818cf8; letter-spacing: 0.5px;">SITE SHORT DESCRIPTION</label>
              <input type="text" bind:value={sitesDesc} on:input={handleSaveSites} placeholder="Renders templates cleanly" style="background:#1e1e1e; border: 1px solid #2e2e2e; border-radius: 6px; padding: 8px 12px; color: white;" />
            </div>
            <div style="display: flex; flex-direction: column; gap: 4px;">
              <label style="font-size: 10px; font-weight: 700; color: #818cf8; letter-spacing: 0.5px;">HTML CODE</label>
              <textarea bind:value={sitesHtml} on:input={handleSaveSites} placeholder="Write HTML tags here..." style="background:#121212; border: 1px solid #2e2e2e; border-radius: 6px; color: #e2e8f0; font-family: monospace; font-size: 13px; padding: 12px; height: 260px; resize: vertical; line-height: 1.5;"></textarea>
            </div>
          </div>
        {:else}
          <div style="width: 100%; height: 100%; display: flex; flex-direction: column; gap: 16px; align-items: center; justify-content: center; padding: 40px 20px; text-align: center;">
            <span style="font-size: 48px;">🌐</span>
            <h3 style="margin: 0; color: white;">Local Sandbox Server Active</h3>
            <p style="color: #64748b; font-size: 13px; text-align: center; max-width: 440px; margin: 0; line-height: 1.5;">
              The HTML code is temporarily served on your local machine. Open the address below in any web browser to view the live rendering.
            </p>

            <div style="display: flex; flex-direction: column; gap: 8px; background: #121212; border: 1px solid #2e2e2e; padding: 12px 18px; border-radius: 6px; width: 100%; max-width: 440px; align-items: center;">
              <span style="font-size: 10px; color: #64748b; font-weight: bold; text-transform: uppercase; letter-spacing: 0.5px;">Local Access Address</span>
              <span style="font-size: 15px; color: #818cf8; font-family: monospace; font-weight: bold; word-break: break-all; margin: 4px 0;">
                {sitesLocalUrl || 'Starting local server...'}
              </span>
            </div>

            <div style="display: flex; gap: 12px; margin-top: 12px;">
              <button class="btn primary" on:click={() => { if (sitesLocalUrl) window.open(sitesLocalUrl, '_blank'); }}>
                Open in Browser
              </button>
              <button class="btn secondary" on:click={() => { if (sitesLocalUrl) { navigator.clipboard.writeText(sitesLocalUrl); alert('Copied to clipboard!'); } }}>
                Copy Address
              </button>
            </div>
          </div>
        {/if}
      </div>
    </div>
  </div>
{/if}

<style>
  /* Keep existing styles intact */
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

  /* Modal Backdrop */
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

  /* Modal Container */
  .modal-container {
    background: #1a1a1a;
    border: 1px solid #2e2e2e;
    border-radius: 12px;
    width: 420px;
    max-width: 90%;
    max-height: 400px;
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

  /* Modal Header */
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

  /* Search input */
  .modal-search {
    padding: 12px 16px;
    display: flex;
    align-items: center;
    gap: 8px;
    background: #121212;
    border-bottom: 1px solid #2e2e2e;
    position: relative;
  }

  .search-icon {
    font-size: 14px;
    color: #64748b;
  }

  .modal-search input {
    flex: 1;
    background: transparent;
    border: none;
    color: #e2e8f0;
    font-size: 13px;
    outline: none;
    padding: 4px 0;
  }

  .modal-search input::placeholder {
    color: #4a4a4a;
  }

  .clear-btn {
    background: transparent;
    border: none;
    color: #64748b;
    font-size: 14px;
    cursor: pointer;
    padding: 2px 6px;
    border-radius: 50%;
  }

  .clear-btn:hover {
    color: #cbd5e1;
  }

  /* Modal Body */
  .modal-body {
    flex: 1;
    overflow-y: auto;
    padding: 8px;
  }

  .candidates-list {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .candidate-row {
    display: flex;
    align-items: center;
    gap: 10px;
    background: transparent;
    border: none;
    text-align: left;
    color: #e2e8f0;
    padding: 8px 12px;
    cursor: pointer;
    border-radius: 6px;
    transition: background 0.12s ease;
    width: 100%;
  }

  .candidate-row:hover {
    background: rgba(129, 140, 248, 0.08);
    color: #a5b4fc;
  }

  .cand-emoji {
    font-size: 16px;
    flex-shrink: 0;
  }

  .cand-title {
    font-size: 13px;
    font-weight: 500;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .empty-results {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 32px 16px;
    color: #64748b;
  }

  .empty-results span {
    font-size: 24px;
    margin-bottom: 8px;
  }

  .empty-results p {
    font-size: 12px;
    margin: 0;
  }

  /* Code Card specific hover */
  .sites-card-preview:hover { background: rgba(129, 140, 248, 0.05); border-color: #818cf8; }

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
</style>