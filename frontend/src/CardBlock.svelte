<script>
  import { createEventDispatcher } from "svelte";
  import { marked } from "marked";
  import {
    UpdateCard,
    DeleteCard,
    SaveImage,
    GetImage,
    AddPage,
  } from "../wailsjs/go/main/App.js";
  import PageIcon from "./PageIcon.svelte";

  export let card;
  export let isSelected = false;
  export let index = 0;
  export let allPages = [];
  export let onSelect;
  export let onDeleted;
  export let onNavigate;

  export let activeLiveCardId = null;
  export let activeSitesLocalUrl = "";

  const dispatch = createEventDispatcher();

  let codeLang = "javascript";
  let codeContent = "";

  let sitesName = "My HTML Site";
  let sitesDesc = "Renders customized HTML template preview";
  let sitesHtml =
    "<h1>Sample Site</h1>\n<p>Edit HTML and watch it render live in the preview tab.</p>";

  $: isSitesLive = activeLiveCardId === card.id;
  $: sitesLocalUrl = activeLiveCardId === card.id ? activeSitesLocalUrl : "";

  $: if (card.type === "code") {
    const raw = card.content || "";
    if (raw.includes("\n")) {
      const firstLineIdx = raw.indexOf("\n");
      codeLang = raw.substring(0, firstLineIdx).trim().toLowerCase();
      codeContent = raw.substring(firstLineIdx + 1);
    } else {
      codeLang = "javascript";
      codeContent = raw;
    }
  }

  $: if (card.type === "sites") {
    try {
      const parsed = JSON.parse(card.content || "{}");
      sitesName = parsed.name || "My HTML Site";
      sitesDesc =
        parsed.description || "Renders customized HTML template preview";
      sitesHtml =
        parsed.html ||
        "<h1>Sample Site</h1>\n<p>Edit HTML and watch it render live in the preview tab.</p>";
    } catch (_) {
      sitesName = "My HTML Site";
      sitesDesc = "Renders customized HTML template preview";
      sitesHtml = card.content || "";
    }
  }

  function handleClick() {
    if (onSelect) onSelect(card);
  }

  function startEdit() {
    if (card.type === "markdown") {
      dispatch("editMarkdown", { content: card.content || "" });
    } else if (card.type === "section") {
      const newTitle = prompt("Edit Section Header:", card.content || "");
      if (newTitle !== null) {
        UpdateCard(card.id, card.page_id, newTitle.trim(), card.comment || "")
          .then(() => { card.content = newTitle.trim(); })
          .catch((err) => console.error(err));
      }
    }
  }

  function deleteCard(e) {
    e.stopPropagation();
    if (confirm("Delete this card?")) {
      DeleteCard(card.id, card.page_id)
        .then(() => {
          if (onDeleted) onDeleted(card.id);
        })
        .catch((err) => {
          console.error("Card delete failed:", err);
        });
    }
  }

  let imageInput;

  function setImage() {
    const choice = prompt("Image source:\n1: Enter URL\n2: Upload file");
    if (choice === "1") {
      const url = prompt("Enter image URL:", card.content || "");
      if (url !== null) {
        UpdateCard(card.id, card.page_id, url, card.comment || "")
          .then(() => {
            card.content = url;
          })
          .catch((err) => console.error(err));
      }
    } else if (choice === "2") {
      imageInput.click();
    }
  }

  function handleFileUpload(e) {
    const file = e.target.files[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = () => {
      const base64 = reader.result;
      SaveImage(base64, file.name)
        .then((filename) => {
          UpdateCard(card.id, card.page_id, filename, card.comment || "")
            .then(() => {
              card.content = filename;
            })
            .catch((err) => console.error(err));
        })
        .catch((err) => alert("Failed to save image: " + err));
    };
    reader.readAsDataURL(file);
    e.target.value = "";
  }

  function unlinkImage(e) {
    e.stopPropagation();
    if (confirm("Remove image?")) {
      UpdateCard(card.id, card.page_id, "", card.comment || "")
        .then(() => {
          card.content = "";
        })
        .catch((err) => {
          console.error("Card save failed:", err);
        });
    }
  }

  function selectSubpage(e) {
    e.stopPropagation();
    dispatch("openLinkModal", { card });
  }

  function navigateToSubpage(e) {
    e.stopPropagation();
    if (!card.content || !onNavigate) return;
    const target = allPages.find((p) => p.id === card.content);
    if (target) onNavigate(target);
  }

  function openSitesModal() {
    dispatch("editSites", {
      name: sitesName,
      description: sitesDesc,
      html: sitesHtml,
    });
  }

  function openLivePreviewModal() {
    dispatch("previewSites", {
      name: sitesName,
      html: sitesHtml,
    });
  }

  function getCodeHighlightHtml(code, lang) {
    if (!code)
      return '<span style="color: #64748b; font-style: italic;">// Empty code block...</span>';
    const escaped = code
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;");
    const keywords = [
      "function", "return", "if", "else", "for", "while", "const", "let", "var",
      "import", "class", "void", "final", "def", "package", "func", "interface",
      "fn", "mut", "match", "impl", "struct", "enum", "pub", "use", "mod", "as", "type"
    ];
    const keywordsRegex = new RegExp(`\\b(${keywords.join("|")})\\b`, "g");
    return escaped
      .replace(
        keywordsRegex,
        '<span style="color: #f472b6; font-weight: bold;">$1</span>',
      )
      .replace(
        /("(.*?)"|'([^']*)')/g,
        '<span style="color: #34d399;">$1</span>',
      )
      .replace(
        /(\/\/[^\n]*|#[^\n]*)/g,
        '<span style="color: #94a3b8; font-style: italic;">$1</span>',
      );
  }

  $: linkedPage = (() => {
    if (card.type !== "subpage_link" || !card.content) return null;
    return allPages.find((p) => p.id === card.content) || null;
  })();

  $: renderedMarkdown = marked.parse(card.content || "");

  let showComment = false;
  let editingCommentId = null;
  let commentInputVal = "";

  $: commentsList = (() => {
    if (!card.comment) return [];
    try {
      const parsed = JSON.parse(card.comment);
      return Array.isArray(parsed) ? parsed : [];
    } catch (_) {
      return [];
    }
  })();

  function genId() {
    return Math.random().toString(36).substring(2, 14);
  }

  function saveCommentsList(list) {
    card.comment = JSON.stringify(list);
    UpdateCard(card.id, card.page_id, card.content || "", card.comment)
      .then(() => { card = card; })
      .catch((err) => console.error(err));
  }

  function toggleComment() {
    showComment = !showComment;
    editingCommentId = null;
    commentInputVal = "";
  }

  function startNewComment() {
    editingCommentId = "new";
    commentInputVal = "";
  }

  function startEditComment(id, text) {
    editingCommentId = id;
    commentInputVal = text || "";
  }

  function saveComment() {
    const text = commentInputVal.trim();
    if (!text) { editingCommentId = null; return; }
    const list = [...commentsList];
    if (editingCommentId === "new") {
      list.unshift({ id: genId(), text, createdAt: new Date().toISOString() });
    } else if (editingCommentId) {
      const idx = list.findIndex((c) => c.id === editingCommentId);
      if (idx !== -1) list[idx].text = text;
    }
    editingCommentId = null;
    saveCommentsList(list);
  }

  function handleCommentKeydown(e) {
    if (e.key === "Enter") {
      saveComment();
    } else if (e.key === "Escape") {
      editingCommentId = null;
      if (!commentsList.length) showComment = false;
    }
  }

  function deleteComment(id) {
    const list = commentsList.filter((c) => c.id !== id);
    editingCommentId = null;
    if (!list.length) showComment = false;
    saveCommentsList(list);
  }
</script>

<div
  class="card-block {card.type}"
  class:selected={isSelected}
  on:click={handleClick}
  role="button"
  tabindex="0"
>
  {#if index > 0}
    <div class="card-order-badge">#{index}</div>
  {/if}

  {#if card.type === "section"}
    <div class="section-card-content" on:dblclick={startEdit}>
      <div class="section-header-wrap">
        <PageIcon emoji="heading" size={14} />
        <h2 class="section-title-text">{card.content || "Untitled Section"}</h2>
        <div class="section-controls">
          <button class="section-ctrl-btn" title="Move up" on:click|stopPropagation={() => dispatch("moveUp")}>
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m18 15-6-6-6 6"/></svg>
          </button>
          <button class="section-ctrl-btn" title="Move down" on:click|stopPropagation={() => dispatch("moveDown")}>
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m6 9 6 6 6-6"/></svg>
          </button>
          <button class="section-ctrl-btn delete" title="Delete" on:click|stopPropagation={() => { DeleteCard(card.id, card.page_id); onDeleted(card.id); }}>
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 6h18"/><path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"/><path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"/></svg>
          </button>
        </div>
      </div>
      <div class="section-divider-line"></div>
    </div>

  {:else if card.type === "markdown"}
    <div class="card-preview markdown-rendered" on:dblclick={startEdit}>
      {#if card.content}
        {@html renderedMarkdown}
      {:else}
        <span class="empty-hint">Double-click to write in fullscreen...</span>
      {/if}
    </div>

  {:else if card.type === "image"}
    <div class="image-card-content">
      <input
        type="file"
        bind:this={imageInput}
        accept="image/*"
        style="display:none"
        on:change={handleFileUpload}
      />
      {#if card.content}
        <div class="image-wrapper">
          <img
            src={card.content}
            alt="Card visual"
            class="card-image"
            on:error={(e) => (e.target.style.display = "none")}
          />
        </div>
        <div class="image-actions">
          <button class="img-action-btn" on:click|stopPropagation={setImage}>Change</button>
          <button class="img-action-btn danger" on:click|stopPropagation={unlinkImage}>Remove</button>
        </div>
      {:else}
        <div class="image-placeholder" on:click|stopPropagation={setImage}>
          <PageIcon emoji="image" size={18} />
          <span class="placeholder-text">Click to add image (URL or file)</span>
        </div>
      {/if}
    </div>

  {:else if card.type === "subpage_link"}
    {#if linkedPage}
      <div class="subpage-link-card" on:click|stopPropagation={navigateToSubpage}>
        <span class="link-emoji"><PageIcon emoji={linkedPage.emoji} size={14} /></span>
        <div class="link-info">
          <span class="link-title">{linkedPage.title || "Untitled"}</span>
          <span class="link-hint">Open subpage →</span>
        </div>
        <button class="link-action-btn" on:click|stopPropagation={selectSubpage}>Change</button>
      </div>
    {:else}
      <div class="subpage-empty" on:click|stopPropagation={selectSubpage}>
        <PageIcon emoji="link" size={14} />
        <span>Click to link a subpage...</span>
      </div>
    {/if}

  {:else if card.type === "code"}
    <div class="code-card-content">
      <div
        class="code-preview-container"
        on:dblclick|stopPropagation={() =>
          dispatch("editCode", { content: card.content || "" })}
        style="position: relative; background: #0c0c0e; border: 1px solid #1c1c1f; border-radius: 6px; padding: 14px; font-family: monospace; cursor: pointer;"
      >
        <div
          class="code-lang-tag"
          style="position: absolute; top: 6px; right: 10px; font-size: 9px; color: #818cf8; font-weight: bold; text-transform: uppercase;"
        >
          {codeLang}
        </div>
        <pre style="margin: 0; color: #cbd5e1; font-size: 12px; line-height: 1.5; overflow-x: auto;">{@html getCodeHighlightHtml(codeContent, codeLang)}</pre>
        <span class="empty-hint" style="font-size: 10px; color: #4a4a4a; display: block; margin-top: 6px;">Double-click code block to edit in immersive fullscreen...</span>
      </div>
    </div>

  {:else if card.type === "sites"}
    <div class="sites-card">
      <div class="sites-card-inner">
        <div class="sites-card-icon-container" on:click|stopPropagation={openLivePreviewModal}>
          <PageIcon emoji="globe" size={14} />
        </div>
        <div class="sites-card-details" on:click|stopPropagation={openLivePreviewModal}>
          <span class="sites-card-name">{sitesName}</span>
          <span class="sites-card-desc">
            {isSitesLive && sitesLocalUrl ? `Serving on ${sitesLocalUrl}` : sitesDesc}
          </span>
        </div>
        <div class="sites-card-actions" style="display: flex; gap: 6px; align-items: center; flex-shrink: 0;">
          <button
            class="img-action-btn"
            style="color: #10b981; border-color: rgba(16, 185, 129, 0.15); background: rgba(16, 185, 129, 0.02);"
            on:click|stopPropagation={openLivePreviewModal}
          >
            ▶ Preview
          </button>
          <button
            class="img-action-btn"
            style="color: #818cf8; border-color: rgba(129, 140, 248, 0.15); background: rgba(129, 140, 248, 0.02);"
            on:click|stopPropagation={openSitesModal}
          >
            ⚙ Config
          </button>
        </div>
      </div>
    </div>

  {:else}
    <div class="file-card">
      <span>📎 {card.content || "(attach file)"}</span>
    </div>
  {/if}

  {#if card.type !== "section"}
    <div class="card-comment-section">
      <button
        class="comment-toggle-btn"
        on:click|stopPropagation={toggleComment}
        class:has-comment={commentsList.length > 0}
      >
        <span class="comment-toggle-icon">💬</span>
        <span class="comment-toggle-label">
          {commentsList.length > 0 ? `${commentsList.length} Comment${commentsList.length > 1 ? 's' : ''}` : "Add comment"}
        </span>
        <span class="comment-toggle-arrow">{showComment ? "▲" : "▼"}</span>
      </button>

      {#if showComment}
        <div class="comment-dropdown">
          {#each commentsList as c}
            <div class="comment-row">
              {#if editingCommentId === c.id}
                <div class="comment-input-wrap">
                  <span class="comment-bubble-icon">💬</span>
                  <input
                    type="text"
                    class="comment-input"
                    bind:value={commentInputVal}
                    on:blur={saveComment}
                    on:keydown={handleCommentKeydown}
                    placeholder="Write a comment..."
                    autofocus
                  />
                </div>
              {:else}
                <span class="comment-bubble-icon">💬</span>
                <span class="comment-text" on:dblclick|stopPropagation={() => startEditComment(c.id, c.text)}>
                  {c.text}
                </span>
                <button class="comment-edit-btn" on:click|stopPropagation={() => startEditComment(c.id, c.text)}>edit</button>
                <button class="comment-delete-btn" on:click|stopPropagation={() => deleteComment(c.id)}>&times;</button>
              {/if}
            </div>
          {/each}
          {#if editingCommentId === "new"}
            <div class="comment-row">
              <div class="comment-input-wrap">
                <span class="comment-bubble-icon">💬</span>
                <input
                  type="text"
                  class="comment-input"
                  bind:value={commentInputVal}
                  on:blur={saveComment}
                  on:keydown={handleCommentKeydown}
                  placeholder="Write a comment..."
                  autofocus
                />
              </div>
            </div>
          {:else if editingCommentId === null}
            <button class="add-comment-trigger" on:click|stopPropagation={startNewComment}>
              + Add comment
            </button>
          {/if}
        </div>
      {/if}
    </div>
  {/if}

  {#if isSelected}
    <div class="card-actions">
      <button class="action-btn" title="Delete card" on:click={deleteCard}>×</button>
    </div>
  {/if}
</div>

<style>
  .card-block {
    position: relative;
    background: #09090b;
    border: 1px solid rgba(255, 255, 255, 0.04);
    border-radius: 8px;
    padding: 14px;
    cursor: pointer;
    transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
    text-align: left;
  }

  .card-block.markdown { border-left: 3px solid #71717a; }
  .card-block.image { border-left: 3px solid #a855f7; }
  .card-block.subpage_link { border-left: 3px solid #3b82f6; }
  .card-block.code { border-left: 3px solid #ec4899; }
  .card-block.sites { border-left: 3px solid #10b981; }
  .card-block.section {
    border: none;
    background: transparent;
    padding: 8px 0;
    margin: 16px 0 8px 0;
    cursor: default;
  }
  .card-block.section:hover { background: transparent; }

  .section-card-content { width: 100%; user-select: none; }
  .section-header-wrap {
    display: flex;
    align-items: center;
    gap: 8px;
    color: #a1a1aa;
    margin-bottom: 4px;
  }
  .section-controls {
    display: flex;
    gap: 2px;
    margin-left: auto;
    opacity: 0;
    transition: opacity 0.15s ease;
  }
  .card-block.section:hover .section-controls { opacity: 1; }
  .section-ctrl-btn {
    background: none;
    border: none;
    padding: 2px 4px;
    cursor: pointer;
    color: #71717a;
    border-radius: 4px;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: color 0.15s ease, background 0.15s ease;
  }
  .section-ctrl-btn:hover { color: #e4e4e7; background: rgba(255,255,255,0.06); }
  .section-ctrl-btn.delete:hover { color: #f87171; }
  .section-title-text {
    font-size: 15px;
    font-weight: 800;
    color: #f4f4f5;
    text-transform: uppercase;
    letter-spacing: 1px;
    margin: 0;
  }
  .section-divider-line {
    height: 1px;
    width: 100%;
    background: rgba(255, 255, 255, 0.08);
    margin-top: 4px;
  }

  .card-order-badge {
    position: absolute;
    top: 6px;
    left: 6px;
    background: rgba(129, 140, 248, 0.1);
    color: #818cf8;
    font-size: 9px;
    font-weight: 700;
    padding: 1px 5px;
    border-radius: 4px;
    line-height: 1.4;
    z-index: 1;
    pointer-events: none;
  }
  .card-block:hover {
    border-color: rgba(255, 255, 255, 0.08);
    background: rgba(255, 255, 255, 0.01);
  }
  .card-block.selected {
    border-color: rgba(129, 140, 248, 0.5);
    box-shadow: 0 0 0 1px rgba(129, 140, 248, 0.15);
  }

  .card-preview {
    min-height: 20px;
    font-size: 13px;
    line-height: 1.6;
    color: #d4d4d8;
    text-align: left;
  }
  .card-preview:hover {
    background: rgba(255, 255, 255, 0.01);
    border-radius: 4px;
  }
  .empty-hint { color: #52525b; font-style: italic; font-size: 12px; }

  .image-card-content { display: flex; flex-direction: column; gap: 8px; }
  .image-wrapper { display: flex; justify-content: center; border-radius: 6px; overflow: hidden; background: rgba(0, 0, 0, 0.1); }
  .card-image { max-width: 100%; max-height: 320px; object-fit: contain; }
  .image-placeholder {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 20px;
    border: 1px dashed rgba(255, 255, 255, 0.08);
    border-radius: 6px;
    cursor: pointer;
    transition: all 0.15s ease;
    background: rgba(255, 255, 255, 0.005);
    color: #71717a;
    gap: 6px;
  }
  .image-placeholder:hover { border-color: rgba(129, 140, 248, 0.3); background: rgba(129, 140, 248, 0.01); color: #818cf8; }
  .placeholder-text { font-size: 11px; font-weight: 500; }
  .image-actions { display: flex; gap: 6px; justify-content: flex-end; }
  .img-action-btn {
    background: rgba(255, 255, 255, 0.02);
    border: 1px solid rgba(255, 255, 255, 0.05);
    color: #a1a1aa;
    font-size: 10px;
    font-weight: 600;
    padding: 4px 10px;
    border-radius: 6px;
    cursor: pointer;
    transition: all 0.15s;
  }
  .img-action-btn:hover { background: rgba(255, 255, 255, 0.06); color: #f4f4f5; }
  .img-action-btn.danger:hover { background: rgba(239, 68, 68, 0.08); border-color: rgba(239, 68, 68, 0.2); color: #f87171; }

  .subpage-link-card {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 6px 10px;
    border-radius: 6px;
    cursor: pointer;
    background: rgba(255, 255, 255, 0.01);
    border: 1px solid rgba(255, 255, 255, 0.03);
    transition: all 0.15s ease;
  }
  .subpage-link-card:hover { background: rgba(255, 255, 255, 0.03); border-color: rgba(255, 255, 255, 0.06); }
  .link-info { flex: 1; display: flex; flex-direction: column; min-width: 0; }
  .link-title { color: #e4e4e7; font-weight: 600; font-size: 12px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .link-hint { color: #52525b; font-size: 10px; margin-top: 1px; }
  .link-action-btn {
    background: rgba(255, 255, 255, 0.02);
    border: 1px solid rgba(255, 255, 255, 0.05);
    color: #a1a1aa;
    font-size: 10px;
    font-weight: 600;
    padding: 4px 8px;
    border-radius: 6px;
    cursor: pointer;
    transition: all 0.15s;
  }
  .link-action-btn:hover { background: rgba(255, 255, 255, 0.06); color: #f4f4f5; }

  .subpage-empty {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 10px 14px;
    border: 1px dashed rgba(255, 255, 255, 0.06);
    border-radius: 6px;
    cursor: pointer;
    color: #52525b;
    font-size: 12px;
    transition: all 0.15s ease;
  }
  .subpage-empty:hover { border-color: rgba(129, 140, 248, 0.3); color: #818cf8; }

  .file-card { color: #a1a1aa; font-size: 12px; }

  .card-actions {
    position: absolute;
    top: 10px;
    right: 10px;
    display: flex;
    gap: 4px;
  }
  .action-btn {
    background: rgba(255, 255, 255, 0.03);
    border: 1px solid rgba(255, 255, 255, 0.06);
    color: #71717a;
    cursor: pointer;
    font-size: 11px;
    width: 18px;
    height: 18px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 4px;
    transition: all 0.12s;
  }
  .action-btn:hover { background: rgba(239, 68, 68, 0.08); border-color: rgba(239, 68, 68, 0.2); color: #f87171; }

  .markdown-rendered :global(h1) { font-size: 16px; margin: 12px 0 6px 0; color: #ffffff; font-weight: 700; }
  .markdown-rendered :global(h2) { font-size: 14px; margin: 10px 0 4px 0; color: #ffffff; font-weight: 700; }
  .markdown-rendered :global(h3) { font-size: 12px; margin: 8px 0 2px 0; color: #ffffff; font-weight: 700; }
  .markdown-rendered :global(code) { background: rgba(255, 255, 255, 0.03); padding: 2px 4px; border-radius: 4px; font-size: 11px; font-family: monospace; color: #f472b6; }
  .markdown-rendered :global(pre) { background: rgba(0, 0, 0, 0.2); padding: 10px; border-radius: 6px; overflow-x: auto; border: 1px solid rgba(255, 255, 255, 0.03); }
  .markdown-rendered :global(ul), .markdown-rendered :global(ol) { padding-left: 16px; margin: 6px 0; }
  .markdown-rendered :global(li) { margin-bottom: 2px; }

  .sites-card {
    background: rgba(255, 255, 255, 0.005);
    border: 1px solid rgba(255, 255, 255, 0.04);
    border-radius: 8px;
    padding: 12px;
    cursor: pointer;
    transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
  }
  .sites-card:hover { border-color: rgba(129, 140, 248, 0.3); background: rgba(129, 140, 248, 0.01); }
  .sites-card-inner { display: flex; align-items: center; gap: 12px; width: 100%; }
  .sites-card-icon-container {
    width: 32px; height: 32px;
    display: flex; align-items: center; justify-content: center;
    background: rgba(255, 255, 255, 0.02);
    border: 1px solid rgba(255, 255, 255, 0.05);
    border-radius: 6px;
    transition: all 0.2s ease;
    flex-shrink: 0;
  }
  .sites-card:hover .sites-card-icon-container { background: rgba(129, 140, 248, 0.08); border-color: rgba(129, 140, 248, 0.2); }
  .sites-card-details { display: flex; flex-direction: column; flex: 1; min-width: 0; }
  .sites-card-name { font-weight: 600; color: #ffffff; font-size: 12px; margin-bottom: 1px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .sites-card-desc { color: #71717a; font-size: 11px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .sites-card:hover .sites-card-desc { color: #a1a1aa; }
  .sites-card-status { display: flex; align-items: center; justify-content: center; padding-left: 6px; flex-shrink: 0; }
  .status-indicator-dot {
    width: 6px; height: 6px; border-radius: 50%;
    background: #52525b;
    transition: all 0.3s ease;
  }
  .status-indicator-dot.live {
    background: #10b981;
    box-shadow: 0 0 8px rgba(16, 185, 129, 0.5);
    animation: pulse-active-dot 2s infinite;
  }
  @keyframes pulse-active-dot {
    0% { box-shadow: 0 0 4px rgba(16, 185, 129, 0.3); }
    50% { box-shadow: 0 0 8px rgba(16, 185, 129, 0.6); }
    100% { box-shadow: 0 0 4px rgba(16, 185, 129, 0.3); }
  }

  .card-comment-section { margin-top: 8px; display: flex; flex-direction: column; width: 100%; }
  .comment-toggle-btn {
    display: flex; align-items: center; gap: 6px;
    background: transparent; border: 1px solid rgba(255, 255, 255, 0.04);
    border-radius: 6px; color: #52525b; font-size: 10px; font-weight: 600;
    cursor: pointer; padding: 5px 10px; transition: all 0.12s; align-self: flex-start;
  }
  .comment-toggle-btn:hover { border-color: rgba(129, 140, 248, 0.2); color: #818cf8; background: rgba(129, 140, 248, 0.03); }
  .comment-toggle-btn.has-comment { color: #818cf8; border-color: rgba(129, 140, 248, 0.15); background: rgba(129, 140, 248, 0.03); }
  .comment-toggle-icon { font-size: 10px; }
  .comment-toggle-arrow { font-size: 7px; margin-left: 2px; }
  .comment-dropdown {
    margin-top: 6px; display: flex; flex-direction: column; width: 100%;
    background: rgba(255, 255, 255, 0.015); border: 1px solid rgba(255, 255, 255, 0.04);
    border-radius: 6px; padding: 6px 10px; gap: 2px;
  }
  .comment-row { display: flex; align-items: flex-start; gap: 6px; padding: 3px 0; }
  .comment-row:hover .comment-edit-btn, .comment-row:hover .comment-delete-btn { opacity: 1; }
  .comment-bubble-icon { font-size: 10px; color: #818cf8; opacity: 0.8; flex-shrink: 0; margin-top: 2px; }
  .comment-text { flex: 1; cursor: text; line-height: 1.4; word-break: break-word; color: #a1a1aa; font-size: 11px; }
  .comment-input-wrap { display: flex; align-items: center; gap: 6px; flex: 1; }
  .comment-input { flex: 1; background: transparent !important; border: none !important; outline: none !important; color: #f4f4f5 !important; font-size: 11px !important; padding: 0 !important; margin-bottom: 0 !important; }
  .comment-edit-btn, .comment-delete-btn { background: transparent; border: none; cursor: pointer; font-size: 9px; font-weight: 600; color: #52525b; padding: 2px 4px; border-radius: 4px; transition: all 0.12s; opacity: 0; }
  .comment-edit-btn:hover { color: #818cf8; background: rgba(129, 140, 248, 0.08); }
  .comment-delete-btn:hover { color: #f87171; background: rgba(239, 68, 68, 0.08); }
  .add-comment-trigger { background: transparent; border: none; color: #818cf8; font-size: 10px; font-weight: 600; cursor: pointer; padding: 4px 0; text-align: left; transition: all 0.12s; }
  .add-comment-trigger:hover { color: #a5b4fc; }
</style>
