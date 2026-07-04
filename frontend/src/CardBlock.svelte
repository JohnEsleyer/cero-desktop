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
    // Dispatch event to open the modal globally at the root level (App.svelte)
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
        /("(.*?)"|'([^']*)')/g,
        '<span style="color: #34d399;">$1</span>',
      )
      .replace(
        /(\/\/[^\n]*)/g,
        '<span style="color: #94a3b8; font-style: italic;">$1</span>',
      );
  }

  $: linkedPage = (() => {
    if (card.type !== "subpage_link" || !card.content) return null;
    return allPages.find((p) => p.id === card.content) || null;
  })();

  $: renderedMarkdown = marked.parse(card.content || "");
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

<style>
  .card-block {
    position: relative;
    background: #18181b;
    border: 1px solid rgba(255, 255, 255, 0.04);
    border-radius: 8px;
    padding: 14px;
    cursor: pointer;
    transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
    text-align: left; /* Explicitly align texts inside all block cards from the left */
  }
  .card-block:hover {
    border-color: rgba(255, 255, 255, 0.08);
    background: #1c1c20;
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
    text-align: left; /* Left-align the main card body previews */
  }

  .card-preview:hover {
    background: rgba(255, 255, 255, 0.01);
    border-radius: 4px;
  }
  .empty-hint {
    color: #52525b;
    font-style: italic;
    font-size: 12px;
  }

  .image-card-content {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .image-wrapper {
    display: flex;
    justify-content: center;
    border-radius: 6px;
    overflow: hidden;
    background: rgba(0, 0, 0, 0.1);
  }
  .card-image {
    max-width: 100%;
    max-height: 320px;
    object-fit: contain;
  }
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
  }
  .image-placeholder:hover {
    border-color: rgba(129, 140, 248, 0.3);
    background: rgba(129, 140, 248, 0.01);
  }
  .placeholder-icon {
    font-size: 20px;
    margin-bottom: 6px;
    opacity: 0.8;
  }
  .placeholder-text {
    color: #71717a;
    font-size: 11px;
    font-weight: 500;
  }
  .image-actions {
    display: flex;
    gap: 6px;
    justify-content: flex-end;
  }
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
  .img-action-btn:hover {
    background: rgba(255, 255, 255, 0.06);
    color: #f4f4f5;
  }
  .img-action-btn.danger:hover {
    background: rgba(239, 68, 68, 0.08);
    border-color: rgba(239, 68, 68, 0.2);
    color: #f87171;
  }

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
  .subpage-link-card:hover {
    background: rgba(255, 255, 255, 0.03);
    border-color: rgba(255, 255, 255, 0.06);
  }
  .link-emoji {
    font-size: 18px;
  }
  .link-info {
    flex: 1;
    display: flex;
    flex-direction: column;
    min-width: 0;
  }
  .link-title {
    color: #e4e4e7;
    font-weight: 600;
    font-size: 12px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .link-hint {
    color: #52525b;
    font-size: 10px;
    margin-top: 1px;
  }
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
  .link-action-btn:hover {
    background: rgba(255, 255, 255, 0.06);
    color: #f4f4f5;
  }

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
  .subpage-empty:hover {
    border-color: rgba(129, 140, 248, 0.3);
    color: #818cf8;
  }

  .file-card {
    color: #a1a1aa;
    font-size: 12px;
  }

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
  .action-btn:hover {
    background: rgba(239, 68, 68, 0.08);
    border-color: rgba(239, 68, 68, 0.2);
    color: #f87171;
  }

  .markdown-rendered :global(h1) {
    font-size: 16px;
    margin: 12px 0 6px 0;
    color: #ffffff;
    font-weight: 700;
  }
  .markdown-rendered :global(h2) {
    font-size: 14px;
    margin: 10px 0 4px 0;
    color: #ffffff;
    font-weight: 700;
  }
  .markdown-rendered :global(h3) {
    font-size: 12px;
    margin: 8px 0 2px 0;
    color: #ffffff;
    font-weight: 700;
  }
  .markdown-rendered :global(code) {
    background: rgba(255, 255, 255, 0.03);
    padding: 2px 4px;
    border-radius: 4px;
    font-size: 11px;
    font-family: monospace;
    color: #f472b6;
  }
  .markdown-rendered :global(pre) {
    background: rgba(0, 0, 0, 0.2);
    padding: 10px;
    border-radius: 6px;
    overflow-x: auto;
    border: 1px solid rgba(255, 255, 255, 0.03);
  }
  .markdown-rendered :global(ul),
  .markdown-rendered :global(ol) {
    padding-left: 16px;
    margin: 6px 0;
  }
  .markdown-rendered :global(li) {
    margin-bottom: 2px;
  }

  /* Smooth Site Cards styling */
  .sites-card {
    background: rgba(255, 255, 255, 0.005);
    border: 1px solid rgba(255, 255, 255, 0.04);
    border-radius: 8px;
    padding: 12px;
    cursor: pointer;
    transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
  }
  .sites-card:hover {
    border-color: rgba(129, 140, 248, 0.3);
    background: rgba(129, 140, 248, 0.01);
  }
  .sites-card-inner {
    display: flex;
    align-items: center;
    gap: 12px;
    width: 100%;
  }
  .sites-card-icon-container {
    width: 32px;
    height: 32px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(255, 255, 255, 0.02);
    border: 1px solid rgba(255, 255, 255, 0.05);
    border-radius: 6px;
    transition: all 0.2s ease;
    flex-shrink: 0;
  }
  .sites-card:hover .sites-card-icon-container {
    background: rgba(129, 140, 248, 0.08);
    border-color: rgba(129, 140, 248, 0.2);
  }
  .sites-emoji {
    font-size: 16px;
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
    font-size: 12px;
    margin-bottom: 1px;
  }
  .sites-card-desc {
    color: #71717a;
    font-size: 11px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .sites-card:hover .sites-card-desc {
    color: #a1a1aa;
  }
  .sites-card-status {
    display: flex;
    align-items: center;
    justify-content: center;
    padding-left: 6px;
    flex-shrink: 0;
  }
  .status-indicator-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: #52525b;
    transition: all 0.3s ease;
  }
  .status-indicator-dot.live {
    background: #10b981;
    box-shadow: 0 0 8px rgba(16, 185, 129, 0.5);
    animation: pulse-active-dot 2s infinite;
  }

  @keyframes pulse-active-dot {
    0% {
      box-shadow: 0 0 4px rgba(16, 185, 129, 0.3);
    }
    50% {
      box-shadow: 0 0 8px rgba(16, 185, 129, 0.6);
    }
    100% {
      box-shadow: 0 0 4px rgba(16, 185, 129, 0.3);
    }
  }

  .emoji-tab-btn {
    background: transparent;
    border: none;
    color: #71717a;
    font-size: 11px;
    font-weight: 600;
    padding: 6px 12px;
    border-radius: 4px;
    cursor: pointer;
    transition: all 0.12s;
  }
  .emoji-tab-btn:hover {
    color: #a1a1aa;
    background: rgba(255, 255, 255, 0.02);
  }
  .emoji-tab-btn.active {
    color: #818cf8;
    background: rgba(129, 140, 248, 0.08);
  }
</style>
