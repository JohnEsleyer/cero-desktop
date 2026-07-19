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

  const PREVIEW_WORD_LIMIT = 100;

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
    onReadFullscreen,
    onmoveUp,
    onmoveDown,
  } = $props();

  const { showAlert, showConfirm } = getContext("dialogs");

  function truncateText(text) {
    if (!text) return "";
    const words = text.split(/(\s+)/);
    let wordCount = 0;
    let result = [];
    for (const part of words) {
      if (part.trim().length > 0) {
        wordCount++;
      }
      if (wordCount > PREVIEW_WORD_LIMIT) {
        result.push("...");
        break;
      }
      result.push(part);
    }
    return result.join("");
  }

  let isTruncated = $derived(
    card.content ? card.content.split(/\s+/).filter(w => w.trim().length > 0).length > PREVIEW_WORD_LIMIT : false
  );

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
    if (card.type === "markdown") {
      if (onReadFullscreen) onReadFullscreen(card);
    } else if (card.type === "code") {
      if (onReadFullscreen) onReadFullscreen(card);
    } else if (card.type === "sites") {
      openLivePreviewModal();
    }
  }

  function startEdit() {
    if (card.type === "section") {
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
          class="inline-tool-btn"
          style="color: {activeColor.textMuted}; font-size: 11px; font-weight: bold;"
          onclick={(e) => { e.stopPropagation(); oneditMarkdown && oneditMarkdown(card.content || ""); }}
        >
          ✏️ Edit
        </button>
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

      {#if card.type === "code"}
        <button
          class="inline-tool-btn"
          style="color: {activeColor.textMuted}; font-size: 11px; font-weight: bold;"
          onclick={(e) => { e.stopPropagation(); oneditCode && oneditCode(card.content || ""); }}
        >
          ✏️ Edit
        </button>
      {/if}

      {#if card.type === "sites"}
        <button
          class="inline-tool-btn"
          style="color: {activeColor.textMuted}; font-size: 11px; font-weight: bold;"
          onclick={(e) => { e.stopPropagation(); openSitesModal(); }}
        >
          ✏️ Edit
        </button>
      {/if}

      {#if card.type === "section"}
        <button
          class="inline-tool-btn"
          style="color: {activeColor.textMuted}; font-size: 11px; font-weight: bold;"
          onclick={(e) => { e.stopPropagation(); startEdit(); }}
        >
          ✏️ Edit
        </button>
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
      <div class="section-card-content">
        <h2 class="section-title-text" style="color: {activeColor.text};">{card.content || "Untitled Section"}</h2>
      </div>

    {:else if card.type === "markdown"}
      <div class="card-preview markdown-rendered" style="color: {activeColor.text};">
        {#if card.content}
          {#snippet codeSnippet({ lang, text })}
            <pre class="markdown-code-block"><div class="code-lang-badge">{(lang || '').toUpperCase() || 'CODE'}</div><code>{@html highlightCode(text, lang || '')}</code></pre>
          {/snippet}
          <SvelteMarkdown source={truncateText(card.content)} extensions={katexExtensions} renderers={katexRenderers} code={codeSnippet} />
          {#if isTruncated}
            <button class="read-fullscreen-btn" onclick={(e) => { e.stopPropagation(); onReadFullscreen && onReadFullscreen(card); }}>Read Fullscreen</button>
          {/if}
        {:else}
          <span class="empty-hint">Click Edit button to write content...</span>
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
        <div class="code-preview-container">
          <div class="code-lang-tag">{codeLang}</div>
          <pre style="margin:0; font-size:12px; line-height:1.5; overflow-x:auto;">{@html highlightCode(codeContent, codeLang)}</pre>
        </div>
      </div>

    {:else if card.type === "sites"}
      <div class="sites-card">
        <div class="sites-card-inner">
          <div class="sites-card-details">
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

  /* Redesigned Markdown Card rendering system matching the full screen mode */
  .card-preview.markdown-rendered {
    color: inherit;
    font-size: 13px;
    line-height: 1.6;
    letter-spacing: -0.05px;
    text-align: left !important;
  }
  .card-preview.markdown-rendered :global(p) {
    margin: 0 0 10px 0;
    text-align: left !important;
  }
  .card-preview.markdown-rendered :global(p:last-child) {
    margin-bottom: 0;
  }
  .card-preview.markdown-rendered :global(h1) {
    font-size: 1.25rem;
    margin: 16px 0 8px 0;
    color: inherit;
    font-weight: 800;
    border-bottom: 1px solid rgba(255, 255, 255, 0.06);
    padding-bottom: 4px;
    text-align: left !important;
  }
  .card-preview.markdown-rendered :global(h2) {
    font-size: 1.1rem;
    margin: 14px 0 6px 0;
    color: inherit;
    font-weight: 700;
    letter-spacing: -0.2px;
    text-align: left !important;
  }
  .card-preview.markdown-rendered :global(h3) {
    font-size: 0.95rem;
    margin: 12px 0 4px 0;
    color: #818cf8;
    font-weight: 600;
    letter-spacing: -0.1px;
    text-align: left !important;
  }
  .card-preview.markdown-rendered :global(h4), 
  .card-preview.markdown-rendered :global(h5), 
  .card-preview.markdown-rendered :global(h6) {
    font-size: 0.85rem;
    margin: 10px 0 4px 0;
    color: inherit;
    font-weight: 600;
    opacity: 0.9;
    text-align: left !important;
  }
  .card-preview.markdown-rendered :global(code:not(pre code)) {
    background: rgba(129, 140, 248, 0.08);
    border: 1px solid rgba(129, 140, 248, 0.15);
    padding: 2px 4px;
    border-radius: 4px;
    font-size: 11px;
    font-family: "Fira Code", Consolas, Monaco, monospace;
    color: #f472b6;
  }
  .card-preview.markdown-rendered :global(.markdown-code-block) {
    position: relative;
    background: #09090b;
    border: 1px solid rgba(255, 255, 255, 0.06);
    border-radius: 8px;
    margin: 10px 0;
    padding: 10px 12px;
    overflow-x: auto;
  }
  .card-preview.markdown-rendered :global(.markdown-code-block pre) {
    margin: 0;
    background: transparent;
    border: none;
    padding: 0;
  }
  .card-preview.markdown-rendered :global(.markdown-code-block code) {
    background: transparent !important;
    border: none !important;
    padding: 0 !important;
    font-size: 11.5px;
    line-height: 1.45;
    font-family: "Fira Code", "Cascadia Code", Consolas, monospace;
    color: #e4e4e7;
    display: block;
  }
  .card-preview.markdown-rendered :global(.code-lang-badge) {
    position: absolute;
    top: 6px;
    right: 8px;
    font-size: 8px;
    font-weight: 800;
    color: #818cf8;
    background: rgba(129, 140, 248, 0.08);
    border: 1px solid rgba(129, 140, 248, 0.15);
    padding: 1px 5px;
    border-radius: 4px;
    user-select: none;
    pointer-events: none;
  }
  .card-preview.markdown-rendered :global(ul), 
  .card-preview.markdown-rendered :global(ol) {
    padding-left: 18px;
    margin: 0 0 10px 0;
    text-align: left !important;
  }
  .card-preview.markdown-rendered :global(li) {
    margin-bottom: 3px;
    text-align: left !important;
  }
  .card-preview.markdown-rendered :global(li::marker) {
    color: #818cf8;
  }
  .card-preview.markdown-rendered :global(.task-list-item) {
    list-style-type: none;
    margin-left: -18px;
    margin-bottom: 3px;
  }
  .card-preview.markdown-rendered :global(.task-list-label) {
    display: flex;
    align-items: flex-start;
    gap: 6px;
    cursor: default;
    user-select: none;
  }
  .card-preview.markdown-rendered :global(.task-list-checkbox) {
    appearance: none;
    -webkit-appearance: none;
    width: 12px;
    height: 12px;
    border: 1.5px solid #52525b;
    border-radius: 3px;
    outline: none;
    background-color: transparent;
    cursor: default;
    margin-top: 3px;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }
  .card-preview.markdown-rendered :global(.task-list-checkbox:checked) {
    background-color: #818cf8;
    border-color: #818cf8;
  }
  .card-preview.markdown-rendered :global(.task-list-checkbox:checked::before) {
    content: "✓";
    color: white;
    font-size: 8px;
    font-weight: bold;
  }
  .card-preview.markdown-rendered :global(.task-list-checkbox:checked + .task-list-text) {
    color: #71717a;
    text-decoration: line-through;
  }
  .card-preview.markdown-rendered :global(.markdown-blockquote) {
    border-left: 3px solid #818cf8;
    background: rgba(129, 140, 248, 0.03);
    padding: 6px 12px;
    margin: 10px 0;
    border-radius: 0 4px 4px 0;
    color: #a1a1aa;
    font-style: italic;
    text-align: left !important;
  }
  .card-preview.markdown-rendered :global(.markdown-blockquote p) {
    margin: 0;
  }
  .card-preview.markdown-rendered :global(.markdown-table-wrapper) {
    overflow-x: auto;
    margin: 10px 0;
    border: 1px solid rgba(255, 255, 255, 0.06);
    border-radius: 6px;
    background: #09090b;
  }
  .card-preview.markdown-rendered :global(.markdown-table) {
    width: 100%;
    border-collapse: collapse;
    font-size: 11px;
    text-align: left;
  }
  .card-preview.markdown-rendered :global(.markdown-table th) {
    background: rgba(255, 255, 255, 0.02);
    font-weight: 700;
    color: #ffffff;
    padding: 8px 12px;
    border-bottom: 1px solid rgba(255, 255, 255, 0.06);
  }
  .card-preview.markdown-rendered :global(.markdown-table td) {
    padding: 8px 12px;
    border-bottom: 1px solid rgba(255, 255, 255, 0.04);
    color: inherit;
  }
  .card-preview.markdown-rendered :global(.markdown-table tbody tr:last-child td) {
    border-bottom: none;
  }
  .card-preview.markdown-rendered :global(.markdown-table tbody tr:nth-child(even)) {
    background: rgba(255, 255, 255, 0.005);
  }
  .card-preview.markdown-rendered :global(hr) {
    border: none;
    border-top: 1px solid rgba(255, 255, 255, 0.06);
    margin: 14px 0;
  }
  .card-preview.markdown-rendered :global(a) {
    color: #818cf8;
    text-decoration: none;
    border-bottom: 1px dotted transparent;
    transition: all 0.15s ease;
  }
  .card-preview.markdown-rendered :global(a:hover) {
    color: #a5b4fc;
    border-bottom-color: #a5b4fc;
  }
  .card-preview.markdown-rendered :global(img) {
    max-width: 100%;
    border-radius: 6px;
    margin: 8px 0;
    border: 1px solid rgba(255, 255, 255, 0.05);
  }
  .card-preview.markdown-rendered :global(kbd) {
    background: rgba(255, 255, 255, 0.04);
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: 4px;
    padding: 2px 4px;
    font-size: 9px;
    font-family: inherit;
    color: inherit;
    box-shadow: 0 1px 0 rgba(0,0,0,0.2);
  }

  .read-fullscreen-btn {
    display: block;
    margin-top: 12px;
    background: rgba(129, 140, 248, 0.12);
    color: #818cf8;
    border: 1px solid rgba(129, 140, 248, 0.25);
    padding: 6px 16px;
    border-radius: 6px;
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.2s ease;
  }
  .read-fullscreen-btn:hover {
    background: rgba(129, 140, 248, 0.2);
  }

  .reading-modal-overlay {
    position: fixed;
    top: 0;
    left: 0;
    width: 100vw;
    height: 100vh;
    background: rgba(0, 0, 0, 0.7);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 9999;
  }

  .reading-modal {
    background: #1a1a22;
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: 12px;
    width: 90vw;
    max-width: 800px;
    max-height: 85vh;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .reading-modal-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 16px 20px;
    border-bottom: 1px solid rgba(255, 255, 255, 0.06);
  }

  .reading-modal-title {
    font-size: 14px;
    font-weight: 700;
    color: #e2e2e2;
  }

  .reading-modal-close {
    background: none;
    border: none;
    color: #888;
    font-size: 22px;
    cursor: pointer;
    padding: 0 4px;
    line-height: 1;
  }
  .reading-modal-close:hover {
    color: #fff;
  }

  .reading-modal-content {
    padding: 24px 28px;
    overflow-y: auto;
    font-size: 16px;
    line-height: 1.7;
    color: #d4d4d4;
    font-family: "Georgia", "Times New Roman", serif;
  }
  .reading-modal-content :global(h1),
  .reading-modal-content :global(h2),
  .reading-modal-content :global(h3),
  .reading-modal-content :global(h4),
  .reading-modal-content :global(h5),
  .reading-modal-content :global(h6),
  .reading-modal-content :global(p),
  .reading-modal-content :global(li),
  .reading-modal-content :global(blockquote) {
    color: #e8e8e8 !important;
  }
  .reading-modal-content :global(p) {
    margin: 0 0 12px 0;
  }
  .reading-modal-content :global(p:last-child) {
    margin-bottom: 0;
  }
  .reading-modal-content :global(h1) {
    font-size: 1.5rem;
    margin: 20px 0 10px 0;
    font-weight: 800;
    border-bottom: 1px solid rgba(255, 255, 255, 0.06);
    padding-bottom: 6px;
  }
  .reading-modal-content :global(h2) {
    font-size: 1.3rem;
    margin: 18px 0 8px 0;
    font-weight: 700;
    letter-spacing: -0.2px;
  }
  .reading-modal-content :global(h3) {
    font-size: 1.15rem;
    margin: 14px 0 6px 0;
    font-weight: 600;
  }
  .reading-modal-content :global(h4),
  .reading-modal-content :global(h5),
  .reading-modal-content :global(h6) {
    font-size: 1rem;
    margin: 12px 0 6px 0;
    font-weight: 600;
    opacity: 0.9;
  }
  .reading-modal-content :global(code:not(pre code)) {
    background: rgba(129, 140, 248, 0.08);
    border: 1px solid rgba(129, 140, 248, 0.15);
    padding: 2px 5px;
    border-radius: 4px;
    font-size: 13px;
    font-family: "Fira Code", Consolas, Monaco, monospace;
    color: #f472b6;
  }
  .reading-modal-content :global(.markdown-code-block) {
    position: relative;
    background: #09090b;
    border: 1px solid rgba(255, 255, 255, 0.06);
    border-radius: 8px;
    margin: 14px 0;
    padding: 14px 16px;
    overflow-x: auto;
  }
  .reading-modal-content :global(.markdown-code-block pre) {
    margin: 0;
    background: transparent;
    border: none;
    padding: 0;
  }
  .reading-modal-content :global(.markdown-code-block code) {
    background: transparent !important;
    border: none !important;
    padding: 0 !important;
    font-size: 13px;
    line-height: 1.5;
    font-family: "Fira Code", "Cascadia Code", Consolas, monospace;
    color: #e4e4e7;
    display: block;
  }
  .reading-modal-content :global(.code-lang-badge) {
    position: absolute;
    top: 6px;
    right: 8px;
    font-size: 9px;
    font-weight: 800;
    color: #818cf8;
    background: rgba(129, 140, 248, 0.08);
    border: 1px solid rgba(129, 140, 248, 0.15);
    padding: 1px 6px;
    border-radius: 4px;
    user-select: none;
    pointer-events: none;
  }
  .reading-modal-content :global(ul),
  .reading-modal-content :global(ol) {
    padding-left: 22px;
    margin: 0 0 12px 0;
  }
  .reading-modal-content :global(li) {
    margin-bottom: 4px;
  }
  .reading-modal-content :global(li::marker) {
    color: #818cf8;
  }
  .reading-modal-content :global(.task-list-item) {
    list-style-type: none;
    margin-left: -22px;
    margin-bottom: 4px;
  }
  .reading-modal-content :global(.task-list-label) {
    display: flex;
    align-items: flex-start;
    gap: 8px;
    cursor: default;
    user-select: none;
  }
  .reading-modal-content :global(.task-list-checkbox) {
    appearance: none;
    -webkit-appearance: none;
    width: 14px;
    height: 14px;
    border: 1.5px solid #52525b;
    border-radius: 3px;
    outline: none;
    background-color: transparent;
    cursor: default;
    margin-top: 4px;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }
  .reading-modal-content :global(.task-list-checkbox:checked) {
    background-color: #818cf8;
    border-color: #818cf8;
  }
  .reading-modal-content :global(.task-list-checkbox:checked::before) {
    content: "✓";
    color: white;
    font-size: 10px;
    font-weight: bold;
  }
  .reading-modal-content :global(.task-list-checkbox:checked + .task-list-text) {
    color: #71717a;
    text-decoration: line-through;
  }
  .reading-modal-content :global(.markdown-blockquote) {
    border-left: 3px solid #818cf8;
    background: rgba(129, 140, 248, 0.03);
    padding: 8px 16px;
    margin: 14px 0;
    border-radius: 0 4px 4px 0;
    color: #a1a1aa;
    font-style: italic;
  }
  .reading-modal-content :global(.markdown-blockquote p) {
    margin: 0;
  }
  .reading-modal-content :global(.markdown-table-wrapper) {
    overflow-x: auto;
    margin: 14px 0;
    border: 1px solid rgba(255, 255, 255, 0.06);
    border-radius: 6px;
    background: #09090b;
  }
  .reading-modal-content :global(.markdown-table) {
    width: 100%;
    border-collapse: collapse;
    font-size: 13px;
    text-align: left;
  }
  .reading-modal-content :global(.markdown-table th) {
    background: rgba(255, 255, 255, 0.02);
    font-weight: 700;
    color: #ffffff;
    padding: 10px 14px;
    border-bottom: 1px solid rgba(255, 255, 255, 0.06);
  }
  .reading-modal-content :global(.markdown-table td) {
    padding: 10px 14px;
    border-bottom: 1px solid rgba(255, 255, 255, 0.04);
    color: inherit;
  }
  .reading-modal-content :global(.markdown-table tbody tr:last-child td) {
    border-bottom: none;
  }
  .reading-modal-content :global(.markdown-table tbody tr:nth-child(even)) {
    background: rgba(255, 255, 255, 0.005);
  }
  .reading-modal-content :global(hr) {
    border: none;
    border-top: 1px solid rgba(255, 255, 255, 0.06);
    margin: 18px 0;
  }
  .reading-modal-content :global(a) {
    color: #818cf8;
    text-decoration: none;
    border-bottom: 1px dotted transparent;
    transition: all 0.15s ease;
  }
  .reading-modal-content :global(a:hover) {
    color: #a5b4fc;
    border-bottom-color: #a5b4fc;
  }
  .reading-modal-content :global(img) {
    max-width: 100%;
    border-radius: 6px;
    margin: 10px 0;
    border: 1px solid rgba(255, 255, 255, 0.05);
  }
  .reading-modal-content :global(kbd) {
    background: rgba(255, 255, 255, 0.04);
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: 4px;
    padding: 2px 5px;
    font-size: 11px;
    font-family: inherit;
    color: inherit;
    box-shadow: 0 1px 0 rgba(0,0,0,0.2);
  }
</style>
