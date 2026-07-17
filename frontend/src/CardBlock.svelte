<script>
  import { getContext } from "svelte";
  import {
    UpdateCard,
    DeleteCard,
    SaveImage,
  } from "../wailsjs/go/main/App.js";
  import PageIcon from "./PageIcon.svelte";
  import SvelteMarkdown from "@humanspeak/svelte-markdown";
  import { markedKatex, KatexRenderer } from "@humanspeak/svelte-markdown/extensions";

  let {
    card,
    isSelected = false,
    index = 0,
    allPages = [],
    onSelect,
    onDeleted,
    onNavigate,
    activeLiveCardId = null,
    activeSitesLocalUrl = "",
    oneditMarkdown,
    oneditCode,
    oneditSites,
    onpreviewSites,
    onopenLinkModal,
    onopenCommentsModal,
    onopenMoveBlockModal,
    onmoveUp,
    onmoveDown,
  } = $props();

  const { showAlert, showConfirm } = getContext("dialogs");

  let codeLang = $state("javascript");
  let codeContent = $state("");

  let sitesName = $state("My HTML Site");
  let sitesDesc = $state("Renders customized HTML template preview");
  let sitesHtml = $state("<h1>Sample Site</h1>\n<p>Edit HTML and watch it render live.</p>");

  let isSitesLive = $derived(activeLiveCardId === card.id);
  let sitesLocalUrl = $derived(activeLiveCardId === card.id ? activeSitesLocalUrl : "");

  const colorPresets = {
    default: { bg: "#09090b", text: "#e4e4e7", border: "rgba(255, 255, 255, 0.04)", textMuted: "#71717a", borderLeft: "#71717a" },
    red: { bg: "#fee2e2", text: "#991b1b", border: "#fca5a5", textMuted: "#b91c1c", borderLeft: "#ef4444" },
    orange: { bg: "#ffedd5", text: "#9a3412", border: "#fdba74", textMuted: "#c2410c", borderLeft: "#f97316" },
    yellow: { bg: "#fef9c3", text: "#854d0e", border: "#fde047", textMuted: "#a16207", borderLeft: "#eab308" },
    green: { bg: "#d1fae5", text: "#065f46", border: "#6ee7b7", textMuted: "#047857", borderLeft: "#10b981" },
    blue: { bg: "#dbeafe", text: "#1e40af", border: "#93c5fd", textMuted: "#1d4ed8", borderLeft: "#3b82f6" },
    purple: { bg: "#f3e8ff", text: "#6b21a8", border: "#c084fc", textMuted: "#7e22ce", borderLeft: "#a855f7" }
  };

  function parseMetadata(commentField) {
    if (!commentField) {
      return { color: "default", comments: [] };
    }
    try {
      const data = JSON.parse(commentField);
      return {
        color: data.color || "default",
        comments: Array.isArray(data.comments) ? data.comments : []
      };
    } catch (_) {
      if (colorPresets[commentField]) {
        return { color: commentField, comments: [] };
      }
      return { color: "default", comments: [commentField] };
    }
  }

  function serializeMetadata(color, comments) {
    return JSON.stringify({ color, comments });
  }

  let meta = $derived(parseMetadata(card.comment));
  let activeColor = $derived(colorPresets[meta.color] || colorPresets.default);

  $effect(() => {
    if (card.type === "code") {
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
  });

  $effect(() => {
    if (card.type === "sites") {
      try {
        const parsed = JSON.parse(card.content || "{}");
        sitesName = parsed.name || "My HTML Site";
        sitesDesc = parsed.description || "Renders customized HTML template preview";
        sitesHtml = parsed.html || "<h1>Sample Site</h1>";
      } catch (_) {
        sitesName = "My HTML Site";
        sitesDesc = "Renders customized HTML template preview";
        sitesHtml = card.content || "";
      }
    }
  });

  function handleClick() {
    if (onSelect) onSelect(card);
  }

  function startEdit() {
    if (card.type === "markdown") {
      oneditMarkdown && oneditMarkdown(card.content || "");
    } else if (card.type === "section") {
      const newTitle = prompt("Edit Section Header:", card.content || "");
      if (newTitle !== null) {
        UpdateCard(card.id, card.page_id, newTitle.trim(), card.comment || "")
          .then(() => { card.content = newTitle.trim(); })
          .catch((err) => console.error(err));
      }
    }
  }

  async function deleteCard(e) {
    e.stopPropagation();
    const confirmed = await showConfirm("Delete Card", "Are you sure you want to delete this card?");
    if (!confirmed) return;

    try {
      await DeleteCard(card.id, card.page_id || "");
      if (onDeleted) onDeleted(card.id);
    } catch (err) {
      console.error("DeleteCard backend error:", err);
      showAlert("Delete Failed", err?.toString() || "Could not delete card from server.");
    }
  }

  function setImage() {
    const choice = prompt("Image source:\n1: Enter URL\n2: Upload file");
    if (choice === "1") {
      const url = prompt("Enter image URL:", card.content || "");
      if (url !== null) {
        UpdateCard(card.id, card.page_id, url, card.comment || "")
          .then(() => { card.content = url; })
          .catch((err) => console.error(err));
      }
    } else if (choice === "2") {
      imageInputEl?.click();
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
            .then(() => { card.content = filename; })
            .catch((err) => console.error(err));
        })
        .catch((err) => showAlert("Upload Failed", err.toString()));
    };
    reader.readAsDataURL(file);
    e.target.value = "";
  }

  async function unlinkImage(e) {
    e.stopPropagation();
    const confirmed = await showConfirm("Remove Image", "Are you sure you want to remove this image?");
    if (!confirmed) return;
    UpdateCard(card.id, card.page_id, "", card.comment || "")
      .then(() => { card.content = ""; })
      .catch((err) => console.error(err));
  }

  function selectSubpage(e) {
    e.stopPropagation();
    onopenLinkModal && onopenLinkModal(card);
  }

  function navigateToSubpage(e) {
    e.stopPropagation();
    if (!card.content || !onNavigate) return;
    const target = allPages.find((p) => p.id === card.content);
    if (target) onNavigate(target);
  }

  function openSitesModal() {
    oneditSites && oneditSites(sitesName, sitesDesc, sitesHtml);
  }

  function openLivePreviewModal() {
    onpreviewSites && onpreviewSites(sitesName, sitesHtml);
  }

  function handleColorChange(e) {
    const newColor = e.target.value;
    const payload = serializeMetadata(newColor, meta.comments);
    UpdateCard(card.id, card.page_id, card.content || "", payload)
      .then(() => {
        card.comment = payload;
      })
      .catch((err) => console.error(err));
  }

  function highlightCode(code, lang) {
    if (!code) return '<span class="tok-comment">// Start writing your code here...</span>';

    let html = code
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;");

    const keywords = [
      "function", "return", "if", "else", "for", "while", "const", "let", "var",
      "import", "export", "class", "void", "final", "def", "package", "func", "interface",
      "fn", "mut", "match", "impl", "struct", "enum", "pub", "use", "mod", "as", "type",
      "select", "from", "where", "insert", "into", "update", "delete", "create", "table", "alter"
    ];
    const builtins = [
      "console", "window", "document", "process", "Object", "Array", "String", "Number",
      "Boolean", "Math", "JSON", "Option", "Result", "Some", "None", "Ok", "Err", "Box",
      "Vec", "Self", "self", "map", "filter", "reduce", "print", "len", "range"
    ];

    const tokenRegex = new RegExp(
      [
        "(?<multilinecomment>\\/\\*[\\s\\S]*?\\*\\/)",
        "(?<singlecomment>\\/\\/[^\\n]*|#[^\\n]*)",
        "(?<string>\"(?:\\\\.|[^\"\\\\])*\"|'(?:\\\\.|[^'\\\\])*'|`(?:\\\\.|[^`\\\\])*`)",
        "\\b(?<number>\\d+(?:\\.\\d+)?)\\b",
        "\\b(?<function>\\w+)(?=\\s*\\()",
        "\\b(?<identifier>\\w+)\\b",
        "(?<operator>[{}()\\[\\].,:;+\\-*/%=&|^<>!~?])"
      ].join("|"),
      "g"
    );

    return html.replace(tokenRegex, (...args) => {
      const groups = args[args.length - 1];
      if (typeof groups === 'object' && groups !== null) {
        if (groups.multilinecomment || groups.singlecomment) {
          return `<span class="tok-comment">${groups.multilinecomment || groups.singlecomment}</span>`;
        }
        if (groups.string) {
          return `<span class="tok-string">${groups.string}</span>`;
        }
        if (groups.number) {
          return `<span class="tok-number">${groups.number}</span>`;
        }
        if (groups.function) {
          return `<span class="tok-function">${groups.function}</span>`;
        }
        if (groups.identifier) {
          const word = groups.identifier;
          if (keywords.includes(word)) {
            return `<span class="tok-keyword">${word}</span>`;
          }
          if (builtins.includes(word)) {
            return `<span class="tok-builtin">${word}</span>`;
          }
          return word;
        }
        if (groups.operator) {
          return `<span class="tok-operator">${groups.operator}</span>`;
        }
      }
      return args[0];
    });
  }

  let linkedPage = $derived((() => {
    if (card.type !== "subpage_link" || !card.content) return null;
    return allPages.find((p) => p.id === card.content) || null;
  })());

  const katexExtensions = [markedKatex({ singleDollarInline: true })];
  const katexRenderers = {
    inlineKatex: KatexRenderer,
    blockKatex: KatexRenderer
  };

  let imageInputEl = $state(null);
