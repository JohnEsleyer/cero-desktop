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

  export let card;
  export let isSelected = false;
  export let allPages = [];
  export let onSelect;
  export let onDeleted;
  export let onNavigate;

  // External live-serving tracking props passed from App.svelte
  export let activeLiveCardId = null;
  export let activeSitesLocalUrl = "";

  const dispatch = createEventDispatcher();

  let showLinkModal = false;
  let pageSearchQuery = "";

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
    if (card.type !== "markdown") return;
    dispatch("editMarkdown", { content: card.content || "" });
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
        UpdateCard(card.id, card.page_id, url)
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
          UpdateCard(card.id, card.page_id, filename)
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
      UpdateCard(card.id, card.page_id, "")
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
    pageSearchQuery = "";
    showLinkModal = true;
  }

  function selectPageToLink(target) {
    showLinkModal = false;
    UpdateCard(card.id, card.page_id, target.id)
      .then(() => {
        card.content = target.id;
      })
      .catch((err) => console.error(err));
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

  function getCodeHighlightHtml(code, lang) {
    if (!code)
      return '<span style="color: #64748b; font-style: italic;">// Empty code block...</span>';
    const escaped = code
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;");
    const keywords = [
      "function",
      "return",
      "if",
      "else",
      "for",
      "while",
      "const",
      "let",
      "var",
      "import",
      "class",
      "void",
      "final",
      "def",
      "package",
      "func",
      "interface",
    ];
    const keywordsRegex = new RegExp(`\\b(${keywords.join("|")})\\b`, "g");
    return escaped
      .replace(
        keywordsRegex,
        '<span style="color: #f472b6; font-weight: bold;">$1</span>',
      )
      .replace(
        /("(.*?)"|\'([^\']*)\')/g,
        '<span style="color: #34d399;">$1</span>',
      )
      .replace(
        /(\/\/[^\n]*)/g,
        '<span style="color: #94a3b8; font-style: italic;">$1</span>',
      );
  }

  async function handleCreateNewPageAndLink() {
    showLinkModal = false;
    try {
      const res = await AddPage(
        card.page_id,
        "subpage",
        "New Subpage Link",
        "📝",
      );
      if (res && res.id) {
        await UpdateCard(card.id, card.page_id, res.id);
        card.content = res.id;
      }
    } catch (err) {
      console.error(err);
    }
  }

  $: linkedPage = (() => {
    if (card.type !== "subpage_link" || !card.content) return null;
    return allPages.find((p) => p.id === card.content) || null;
  })();

  $: renderedMarkdown = marked.parse(card.content || "");

  $: filteredCandidates = allPages
    .filter((p) => p.id !== card.page_id && p.relation_type !== "sidepage")
    .filter((p) => {
      if (!pageSearchQuery) return true;
      const query = pageSearchQuery.toLowerCase();
      return (
        (p.title || "").toLowerCase().includes(query) ||
        (p.emoji || "").includes(query)
      );
    });
</script>

<div
  class="card-block {card.type}"
  class:selected={isSelected}
  on:click={handleClick}
  role="button"
  tabindex="0"
>
  <!-- Markdown Card -->
  {#if card.type === "markdown"}
    <!-- svelte-ignore a11y-no-static-element-interactions -->
    <div class="card-preview markdown-rendered" on:dblclick={startEdit}>
      {#if card.content}
        {@html renderedMarkdown}
      {:else}
        <span class="empty-hint">Double-click to write in fullscreen...</span>
      {/if}
    </div>

    <!-- Image Card -->
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
          <button class="img-action-btn" on:click|stopPropagation={setImage}
            >Change</button
          >
          <button
            class="img-action-btn danger"
            on:click|stopPropagation={unlinkImage}>Remove</button
          >
        </div>
      {:else}
        <!-- svelte-ignore a11y-no-static-element-interactions -->
        <div class="image-placeholder" on:click|stopPropagation={setImage}>
          <span class="placeholder-icon">🖼️</span>
          <span class="placeholder-text">Click to add image (URL or file)</span>
        </div>
      {/if}
    </div>

    <!-- Subpage Link Card -->
  {:else if card.type === "subpage_link"}
    {#if linkedPage}
      <!-- svelte-ignore a11y-no-static-element-interactions -->
      <div
        class="subpage-link-card"
        on:click|stopPropagation={navigateToSubpage}
      >
        <span class="link-emoji">{linkedPage.emoji}</span>
        <div class="link-info">
          <span class="link-title">{linkedPage.title || "Untitled"}</span>
          <span class="link-hint">Open subpage →</span>
        </div>
        <button class="link-action-btn" on:click|stopPropagation={selectSubpage}
          >Change</button
        >
      </div>
    {:else}
      <!-- svelte-ignore a11y-no-static-element-interactions -->
      <div class="subpage-empty" on:click|stopPropagation={selectSubpage}>
        <span>🔗</span>
        <span>Click to link a subpage...</span>
      </div>
    {/if}

    <!-- Syntax Code Block Card -->
  {:else if card.type === "code"}
    <div class="code-card-content">
      <!-- svelte-ignore a11y-no-static-element-interactions -->
      <div
        class="code-preview-container"
        on:dblclick|stopPropagation={() =>
          dispatch("editCode", { content: card.content || "" })}
        style="position: relative; background: #131313; border: 1px solid #2a2a2a; border-radius: 6px; padding: 14px; font-family: monospace; cursor: pointer;"
      >
        <div
          class="code-lang-tag"
          style="position: absolute; top: 6px; right: 10px; font-size: 9px; color: #818cf8; font-weight: bold; text-transform: uppercase;"
        >
          {codeLang}
        </div>
        <pre
          style="margin: 0; color: #cbd5e1; font-size: 12px; line-height: 1.5; overflow-x: auto;">{@html getCodeHighlightHtml(
            codeContent,
            codeLang,
          )}</pre>
        <span
          class="empty-hint"
          style="font-size: 10px; color: #4a4a4a; display: block; margin-top: 6px;"
          >Double-click code block to edit in immersive fullscreen...</span
        >
      </div>
    </div>

    <!-- HTML Sites Sandbox Card -->
  {:else if card.type === "sites"}
    <!-- svelte-ignore a11y-no-static-element-interactions -->
    <div class="sites-card" on:click|stopPropagation={openSitesModal}>
      <div class="sites-card-inner">
        <div class="sites-card-icon-container">
          <span class="sites-emoji">🌐</span>
        </div>
        <div class="sites-card-details">
          <span class="sites-card-name">{sitesName}</span>
          <span class="sites-card-desc">
            {isSitesLive && sitesLocalUrl
              ? `Serving on ${sitesLocalUrl}`
              : sitesDesc}
          </span>
        </div>
        <div class="sites-card-status">
          <span class="status-indicator-dot" class:live={isSitesLive}></span>
        </div>
      </div>
    </div>

    <!-- File Card -->
  {:else}
    <div class="file-card">
      <span>📎 {card.content || "(attach file)"}</span>
    </div>
  {/if}

  <!-- Actions (visible when selected) -->
  {#if isSelected}
    <div class="card-actions">
      <button class="action-btn" title="Delete card" on:click={deleteCard}
        >×</button
      >
    </div>
  {/if}
</div>

{#if showLinkModal}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
  <div
    class="modal-backdrop"
    on:click|self={() => (showLinkModal = false)}
    role="button"
    tabindex="-1"
  >
    <div class="modal-container">
      <div class="modal-header">
        <h3>Link a Subpage</h3>
        <button class="close-btn" on:click={() => (showLinkModal = false)}
          >&times;</button
        >
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
          <button class="clear-btn" on:click={() => (pageSearchQuery = "")}
            >&times;</button
          >
        {/if}
      </div>

      <div
        class="modal-create-action-row"
        style="padding: 10px 16px; border-bottom: 1px solid #2e2e2e; display: flex;"
      >
        <button
          class="btn primary"
          style="width: 100%; display: flex; align-items: center; justify-content: center; gap: 6px; font-size: 11px;"
          on:click|stopPropagation={handleCreateNewPageAndLink}
        >
          <span>+</span> Create New Page & Link
        </button>
      </div>

      <div class="modal-body">
        {#if filteredCandidates.length === 0}
          <div class="empty-results">
            <span>📭</span>
            <p>
              {pageSearchQuery
                ? "No matching pages"
                : "No pages available to link"}
            </p>
          </div>
        {:else}
          <div class="candidates-list">
            {#each filteredCandidates as p}
              <button
                class="candidate-row"
                on:click={() => selectPageToLink(p)}
              >
                <span class="cand-emoji">{p.emoji}</span>
                <span class="cand-title">{p.title || "Untitled"}</span>
              </button>
            {/each}
          </div>
        {/if}
      </div>
    </div>
  </div>
{/if}

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
  .card-block:hover {
    border-color: #4a4a4a;
  }
  .card-block.selected {
    border-color: #818cf8;
  }

  .card-preview {
    min-height: 20px;
    font-size: 13px;
    line-height: 1.6;
    color: #cbd5e1;
  }
  .card-preview:hover {
    background: rgba(255, 255, 255, 0.02);
    border-radius: 4px;
  }
  .empty-hint {
    color: #4a4a4a;
    font-style: italic;
  }

  .image-card-content {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .image-wrapper {
    display: flex;
    justify-content: center;
  }
  .card-image {
    max-width: 100%;
    max-height: 400px;
    border-radius: 4px;
    object-fit: contain;
  }
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
  .image-placeholder:hover {
    border-color: #818cf8;
  }
  .placeholder-icon {
    font-size: 28px;
    margin-bottom: 6px;
  }
  .placeholder-text {
    color: #64748b;
    font-size: 12px;
  }
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
  .img-action-btn:hover {
    background: #3e3e3e;
    color: #e2e8f0;
  }
  .img-action-btn.danger:hover {
    background: rgba(239, 68, 68, 0.15);
    color: #f87171;
  }

  .subpage-link-card {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 8px;
    border-radius: 6px;
    cursor: pointer;
    transition: background 0.12s ease;
  }
  .subpage-link-card:hover {
    background: #252525;
  }
  .link-emoji {
    font-size: 24px;
  }
  .link-info {
    flex: 1;
    display: flex;
    flex-direction: column;
  }
  .link-title {
    color: #e2e8f0;
    font-weight: 600;
    font-size: 14px;
  }
  .link-hint {
    color: #64748b;
    font-size: 11px;
    margin-top: 2px;
  }
  .link-action-btn {
    background: #2a2a2a;
    border: 1px solid #3e3e3e;
    color: #94a3b8;
    font-size: 11px;
    padding: 3px 8px;
    border-radius: 4px;
    cursor: pointer;
  }
  .link-action-btn:hover {
    background: #3e3e3e;
    color: #e2e8f0;
  }

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
  .subpage-empty:hover {
    border-color: #818cf8;
    color: #818cf8;
  }

  .file-card {
    color: #94a3b8;
    font-size: 13px;
  }

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
  .action-btn:hover {
    background: #3e3e3e;
    color: white;
  }

  .markdown-rendered :global(h1) {
    font-size: 18px;
    margin: 16px 0 8px 0;
    color: white;
  }
  .markdown-rendered :global(h2) {
    font-size: 16px;
    margin: 14px 0 6px 0;
    color: white;
  }
  .markdown-rendered :global(h3) {
    font-size: 14px;
    margin: 12px 0 4px 0;
    color: white;
  }
  .markdown-rendered :global(code) {
    background: #1a1a1a;
    padding: 2px 4px;
    border-radius: 4px;
    font-size: 12px;
  }
  .markdown-rendered :global(pre) {
    background: #1a1a1a;
    padding: 10px;
    border-radius: 6px;
    overflow-x: auto;
  }
  .markdown-rendered :global(ul),
  .markdown-rendered :global(ol) {
    padding-left: 18px;
  }
  .markdown-rendered :global(li) {
    margin-bottom: 3px;
  }

  /* Smooth Site Cards styling */
  .sites-card {
    background: linear-gradient(
      135deg,
      rgba(30, 30, 30, 0.6) 0%,
      rgba(20, 20, 20, 0.8) 100%
    );
    border: 1px solid rgba(255, 255, 255, 0.05);
    border-radius: 12px;
    padding: 16px;
    cursor: pointer;
    transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1);
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  }
  .sites-card:hover {
    border-color: rgba(129, 140, 248, 0.4);
    transform: translateY(-1px);
    box-shadow: 0 6px 20px rgba(129, 140, 248, 0.08);
    background: linear-gradient(
      135deg,
      rgba(35, 35, 35, 0.7) 0%,
      rgba(25, 20, 35, 0.9) 100%
    );
  }
  .sites-card-inner {
    display: flex;
    align-items: center;
    gap: 16px;
    width: 100%;
  }
  .sites-card-icon-container {
    width: 44px;
    height: 44px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(129, 140, 248, 0.08);
    border: 1px solid rgba(129, 140, 248, 0.15);
    border-radius: 10px;
    transition: all 0.2s ease;
    flex-shrink: 0;
  }
  .sites-card:hover .sites-card-icon-container {
    background: rgba(129, 140, 248, 0.14);
    border-color: rgba(129, 140, 248, 0.3);
  }
  .sites-emoji {
    font-size: 22px;
  }
  .sites-card-details {
    display: flex;
    flex-direction: column;
    flex: 1;
    min-width: 0;
  }
  .sites-card-name {
    font-weight: 600;
    color: #ffffff;
    font-size: 14px;
    letter-spacing: -0.2px;
    margin-bottom: 2px;
  }
  .sites-card-desc {
    color: #94a3b8;
    font-size: 12px;
    line-height: 1.4;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    transition: color 0.2s ease;
  }
  .sites-card:hover .sites-card-desc {
    color: #cbd5e1;
  }
  .sites-card-status {
    display: flex;
    align-items: center;
    justify-content: center;
    padding-left: 8px;
    flex-shrink: 0;
  }
  .status-indicator-dot {
    width: 10px;
    height: 10px;
    border-radius: 50%;
    background: #ef4444;
    transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
    box-shadow: 0 0 8px rgba(239, 68, 68, 0.2);
  }
  .status-indicator-dot.live {
    background: #10b981;
    box-shadow: 0 0 12px rgba(16, 185, 129, 0.6);
    animation: pulse-active 2s infinite;
  }

  @keyframes pulse-active {
    0% {
      box-shadow: 0 0 8px rgba(16, 185, 129, 0.4);
    }
    50% {
      box-shadow: 0 0 16px rgba(16, 185, 129, 0.7);
    }
    100% {
      box-shadow: 0 0 8px rgba(16, 185, 129, 0.4);
    }
  }

  /* Modals Layout styling */
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
  .modal-container {
    background: #1a1a1a;
    border: 1px solid #2e2e2e;
    border-radius: 12px;
    width: 420px;
    max-width: 90%;
    max-height: 400px;
    display: flex;
    flex-direction: column;
    box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.5);
    overflow: hidden;
  }
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
  }
  .close-btn:hover {
    color: #f87171;
    background: rgba(239, 68, 68, 0.1);
  }
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
  .clear-btn {
    background: transparent;
    border: none;
    color: #64748b;
    font-size: 14px;
    cursor: pointer;
    padding: 2px 6px;
  }
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

  /* Premium minimalist Buttons styling */
  .btn {
    border-radius: 6px;
    font-size: 12px;
    font-weight: 600;
    padding: 8px 16px;
    cursor: pointer;
    border: none;
    transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
  }
  .btn.primary {
    background: #818cf8;
    color: #ffffff;
    border: 1px solid rgba(129, 140, 248, 0.4);
    box-shadow: 0 4px 12px rgba(129, 140, 248, 0.2);
  }
  .btn.primary:hover {
    background: #6366f1;
    border-color: rgba(99, 102, 241, 0.6);
    box-shadow: 0 6px 16px rgba(99, 102, 241, 0.35);
    transform: translateY(-0.5px);
  }
  .btn.secondary {
    background: rgba(255, 255, 255, 0.04);
    border: 1px solid rgba(255, 255, 255, 0.08);
    color: #cbd5e1;
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
  }
  .btn.secondary:hover {
    background: rgba(255, 255, 255, 0.08);
    border-color: rgba(255, 255, 255, 0.16);
    color: #ffffff;
    transform: translateY(-0.5px);
  }
  .img-action-btn {
    background: rgba(255, 255, 255, 0.04);
    border: 1px solid rgba(255, 255, 255, 0.08);
    color: #94a3b8;
    font-size: 11px;
    padding: 4px 10px;
    border-radius: 6px;
    cursor: pointer;
    transition: all 0.2s;
  }
  .img-action-btn:hover {
    background: rgba(255, 255, 255, 0.08);
    border-color: rgba(255, 255, 255, 0.16);
    color: #ffffff;
  }
  .img-action-btn.danger:hover {
    background: rgba(239, 68, 68, 0.08);
    border-color: rgba(239, 68, 68, 0.2);
    color: #f87171;
  }

  .link-action-btn {
    background: rgba(255, 255, 255, 0.04);
    border: 1px solid rgba(255, 255, 255, 0.08);
    color: #cbd5e1;
    font-size: 11px;
    padding: 4px 10px;
    border-radius: 6px;
    cursor: pointer;
    transition: all 0.2s;
  }
  .link-action-btn:hover {
    background: rgba(255, 255, 255, 0.08);
    border-color: rgba(255, 255, 255, 0.16);
    color: #ffffff;
  }

  .sites-card-preview:hover {
    background: rgba(129, 140, 248, 0.05);
    border-color: #818cf8;
  }
  .emoji-tab-btn {
    background: transparent;
    border: none;
    color: #64748b;
    font-size: 12px;
    font-weight: 600;
    padding: 6px 12px;
    border-radius: 4px;
    cursor: pointer;
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