</script>

<div
  class="card-block {card.type}"
  style="background: {activeColor.bg}; color: {activeColor.text}; border-color: {activeColor.border}; border-left: 4px solid {activeColor.borderLeft};"
  class:selected={isSelected}
  onclick={handleClick}
  role="button"
  tabindex="0"
>
  <div class="card-horizontal-header">
    <div class="header-left">
      {#if index > 0}
        <span class="card-order-badge-new">#{index}</span>
      {/if}
      <span class="card-type-label" style="color: {activeColor.textMuted};">
        {card.type}
      </span>
    </div>

    <div class="card-inline-toolbar">
      {#if card.type === "markdown"}
        <button
          class="inline-comment-toggle-btn"
          title="View block comments"
          onclick={(e) => { e.stopPropagation(); onopenCommentsModal && onopenCommentsModal(card); }}
        >
          💬
          {#if meta.comments.length > 0}
            <span class="comment-count-badge">{meta.comments.length}</span>
          {/if}
        </button>
      {/if}

      <select
        value={meta.color}
        onchange={handleColorChange}
        onclick={(e) => e.stopPropagation()}
        class="color-select"
        style="color: {activeColor.textMuted};"
      >
        <option value="default">⚫ Default</option>
        <option value="red">🔴 Red</option>
        <option value="orange">🟠 Orange</option>
        <option value="yellow">🟡 Yellow</option>
        <option value="green">🟢 Green</option>
        <option value="blue">🔵 Blue</option>
        <option value="purple">🟣 Purple</option>
      </select>

      {#if card.type === "section"}
        <button class="inline-tool-btn" style="color: {activeColor.textMuted}" onclick={(e) => { e.stopPropagation(); onmoveUp && onmoveUp() }}>↑</button>
        <button class="inline-tool-btn" style="color: {activeColor.textMuted}" onclick={(e) => { e.stopPropagation(); onmoveDown && onmoveDown() }}>↓</button>
      {/if}
      <button
        class="inline-tool-btn"
        title="Move Block to another page"
        style="color: {activeColor.textMuted}; font-size: 11px; font-weight: bold; margin-left: 8px;"
        onclick={(e) => { e.stopPropagation(); onopenMoveBlockModal && onopenMoveBlockModal(card) }}
      >
        📦 Move
      </button>
      {#if isSelected}
        <button class="inline-delete-btn" onclick={(e) => { e.stopPropagation(); deleteCard(e); }}>&times;</button>
      {/if}
    </div>
  </div>

  <div class="card-horizontal-content-body">
    {#if card.type === "section"}
      <div class="section-card-content" ondblclick={startEdit}>
        <h2 class="section-title-text" style="color: {activeColor.text};">{card.content || "Untitled Section"}</h2>
      </div>

    {:else if card.type === "markdown"}
      <div class="card-preview markdown-rendered" style="color: {activeColor.text};" ondblclick={startEdit}>
        {#if card.content}
          <SvelteMarkdown source={card.content} extensions={katexExtensions} renderers={katexRenderers}>
            {#snippet code({ lang, text })}
              <pre class="markdown-code-block"><div class="code-lang-badge">{(lang || '').toUpperCase() || 'CODE'}</div><code>{@html highlightCode(text, lang || '')}</code></pre>
            {/snippet}
          </SvelteMarkdown>
        {:else}
          <span class="empty-hint">Double-click to write content...</span>
        {/if}
      </div>

    {:else if card.type === "image"}
      <div class="image-card-content">
        <input type="file" bind:this={imageInputEl} accept="image/*" style="display:none" onchange={handleFileUpload} />
        {#if card.content}
          <div class="image-wrapper">
            <img src={card.content} alt="Card visual" class="card-image" onerror={(e) => (e.target.style.display = "none")} />
          </div>
          <div class="image-actions">
            <button class="img-action-btn" onclick={(e) => { e.stopPropagation(); setImage(e); }}>Change</button>
            <button class="img-action-btn danger" onclick={(e) => { e.stopPropagation(); unlinkImage(e); }}>Remove</button>
          </div>
        {:else}
          <div class="image-placeholder" onclick={(e) => { e.stopPropagation(); setImage(e); }}>
            <span class="placeholder-text">Click to add image file or web URL</span>
          </div>
        {/if}
      </div>

    {:else if card.type === "subpage_link"}
      {#if linkedPage}
        <div 
          class="subpage-link-card-custom" 
          style="
            background: {linkedPage.cover ? `url(${linkedPage.cover})` : 'linear-gradient(135deg, #1e1e24, #2a2a35)'}; 
            background-size: cover; 
            background-position: center; 
            border: 1px solid rgba(255, 255, 255, 0.08); 
            padding: 0 12px; 
            height: 48px; 
            display: flex; 
            align-items: center; 
            gap: 8px; 
            border-radius: 6px; 
            width: 100%; 
            box-sizing: border-box; 
            position: relative; 
            overflow: hidden;
          "
          onclick={(e) => { e.stopPropagation(); navigateToSubpage(e); }}
        >
          <div style="position: absolute; inset: 0; background: rgba(0, 0, 0, {linkedPage.cover ? '0.45' : '0.15'}); z-index: 1;"></div>
          
          <span class="link-emoji" style="display: flex; align-items: center; z-index: 2;"><PageIcon emoji={linkedPage.emoji} size={14} /></span>
          <span class="link-title" style="color: #ffffff; font-size: 11.5px; font-weight: 700; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; flex: 1; z-index: 2; text-shadow: 0 1px 2px rgba(0,0,0,0.8);">
            {linkedPage.title || "Untitled"}
          </span>
          <div style="display: flex; align-items: center; gap: 8px; margin-left: auto; flex-shrink: 0; z-index: 2;">
            <span class="link-hint" style="color: rgba(255,255,255,0.7); font-size: 9px; font-weight: 700; text-shadow: 0 1px 2px rgba(0,0,0,0.8);">Open &rarr;</span>
            <button 
              class="link-action-btn-mini" 
              style="background: transparent; border: none; padding: 0; color: rgba(255,255,255,0.7); font-size: 10px; cursor: pointer; display: flex; align-items: center;" 
              onclick={(e) => { e.stopPropagation(); selectSubpage(e); }}
              title="Change Target Link"
            >
              ⚙️
            </button>
          </div>
        </div>
      {:else}
        <div 
          class="subpage-empty" 
          style="background: rgba(255, 255, 255, 0.02); border: 1px dashed {activeColor.textMuted}44; padding: 4px 10px; height: 32px; display: flex; align-items: center; justify-content: center; border-radius: 6px; font-size: 11px; color: {activeColor.textMuted};"
          onclick={(e) => { e.stopPropagation(); selectSubpage(e); }}
        >
          <span>Link subpage...</span>
        </div>
      {/if}

    {:else if card.type === "code"}
      <div class="code-card-content">
        <div
          class="code-preview-container"
          ondblclick={(e) => { e.stopPropagation(); oneditCode && oneditCode(card.content || "") }}
        >
          <div class="code-lang-tag">{codeLang}</div>
          <pre style="margin:0; font-size:12px; line-height:1.5; overflow-x:auto;">{@html highlightCode(codeContent, codeLang)}</pre>
        </div>
      </div>

    {:else if card.type === "sites"}
      <div class="sites-card">
        <div class="sites-card-inner">
          <div class="sites-card-details" onclick={(e) => { e.stopPropagation(); openLivePreviewModal(); }}>
            <span class="sites-card-name" style="color: {activeColor.text};">{sitesName}</span>
            <span class="sites-card-desc">{sitesDesc}</span>
          </div>
          <div class="sites-card-actions">
            <button class="img-action-btn" style="color: #10b981;" onclick={(e) => { e.stopPropagation(); openLivePreviewModal(); }}>▶ Preview</button>
            <button class="img-action-btn" style="color: #818cf8;" onclick={(e) => { e.stopPropagation(); openSitesModal(); }}>⚙ Config</button>
          </div>
        </div>
      </div>
    {/if}
  </div>
</div>

<style>
  .card-block {
    position: relative;
    border-radius: 8px;
    border-width: 1px;
    border-style: solid;
    padding: 10px 14px;
    cursor: pointer;
    display: flex;
    flex-direction: column;
    gap: 8px;
    transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
    text-align: left;
  }

  .card-block:hover {
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  }

  .card-horizontal-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    width: 100%;
    gap: 12px;
  }

  .header-left {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .card-order-badge-new {
    background: rgba(129, 140, 248, 0.1);
    color: #818cf8;
    font-size: 9px;
    font-weight: 700;
    padding: 1px 5px;
    border-radius: 4px;
  }

  .card-type-label {
    font-size: 8px;
    font-weight: 800;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .card-inline-toolbar {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .inline-comment-toggle-btn {
    position: relative;
    background: transparent;
    border: none;
    font-size: 13px;
    cursor: pointer;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    padding: 4px;
  }

  .comment-count-badge {
    position: absolute;
    top: -4px;
    right: -4px;
    background: #818cf8;
    color: white;
    font-size: 8px;
    font-weight: 700;
    border-radius: 50%;
    min-width: 12px;
    height: 12px;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 1px;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.2);
  }

  .color-select {
    background: transparent;
    border: none;
    font-size: 9px;
    font-weight: 700;
    cursor: pointer;
    outline: none;
  }

  .color-select option {
    background: #18181a;
    color: #cbd5e1;
  }

  .inline-tool-btn {
    background: transparent;
    border: none;
    cursor: pointer;
    font-size: 14px;
    font-weight: bold;
    padding: 2px 6px;
    border-radius: 4px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
  }

  .inline-delete-btn {
    background: transparent;
    border: none;
    cursor: pointer;
    font-size: 14px;
    font-weight: bold;
    padding: 2px 6px;
    border-radius: 4px;
    margin-left: 16px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
  }

  .inline-tool-btn:hover {
    background: rgba(255, 255, 255, 0.1);
  }

  .inline-delete-btn:hover {
    background: rgba(239, 68, 68, 0.15);
    color: #ef4444;
  }

  .card-horizontal-content-body {
    width: 100%;
  }

  .section-card-content {
    width: 100%;
  }

  .section-title-text {
    font-size: 13px;
    font-weight: 800;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    margin: 0;
  }

  .empty-hint {
    font-style: italic;
    font-size: 11px;
    opacity: 0.5;
  }

  .image-card-content {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .image-wrapper {
    display: flex;
    border-radius: 6px;
    overflow: hidden;
  }

  .card-image {
    max-width: 100%;
    max-height: 240px;
    object-fit: contain;
  }

  .image-placeholder {
    padding: 16px;
    border: 1px dashed rgba(255, 255, 255, 0.1);
    border-radius: 6px;
    text-align: center;
    color: #71717a;
  }

  .placeholder-text {
    font-size: 11px;
  }

  .image-actions {
    display: flex;
    gap: 6px;
    justify-content: flex-end;
  }

  .img-action-btn {
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid rgba(255, 255, 255, 0.1);
    color: inherit;
    font-size: 9px;
    font-weight: 600;
    padding: 2px 6px;
    border-radius: 4px;
    cursor: pointer;
  }



  .code-preview-container {
    background: rgba(0, 0, 0, 0.25);
    border-radius: 6px;
    padding: 10px;
    position: relative;
  }

  .code-lang-tag {
    position: absolute;
    top: 4px;
    right: 8px;
    font-size: 8px;
    font-weight: bold;
    opacity: 0.5;
  }

  .sites-card-inner {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
  }

  .sites-card-details {
    display: flex;
    flex-direction: column;
    flex: 1;
    min-width: 0;
  }

  .sites-card-name {
    font-weight: 700;
    font-size: 12px;
  }

  .sites-card-desc {
    font-size: 10px;
    opacity: 0.6;
  }

  .sites-card-actions {
    display: flex;
    gap: 6px;
  }



  .new-comment-input-row input {
    flex: 1;
    background: #18181a;
    border: 1px solid rgba(255, 255, 255, 0.05);
    border-radius: 4px;
    color: #e4e4e7;
    font-size: 11px;
    padding: 4px 8px;
    outline: none;
  }

  .new-comment-input-row button {
    background: #818cf8;
    border: none;
    border-radius: 4px;
    color: white;
    font-size: 10px;
    font-weight: 600;
    padding: 4px 10px;
    cursor: pointer;
  }

  .markdown-rendered {
    text-align: left !important;
  }

  .markdown-rendered :global(h1),
  .markdown-rendered :global(h2),
  .markdown-rendered :global(h3),
  .markdown-rendered :global(h4),
  .markdown-rendered :global(h5),
  .markdown-rendered :global(h6),
  .markdown-rendered :global(p),
  .markdown-rendered :global(li),
  .markdown-rendered :global(span),
  .markdown-rendered :global(strong),
  .markdown-rendered :global(em),
  .markdown-rendered :global(blockquote) {
    color: inherit !important;
    text-align: left !important;
  }

  .markdown-rendered :global(h1) { font-size: 1.4em; margin-top: 8px; margin-bottom: 4px; }
  .markdown-rendered :global(h2) { font-size: 1.25em; margin-top: 8px; margin-bottom: 4px; }
  .markdown-rendered :global(h3) { font-size: 1.1em; margin-top: 6px; margin-bottom: 3px; }
  .markdown-rendered :global(h4),
  .markdown-rendered :global(h5),
  .markdown-rendered :global(h6) { font-size: 1em; margin-top: 6px; margin-bottom: 3px; }
</style>
