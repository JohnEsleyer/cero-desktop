<script context="module">
  import { default as CardBlock } from "./CardBlock.svelte";
  import { default as RightSidebar } from "./RightSidebar.svelte";
</script>

<script>
  import { onMount, tick, setContext } from "svelte";
  import { EventsOn } from "../wailsjs/runtime/runtime.js";
  import PageIcon from "./PageIcon.svelte";
  import SvelteMarkdown from "@humanspeak/svelte-markdown";
  import { markedKatex, KatexRenderer } from "@humanspeak/svelte-markdown/extensions";
  import {
    GetConnectionStatus,
    GetDbPages,
    GetDiscoveredDevices,
    ConnectToDevice,
    Disconnect,
    AddPage as _AddPage,
    UpdatePage as _UpdatePage,
    DeletePage as _DeletePage,
    RestorePage as _RestorePage,
    HardDeletePage as _HardDeletePage,
    MovePage as _MovePage,
    GetCards as _GetCards,
    FetchCards as _FetchCards,
    AddCard as _AddCard,
    UpdateCard as _UpdateCard,
    DeleteCard as _DeleteCard,
    ReorderCards as _ReorderCards,
    StartHTMLServer as _StartHTMLServer,
    StopHTMLServer as _StopHTMLServer,
    SaveImage as _SaveImage,
  } from "../wailsjs/go/main/App.js";

  let connectionStatus = $state("disconnected");
  let discoveredDevices = $state([]);
  let connectedDevice = $state(null);
  let manualIp = $state("");
  let manualPort = $state(9090);
  let manualPin = $state("");
  let connectionError = $state("");

  let activeWorkspace = $state("");

  let allPages = $state([]);
  let selectedPage = $state(null);
  let navigationHistory = $state([]);
  let recentlyOpenedRootPageIds = $state([]);

  let pageCards = $state([]);
  let selectedCardId = $state(null);

  let editorTitle = $state("");
  let editorEmoji = $state("");
  let saveTimeout = $state(null);

  // Global Sync Request Tracker
  let activeRequestsCount = $state(0);
  let isSyncing = $derived(activeRequestsCount > 0);

  setContext("syncState", {
    increment: () => { activeRequestsCount++; },
    decrement: () => { activeRequestsCount = Math.max(0, activeRequestsCount - 1); }
  });

  function wrapSync(fn) {
    return (...args) => {
      activeRequestsCount++;
      return fn(...args).finally(() => {
        activeRequestsCount = Math.max(0, activeRequestsCount - 1);
      });
    };
  }

  const AddPage = wrapSync(_AddPage);
  const UpdatePage = wrapSync(_UpdatePage);
  const DeletePage = wrapSync(_DeletePage);
  const RestorePage = wrapSync(_RestorePage);
  const HardDeletePage = wrapSync(_HardDeletePage);
  const MovePage = wrapSync(_MovePage);
  const GetCards = wrapSync(_GetCards);
  const FetchCards = wrapSync(_FetchCards);
  const AddCard = wrapSync(_AddCard);
  const UpdateCard = wrapSync(_UpdateCard);
  const DeleteCard = wrapSync(_DeleteCard);
  const ReorderCards = wrapSync(_ReorderCards);
  const StartHTMLServer = wrapSync(_StartHTMLServer);
  const StopHTMLServer = wrapSync(_StopHTMLServer);
  const SaveImage = wrapSync(_SaveImage);

  // Root Page search query state
  let sidebarSearchQuery = $state("");

  // Reactive filtration to only display filtered active root pages
  let filteredSidebarRootPages = $derived(rootPages.filter((p) => {
    if (!sidebarSearchQuery) return true;
    return (p.title || "").toLowerCase().includes(sidebarSearchQuery.toLowerCase());
  }));

  // View state option for block-by-block pagination
  let isPaginatedView = $state(false);
  let currentBlockIndex = $state(0);
  let blockNumberInput = $state("1");
  let pendingBlockIndex = $state(null);

  $effect(() => {
    if (pageCards && pageCards.length > 0) {
      if (currentBlockIndex >= pageCards.length) {
        currentBlockIndex = pageCards.length - 1;
      }
      blockNumberInput = (currentBlockIndex + 1).toString();
    } else {
      currentBlockIndex = 0;
      blockNumberInput = "1";
    }
  });

  $effect(() => {
    if (pageCards && pendingBlockIndex !== null) {
      if (pendingBlockIndex >= 0 && pendingBlockIndex < pageCards.length) {
        currentBlockIndex = pendingBlockIndex;
        blockNumberInput = (currentBlockIndex + 1).toString();
        pendingBlockIndex = null;
      }
    }
  });

  // Reusable Sleek Modal Dialog Configuration
  let modalConfig = $state({
    show: false,
    title: "",
    message: "",
    type: "alert",
    onConfirm: null,
    onCancel: null,
    confirmText: "OK",
    cancelText: "Cancel"
  });

  function showAlert(title, message) {
    return new Promise((resolve) => {
      modalConfig = {
        show: true,
        title: title || "Notification",
        message: message,
        type: "alert",
        confirmText: "OK",
        cancelText: "",
        onConfirm: () => {
          modalConfig.show = false;
          resolve(true);
        },
        onCancel: () => {
          modalConfig.show = false;
          resolve(false);
        }
      };
    });
  }

  function showConfirm(title, message, confirmText = "Confirm", cancelText = "Cancel") {
    return new Promise((resolve) => {
      modalConfig = {
        show: true,
        title: title || "Confirm Action",
        message: message,
        type: "confirm",
        confirmText: confirmText,
        cancelText: cancelText,
        onConfirm: () => {
          modalConfig.show = false;
          resolve(true);
        },
        onCancel: () => {
          modalConfig.show = false;
          resolve(false);
        }
      };
    });
  }

  setContext("dialogs", { showAlert, showConfirm });

  // Custom Modal States
  let showBlockSelectorModal = $state(false);
  let blockInsertIndex = $state(null);
  let cardColumnContainer = $state(null);

  let showEmojiPickerModal = $state(false);

  let showScrollToBottom = $state(false);

  let iconImageInput = $state(null);
  let customIconUrl = $state("");

  function handleIconUpload(e) {
    const file = e.target.files[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = () => {
      const base64 = reader.result;
      if (typeof SaveImage === "function") {
        SaveImage(base64, file.name)
          .then((filename) => {
            editorEmoji = filename;
            showEmojiPickerModal = false;
            savePageImmediate();
          })
          .catch((err) => showAlert("Upload Failed", "Failed to save icon image: " + err));
      }
    };
    reader.readAsDataURL(file);
    e.target.value = "";
  }

  function applyCustomIconUrl() {
    if (customIconUrl.trim()) {
      editorEmoji = customIconUrl.trim();
      showEmojiPickerModal = false;
      savePageImmediate();
    }
  }

  function handleCardColumnScroll() {
    if (!cardColumnContainer) return;
    const { scrollTop, scrollHeight, clientHeight } = cardColumnContainer;
    const maxScroll = scrollHeight - clientHeight;
    const threshold = 200;
    const shouldShow = maxScroll > 400 && (maxScroll - scrollTop) > threshold;
    showScrollToBottom = shouldShow;
  }

  function scrollToBottom() {
    if (cardColumnContainer) {
      cardColumnContainer.scrollTo({
        top: cardColumnContainer.scrollHeight,
        behavior: "smooth",
      });
    }
  }

  // PIN Entry Modal States
  let showPinModal = $state(false);
  let pinInput = $state("");
  let pendingDeviceToConnect = $state(null);
  let pinModalError = $state("");
  let isConnecting = $state(false);

  // Link a Subpage Modal States (Moved to root level)
  let showLinkPageModal = $state(false);
  let linkingCard = $state(null);
  let pageSearchQuery = $state("");

  // Immersive Fullscreen States at App Level
  let editingCard = $state(null);
  let showMarkdownFullscreenModal = $state(false);
  let showCodeFullscreenModal = $state(false);
  let showSitesModal = $state(false);

  let showSitesPreviewModal = $state(false);
  let previewSitesName = $state("");
  let previewSitesHtml = $state("");

  // Move Page Modal state
  let showMovePageModal = $state(false);
  let movingPage = $state(null);
  let moveBrowsingId = $state(null);

  // Move Block Modal states
  let showMoveBlockModal = $state(false);
  let movingCard = $state(null);

  function openMoveBlockModal(card) {
    movingCard = card;
    showMoveBlockModal = true;
  }

  function handleMoveBlockToPage(page) {
    if (!movingCard || !page) return;
    UpdateCard(movingCard.id, page.id, movingCard.content || "", movingCard.comment || "")
      .then(() => {
        const id = movingCard.id;
        pageCards = pageCards.filter((c) => c.id !== id);
        if (currentBlockIndex >= pageCards.length) {
          currentBlockIndex = pageCards.length > 0 ? pageCards.length - 1 : 0;
          blockNumberInput = (currentBlockIndex + 1).toString();
        }
        showMoveBlockModal = false;
        movingCard = null;
        showAlert("Success", `Moved block to "${page.title || 'Untitled'}"`);
      })
      .catch((err) => showAlert("Move Failed", err.toString()));
  }

  let moveBlockSubpages = $derived(selectedPage ? allPages.filter(p => (p.parent_id || p.parentId) === selectedPage.id && p.relation_type !== "sidepage" && p.id !== selectedPage.id) : []);
  let moveBlockNeighbors = $derived(selectedPage ? allPages.filter(p => (p.parent_id || p.parentId) === (selectedPage.parent_id || selectedPage.parentId) && p.id !== selectedPage.id && p.relation_type !== "sidepage") : []);
  let moveBlockOthers = $derived((() => {
    if (!selectedPage) return [];
    const excluded = new Set([selectedPage.id, ...moveBlockSubpages.map(p => p.id), ...moveBlockNeighbors.map(p => p.id)]);
    return allPages.filter(p => !excluded.has(p.id) && p.relation_type !== "sidepage");
  })());

  // Comments Modal state
  let showCommentsModal = $state(false);
  let commentCard = $state(null);
  let commentInput = $state("");

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
    if (!commentField) return { color: "default", comments: [] };
    try {
      const data = JSON.parse(commentField);
      return {
        color: data.color || "default",
        comments: Array.isArray(data.comments) ? data.comments : []
      };
    } catch (_) {
      if (colorPresets[commentField]) return { color: commentField, comments: [] };
      return { color: "default", comments: [commentField] };
    }
  }

  function serializeMetadata(color, comments) {
    return JSON.stringify({ color, comments });
  }

  function openCommentsModal(card) {
    commentCard = card;
    commentInput = "";
    showCommentsModal = true;
  }

  function handleAddComment() {
    if (!commentCard || !commentInput.trim()) return;
    const meta = parseMetadata(commentCard.comment);
    meta.comments.push(commentInput.trim());
    const payload = serializeMetadata(meta.color, meta.comments);
    UpdateCard(commentCard.id, commentCard.page_id || commentCard.pageId, commentCard.content || "", payload)
      .then(() => {
        const idx = pageCards.findIndex(c => c.id === commentCard.id);
        if (idx !== -1) pageCards[idx].comment = payload;
        commentCard.comment = payload;
        commentInput = "";
      })
      .catch(err => console.error(err));
  }

  function handleDeleteComment(index) {
    if (!commentCard) return;
    const meta = parseMetadata(commentCard.comment);
    meta.comments.splice(index, 1);
    const payload = serializeMetadata(meta.color, meta.comments);
    UpdateCard(commentCard.id, commentCard.page_id || commentCard.pageId, commentCard.content || "", payload)
      .then(() => {
        const idx = pageCards.findIndex(c => c.id === commentCard.id);
        if (idx !== -1) {
          pageCards[idx].comment = payload;
          pageCards = [...pageCards];
        }
        commentCard.comment = payload;
      })
      .catch(err => console.error(err));
  }

  function openSitesPreview(name, html) {
    previewSitesName = name;
    previewSitesHtml = html;
    showSitesPreviewModal = true;
  }

  // Real-time server live tracking
  let activeLiveCardId = $state(null);
  let activeSitesLocalUrl = $state("");
  let isSitesLive = $state(false);

  let autoOpenEditCardAfterAdd = $state(false);

  // Fullscreen reading state
  let readingCard = $state(null);
  let showReadingFullscreenModal = $state(false);
  let readingViewTab = $state("read"); // "read" | "scratchpad" | "tallies"

  let readingCodeLang = $derived((() => {
    if (!readingCard || readingCard.type !== "code") return "javascript";
    const raw = readingCard.content || "";
    if (raw.includes("\n")) {
      return raw.substring(0, raw.indexOf("\n")).trim().toLowerCase();
    }
    return "javascript";
  })());

  let readingCodeContent = $derived((() => {
    if (!readingCard || readingCard.type !== "code") return "";
    const raw = readingCard.content || "";
    if (raw.includes("\n")) {
      return raw.substring(raw.indexOf("\n") + 1);
    }
    return raw;
  })());

  // Fullscreen edit copy
  let fullscreenEditContent = $state("");
  let fullscreenCodeLang = $state("javascript");
  let fullscreenCodeContent = $state("");

  // Sites Card variables inside sandbox modal
  let sitesTab = $state("edit");
  let sitesName = $state("");
  let sitesDesc = $state("");
  let sitesHtml = $state("");
  let sitesLocalUrl = $state("");

  // Immersive Markdown Helpers
  let markdownTextarea = $state(null);
  let markdownCharCount = $derived(fullscreenEditContent
    ? fullscreenEditContent.length
    : 0);
  let markdownWordCount = $derived(fullscreenEditContent
    ? fullscreenEditContent.trim().split(/\s+/).filter(Boolean).length
    : 0);

  let parentPage = $derived(selectedPage && (selectedPage.parent_id || selectedPage.parentId) ? allPages.find(p => p.id === (selectedPage.parent_id || selectedPage.parentId)) : null);

  function insertMarkdownSymbol(prefix, suffix = "") {
    if (!markdownTextarea) return;
    const start = markdownTextarea.selectionStart;
    const end = markdownTextarea.selectionEnd;
    const text = fullscreenEditContent;
    const selected = text.substring(start, end);

    fullscreenEditContent =
      text.substring(0, start) +
      prefix +
      selected +
      suffix +
      text.substring(end);

    setTimeout(() => {
      markdownTextarea.focus();
      const newPos = start + prefix.length + selected.length;
      markdownTextarea.setSelectionRange(newPos, newPos);
    }, 0);
  }

  // Immersive Code Editor Real-time Sync & Highlighting
  let lineNumbersElement = $state(null);
  let preElement = $state(null);

  function handleEditorScroll(e) {
    const { scrollTop, scrollLeft } = e.target;
    if (preElement) {
      preElement.scrollTop = scrollTop;
      preElement.scrollLeft = scrollLeft;
    }
    if (lineNumbersElement) {
      lineNumbersElement.scrollTop = scrollTop;
    }
  }

  // Custom robust Named Capture Groups syntax highlighter
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

  let lineNumbers = $derived((fullscreenCodeContent.match(/\n/g) || []).length + 1);
  let highlightedCodeHtml = $derived(highlightCode(
    fullscreenCodeContent,
    fullscreenCodeLang,
  ));

  const emojiCategories = [
    {
      id: "custom_image",
      label: "🖼️ Custom",
    },
    {
      id: "general",
      label: "✨ Base",
      emojis: [
        "file-text",
        "book-open",
        "folder",
        "link",
        "settings",
        "archive"
      ],
    },
    {
      id: "dev",
      label: "💻 Tech",
      emojis: [
        "code",
        "terminal",
        "globe",
        "activity",
        "hash",
        "heading"
      ],
    },
  ];

  // Sidebar widths (pixels)
  let leftSidebarWidth = $state(260);
  let rightSidebarWidth = $state(260);
  let showRightSidebar = $state(false);
  let dragging = $state(null); // 'left' | 'right' | null
  let dragStartX = 0;
  let dragStartY = 0;
  let dragStartWidth = 0;
  let dragStartHeight = 0;

  // Persistent Scratchpad State
  let showScratchpad = $state(false);
  let scratchpadContent = $state("");
  let scratchpadTab = $state("write");
  let scratchpadWidth = $state(320);
  let scratchpadHeight = $state(240);
  let scratchpadLayout = $state("bottom"); // "side" | "bottom"

  // Persistent Tally State
  let showTally = $state(false);
  let tallyCards = $state([]);
  let newTallyName = $state("");
  let tallyWidth = $state(320);
  let tallyHeight = $state(240);
  let tallyLayout = $state("bottom"); // "side" | "bottom"

  function toggleScratchpad() {
    showScratchpad = !showScratchpad;
    if (showScratchpad) {
      showTally = false;
    }
  }

  function toggleTally() {
    showTally = !showTally;
    if (showTally) {
      showScratchpad = false;
      if (typeof FetchCards === "function") {
        FetchCards("global-tally");
      }
    }
  }

  // Debounced database sync for local editing
  let scratchpadSaveTimeout;
  function saveScratchpadDebounced() {
    if (scratchpadSaveTimeout) clearTimeout(scratchpadSaveTimeout);
    scratchpadSaveTimeout = setTimeout(() => {
      UpdateCard("global-scratchpad-card", "global-scratchpad", scratchpadContent, "")
        .catch((err) => console.error("Scratchpad sync failed:", err));
    }, 400);
  }

  let rootPages = $derived((() => {
    const rawRoot = allPages.filter(
      (p) => !(p.parent_id || p.parentId) && p.relation_type !== "sidepage" && p.relation_type !== "scratchpad" && p.relation_type !== "tally",
    );
    return rawRoot.sort((a, b) => {
      const idxA = recentlyOpenedRootPageIds.indexOf(a.id);
      const idxB = recentlyOpenedRootPageIds.indexOf(b.id);
      if (idxA === -1 && idxB === -1) return 0;
      if (idxA === -1) return 1;
      if (idxB === -1) return -1;
      return idxA - idxB;
    });
  })());
  let sidePages = $derived(allPages.filter(
    (p) => (p.parent_id || p.parentId) === selectedPage?.id && p.relation_type === "sidepage",
  ));

  // Filter nested subpages candidates globally for the link modal
  let filteredLinkCandidates = $derived((() => {
    if (!linkingCard) return [];
    const currentPageId = linkingCard.page_id || linkingCard.pageId;
    const currentPage = allPages.find(p => p.id === currentPageId);
    if (!currentPage) return [];

    const directChildrenIds = new Set(
      allPages
        .filter((p) => (p.parent_id || p.parentId) === currentPage.id && p.relation_type !== "sidepage")
        .map((p) => p.id),
    );

    const candidates = allPages.filter((p) => {
      if (p.id === currentPageId || p.relation_type === "sidepage") return false;
      if ((p.parent_id || p.parentId) === currentPage.id) return true;
      return directChildrenIds.has(p.parent_id || p.parentId);
    });

    if (!pageSearchQuery) return candidates;
    const query = pageSearchQuery.toLowerCase();
    return candidates.filter(
      (p) =>
        (p.title || "").toLowerCase().includes(query) ||
        (p.emoji || "").includes(query),
    );
  })());

  onMount(async () => {
    try {
      const saved = localStorage.getItem("cero_recently_opened_root_pages");
      if (saved) {
        recentlyOpenedRootPageIds = JSON.parse(saved);
      }
    } catch (e) {
      console.error("Failed to load recently opened pages:", e);
    }

    try {
      connectionStatus = await GetConnectionStatus();
      discoveredDevices = await GetDiscoveredDevices();
      allPages = await GetDbPages();
      
      // Initial load from SQLite
      FetchCards("global-scratchpad");
      FetchCards("global-tally");
    } catch (e) {
      console.error("Init failed:", e);
    }

    EventsOn("connection-status", (status) => {
      connectionStatus = status;
      if (status === "disconnected") {
        connectedDevice = null;
        allPages = [];
        selectedPage = null;
        navigationHistory = [];
      }
      connectionError = "";
    });
    EventsOn("discovered-devices", (d) => {
      discoveredDevices = d;
    });
    EventsOn("db-update", (pages) => {
      allPages = pages;
      if (selectedPage) {
        const updated = pages.find((p) => p.id === selectedPage.id);
        if (!updated) {
          selectedPage = null;
          navigationHistory = [];
        } else if (updated.updated_at !== selectedPage.updated_at) {
          selectedPage = updated;
          const focused = document.activeElement?.id === "editor-title-input";
          if (!focused) editorTitle = updated.title;
          editorEmoji = updated.emoji;
        }
      }
      // Fetch scratchpad and tally when database updates
      FetchCards("global-scratchpad");
      FetchCards("global-tally");
    });
    EventsOn("workspace-status", (data) => {
      activeWorkspace = data.activeWorkspace;
    });
    EventsOn("cards-update", (data) => {
      const pid = data.pageId || data.page_id;

      // Update local state when a remote device modifies global pages
      if (pid === "global-scratchpad") {
        const scratchCard = data.cards?.[0];
        if (scratchCard) {
          const remoteContent = scratchCard.content || "";
          const textareaActive = document.activeElement && document.activeElement.id === "scratchpad-textarea";
          if (scratchpadContent !== remoteContent && !textareaActive) {
            scratchpadContent = remoteContent;
          }
        }
      } else if (pid === "global-tally") {
        tallyCards = data.cards || [];
      } else if (selectedPage && pid === selectedPage.id) {
        const oldCards = pageCards;
        pageCards = data.cards || [];

        if (autoOpenEditCardAfterAdd) {
          autoOpenEditCardAfterAdd = false;
          const newCard = pageCards.find((c) => !oldCards.some((oc) => oc.id === c.id) && c.type === "markdown");
          if (newCard) {
            openMarkdownFullscreen(newCard, newCard.content);
          }
        }

        if (selectedCardId && !pageCards.find((c) => c.id === selectedCardId))
          selectedCardId = null;
      }
    });

    window.addEventListener("mousemove", onDragMove);
    window.addEventListener("mouseup", onDragEnd);
    return () => {
      window.removeEventListener("mousemove", onDragMove);
      window.removeEventListener("mouseup", onDragEnd);
    };
  });

  function onDragStart(panel, e) {
    dragging = panel;
    dragStartX = e.clientX;
    dragStartY = e.clientY;
    if (panel === "left") {
      dragStartWidth = leftSidebarWidth;
    } else if (panel === "right") {
      dragStartWidth = rightSidebarWidth;
    } else {
      dragStartWidth = showTally ? tallyWidth : scratchpadWidth;
      dragStartHeight = showTally ? tallyHeight : scratchpadHeight;
    }
    e.preventDefault();
  }

  function onDragMove(e) {
    if (!dragging) return;
    const delta = e.clientX - dragStartX;
    if (dragging === "left") {
      leftSidebarWidth = Math.max(180, Math.min(450, dragStartWidth + delta));
    } else if (dragging === "right") {
      rightSidebarWidth = Math.max(180, Math.min(450, dragStartWidth - delta));
    } else if (dragging === "scratchpad") {
      if (showTally) {
        tallyWidth = Math.max(200, Math.min(600, dragStartWidth - delta));
      } else {
        scratchpadWidth = Math.max(200, Math.min(600, dragStartWidth - delta));
      }
    } else if (dragging === "scratchpad_height") {
      const deltaY = e.clientY - dragStartY;
      if (showTally) {
        tallyHeight = Math.max(120, Math.min(600, dragStartHeight - deltaY));
      } else {
        scratchpadHeight = Math.max(120, Math.min(600, dragStartHeight - deltaY));
      }
    }
  }

  function onDragEnd() {
    dragging = null;
  }

  function handleConnect(device) {
    connectionError = "";
    pinModalError = "";
    pendingDeviceToConnect = device;
    pinInput = "";
    showPinModal = true;
    isConnecting = false;
  }

  async function submitPin() {
    if (!pinInput.trim()) {
      pinModalError = "PIN is required";
      return;
    }
    isConnecting = true;
    pinModalError = "";
    try {
      connectedDevice = pendingDeviceToConnect;
      await ConnectToDevice(
        pendingDeviceToConnect.ip,
        pendingDeviceToConnect.port,
        pinInput.trim(),
      );
      showPinModal = false;
      pendingDeviceToConnect = null;
    } catch (err) {
      pinModalError = err.toString();
      connectedDevice = null;
    } finally {
      isConnecting = false;
    }
  }

  async function handleConnectManually() {
    if (!manualIp || !manualPin) {
      connectionError = "IP and PIN required";
      return;
    }
    connectionError = "";
    try {
      connectedDevice = {
        ip: manualIp.trim(),
        port: parseInt(manualPort) || 9090,
        deviceName: "Manual",
      };
      await ConnectToDevice(
        manualIp.trim(),
        parseInt(manualPort) || 9090,
        manualPin.trim(),
      );
    } catch (err) {
      connectionError = err.toString();
      connectedDevice = null;
    }
  }

  async function handleDisconnect() {
    try {
      await Disconnect();
      connectedDevice = null;
      allPages = [];
      selectedPage = null;
      navigationHistory = [];
    } catch (err) {}
  }

  async function addTally() {
    if (!newTallyName.trim()) return;
    try {
      const sortOrder = tallyCards.length > 0
        ? Math.max(...tallyCards.map((c) => c.sort_order)) + 1
        : 0;
      const initialJson = JSON.stringify({ name: newTallyName.trim(), count: 0 });
      await AddCard("global-tally", "tally", initialJson, sortOrder);
      newTallyName = "";
      FetchCards("global-tally");
    } catch (e) {
      console.error(e);
    }
  }

  async function updateTallyCount(card, name, newCount) {
    if (newCount < 0) return;
    try {
      const updatedJson = JSON.stringify({ name, count: newCount });
      await UpdateCard(card.id, "global-tally", updatedJson, "");
      FetchCards("global-tally");
    } catch (e) {
      console.error(e);
    }
  }

  async function deleteTally(cardId) {
    const confirmed = await showConfirm("Delete Tally", "Are you sure you want to permanently delete this tally?");
    if (!confirmed) return;
    try {
      await DeleteCard(cardId, "global-tally");
      FetchCards("global-tally");
    } catch (e) {
      console.error(e);
    }
  }

  function selectPage(page, pushToHistory = true) {
    if (saveTimeout) {
      clearTimeout(saveTimeout);
      savePageImmediate();
    }
    if (pushToHistory && selectedPage && selectedPage.id !== page.id)
      navigationHistory = [...navigationHistory, selectedPage.id];
    selectedPage = page;
    editorTitle = page.title;
    editorEmoji = page.emoji;
    selectedCardId = null;
    pageCards = [];
    FetchCards(page.id);
    pendingBlockIndex = null;

    if (!(page.parent_id || page.parentId) && page.relation_type !== "sidepage" && page.relation_type !== "scratchpad" && page.relation_type !== "tally") {
      const updated = [page.id, ...recentlyOpenedRootPageIds.filter((id) => id !== page.id)];
      recentlyOpenedRootPageIds = updated;
      try {
        localStorage.setItem("cero_recently_opened_root_pages", JSON.stringify(updated));
      } catch (e) {
        console.error(e);
      }
    }
  }

  function goBack() {
    if (navigationHistory.length === 0) return;
    const prevId = navigationHistory[navigationHistory.length - 1];
    navigationHistory = navigationHistory.slice(0, -1);
    const prev = allPages.find((p) => p.id === prevId);
    if (prev) selectPage(prev, false);
  }

  function savePageImmediate() {
    if (!selectedPage) return;
    const t = editorTitle.trim() || "Untitled";
    UpdatePage(selectedPage.id, t, editorEmoji).catch(() => {});
  }

  function savePageDebounced() {
    if (saveTimeout) clearTimeout(saveTimeout);
    saveTimeout = setTimeout(savePageImmediate, 500);
  }

  function selectCard(card) {
    selectedCardId = card.id;
  }

  async function createPage(parentId = "", relationType = "subpage") {
    let parentIcon = "";
    if (parentId) {
      const parent = allPages.find((p) => p.id === parentId);
      if (parent) {
        parentIcon = parent.emoji || "";
      }
    }
    try {
      const newPageId = await AddPage(parentId, relationType, "New Page", parentIcon);
      if (parentId) {
        const parentCards = await GetCards(parentId);
        const nextOrder = parentCards.length > 0
          ? Math.max(...parentCards.map(c => c.sort_order)) + 1
          : 0;
        if (selectedPage && selectedPage.id === parentId && isPaginatedView) {
          pendingBlockIndex = pageCards.length;
        }
        await AddCard(parentId, "subpage_link", newPageId, nextOrder);
        if (selectedPage && selectedPage.id === parentId) {
          FetchCards(parentId);
        }
      }
    } catch (err) {
      showAlert("Action Failed", err.toString());
    }
  }

  function createSidePage() {
    if (!selectedPage) return;
    createPage(selectedPage.id, "sidepage");
  }

  async function handleAddCardType(type) {
    showBlockSelectorModal = false;
    if (!selectedPage) return;
    try {
      let sortOrder = 0;
      if (blockInsertIndex !== null) {
        sortOrder = blockInsertIndex;
        if (isPaginatedView) {
          pendingBlockIndex = blockInsertIndex;
        }
      } else {
        sortOrder =
          pageCards.length > 0
            ? Math.max(...pageCards.map((c) => c.sort_order)) + 1
            : 0;
        if (isPaginatedView) {
          pendingBlockIndex = pageCards.length;
        }
      }

      let defaultContent = "";
      if (type === "code") {
        defaultContent =
          'javascript\nconsole.log("Hello from Cero Code Block!");\n';
      } else if (type === "section") {
        defaultContent = "New Section";
      } else if (type === "sites") {
        defaultContent = JSON.stringify({
          name: "Interactive Site Card",
          description: "Sandboxed iframe renderer preview widget",
          html: "<h1>Welcome directly to Sandboxed Environment!</h1>\n<p>Change this code inside the editor tab to render customized HTML components.</p>",
        });
      }

      if (type === "markdown") {
        autoOpenEditCardAfterAdd = true;
      }

      await AddCard(selectedPage.id, type, defaultContent, sortOrder);
      await tick();
      if (cardColumnContainer) {
        cardColumnContainer.scrollTo({
          top: cardColumnContainer.scrollHeight,
          behavior: "smooth",
        });
      }
    } catch (e) {
      showAlert("Action Failed", e.toString());
    }
  }

  function moveCard(fromIndex, direction) {
    const toIndex = fromIndex + direction;
    if (toIndex < 0 || toIndex >= pageCards.length || !selectedPage) return;
    const reordered = [...pageCards];
    const [moved] = reordered.splice(fromIndex, 1);
    reordered.splice(toIndex, 0, moved);
    ReorderCards(
      selectedPage.id,
      reordered.map((c) => c.id),
    )
      .then(() => {
        pageCards = reordered;
      })
      .catch(() => {});
  }

  function handleCardDrop(e, toIndex) {
    const fromIndex = parseInt(e.dataTransfer.getData("text/plain"));
    if (isNaN(fromIndex) || fromIndex === toIndex || !selectedPage) return;
    const reordered = [...pageCards];
    const [moved] = reordered.splice(fromIndex, 1);
    reordered.splice(toIndex > fromIndex ? toIndex - 1 : toIndex, 0, moved);
    ReorderCards(
      selectedPage.id,
      reordered.map((c) => c.id),
    )
      .then(() => {
        pageCards = reordered;
      })
      .catch(() => {});
  }

  async function deletePage(id) {
    const confirmed = await showConfirm("Archive Page", "Are you sure you want to archive this page and all its subpages?");
    if (!confirmed) return;
    DeletePage(id)
      .then(() => {
        if (selectedPage?.id === id) {
          selectedPage = null;
          navigationHistory = [];
        }
      })
      .catch(() => {});
  }

  function movePage(id) {
    const page = allPages.find((p) => p.id === id);
    if (!page) return;
    movingPage = page;
    moveBrowsingId = page.parent_id || page.parentId || null;
    showMovePageModal = true;
  }

  // Immersive Modal Handlers at App Level
  function openMarkdownFullscreen(c, initialContent) {
    editingCard = c;
    fullscreenEditContent = initialContent;
    showMarkdownFullscreenModal = true;
  }

  function saveMarkdownFullscreen() {
    if (!editingCard) return;
    UpdateCard(editingCard.id, editingCard.page_id || editingCard.pageId, fullscreenEditContent, editingCard.comment || "")
      .then(() => {
        const idx = pageCards.findIndex((c) => c.id === editingCard.id);
        if (idx !== -1) {
          pageCards[idx].content = fullscreenEditContent;
          pageCards = [...pageCards];
        }
      })
      .catch((err) => console.error(err));
    showMarkdownFullscreenModal = false;
    editingCard = null;
  }

  function openCodeFullscreen(c, initialContent) {
    editingCard = c;
    const raw = initialContent || "";
    if (raw.includes("\n")) {
      const idx = raw.indexOf("\n");
      fullscreenCodeLang = raw.substring(0, idx).trim().toLowerCase();
      fullscreenCodeContent = raw.substring(idx + 1);
    } else {
      fullscreenCodeLang = "javascript";
      fullscreenCodeContent = raw;
    }
    showCodeFullscreenModal = true;
  }

  function saveCodeFullscreen() {
    if (!editingCard) return;
    const combined = `${fullscreenCodeLang}\n${fullscreenCodeContent}`;
    UpdateCard(editingCard.id, editingCard.page_id || editingCard.pageId, combined, editingCard.comment || "")
      .then(() => {
        const idx = pageCards.findIndex((c) => c.id === editingCard.id);
        if (idx !== -1) {
          pageCards[idx].content = combined;
          pageCards = [...pageCards];
        }
      })
      .catch((err) => console.error(err));
    showCodeFullscreenModal = false;
    editingCard = null;
  }

  async function openSitesFullscreen(c, initialName, initialDesc, initialHtml) {
    editingCard = c;
    sitesName = initialName;
    sitesDesc = initialDesc;
    sitesHtml = initialHtml;
    sitesTab = "edit";
    showSitesModal = true;
    sitesLocalUrl = "";

    if (activeLiveCardId === c.id && isSitesLive) {
      sitesLocalUrl = activeSitesLocalUrl;
    } else {
      isSitesLive = false;
    }
  }

  async function handleSaveSites() {
    if (!editingCard) return;
    const combined = JSON.stringify({
      name: sitesName,
      description: sitesDesc,
      html: sitesHtml,
    });
    UpdateCard(editingCard.id, editingCard.page_id || editingCard.pageId, combined, editingCard.comment || "")
      .then(() => {
        const idx = pageCards.findIndex((c) => c.id === editingCard.id);
        if (idx !== -1) {
          pageCards[idx].content = combined;
          pageCards = [...pageCards];
        }
      })
      .catch((err) => console.error(err));

    if (
      isSitesLive &&
      activeLiveCardId === editingCard.id &&
      typeof StartHTMLServer === "function"
    ) {
      try {
        activeSitesLocalUrl = await StartHTMLServer(sitesHtml);
        sitesLocalUrl = activeSitesLocalUrl;
      } catch (err) {
        console.error(err);
      }
    }
  }

  async function closeSitesModal() {
    await handleSaveSites();
    showSitesModal = false;
    editingCard = null;
  }

  async function toggleSitesLive() {
    if (!editingCard) return;
    if (isSitesLive) {
      try {
        if (typeof StopHTMLServer === "function") {
          await StopHTMLServer();
        }
        isSitesLive = false;
        sitesLocalUrl = "";
        activeLiveCardId = null;
        activeSitesLocalUrl = "";
      } catch (err) {
        console.error(err);
      }
    } else {
      try {
        if (typeof StartHTMLServer === "function") {
          activeSitesLocalUrl = await StartHTMLServer(sitesHtml);
          sitesLocalUrl = activeSitesLocalUrl;
          isSitesLive = true;
          activeLiveCardId = editingCard.id;
        }
      } catch (err) {
        console.error(err);
      }
    }
  }

  // Link Subpage Selection Handlers at Root Level
  function selectPageToLink(target) {
    if (!linkingCard) return;
    UpdateCard(linkingCard.id, linkingCard.page_id || linkingCard.pageId, target.id, linkingCard.comment || "")
      .then(() => {
        const idx = pageCards.findIndex((c) => c.id === linkingCard.id);
        if (idx !== -1) {
          pageCards[idx].content = target.id;
          pageCards = [...pageCards]; // trigger updates cleanly in Svelte 5
        }
        showLinkPageModal = false;
        linkingCard = null;
      })
      .catch((err) => console.error(err));
  }

  async function handleCreateNewPageAndLink() {
    if (!linkingCard) return;
    try {
      const parentPage = allPages.find((p) => p.id === (linkingCard.page_id || linkingCard.pageId));
      const inheritedEmoji = parentPage?.emoji || "";
      const newPageId = await AddPage(
        linkingCard.page_id || linkingCard.pageId,
        "subpage",
        "New Subpage Link",
        inheritedEmoji,
      );

      if (newPageId) {
        await UpdateCard(linkingCard.id, linkingCard.page_id || linkingCard.pageId, newPageId, linkingCard.comment || "");
        const idx = pageCards.findIndex((c) => c.id === linkingCard.id);
        if (idx !== -1) {
          pageCards[idx].content = newPageId;
          pageCards = [...pageCards];
        }
        showLinkPageModal = false;
        linkingCard = null;
      }
    } catch (err) {
      console.error("Failed to create new page & link:", err);
      showAlert("Link Failed", "Could not create and link subpage: " + err);
    }
  }
</script>

<main class="app-layout" class:dragging>
  <!-- LEFT SIDEBAR -->
  <aside
    class="sidebar left-sidebar"
    style="width: {leftSidebarWidth}px; min-width: {leftSidebarWidth}px;"
  >
    <div class="sidebar-header">
      <div class="logo-section">
        <svg class="logo-svg" viewBox="0 0 100 100" fill="none" xmlns="http://www.w3.org/2000/svg">
          <defs>
            <linearGradient id="logo-grad" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop offset="0%" stop-color="#818cf8"/>
              <stop offset="100%" stop-color="#c084fc"/>
            </linearGradient>
          </defs>
          <circle cx="50" cy="50" r="32" stroke="url(#logo-grad)" stroke-width="12" stroke-linecap="round"/>
          <path d="M41 28 L59 72" stroke="url(#logo-grad)" stroke-width="12" stroke-linecap="round"/>
        </svg>
        <span class="logo-text">Cero</span>
      </div>
      <div style="display: flex; align-items: center; gap: 8px;">
        {#if isSyncing}
          <div class="sync-indicator" title="Syncing changes with mobile server..." style="display: flex; align-items: center; gap: 4px;">
            <span class="sync-spinner"></span>
            <span style="font-size: 10px; color: #818cf8; font-weight: 600;">Syncing</span>
          </div>
        {/if}
        <div class="status-badge {connectionStatus}">
          <span class="dot"></span>{connectionStatus}
        </div>
      </div>
    </div>

    <!-- Read-only Active Workspace Details -->
    <div class="sidebar-section active-workspace-card">
      <div class="section-label">Active Workspace</div>
      <div class="active-ws-display">
        <span class="ws-icon" style="display: inline-flex; align-items: center; color: #818cf8; margin-right: 4px;">
          <svg class="lucide-icon" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="20" height="8" x="2" y="3" rx="2" ry="2"/><rect width="20" height="8" x="2" y="13" rx="2" ry="2"/><line x1="6" x2="6.01" y1="7" y2="7"/><line x1="6" x2="6.01" y1="17" y2="17"/></svg>
        </span>
        <span class="ws-name">{activeWorkspace || "Personal"}</span>
      </div>
    </div>

    <!-- PERSISTENT SCRATCHPAD TOGGLE BUTTON -->
    <div class="sidebar-section" style="padding-top: 0; padding-bottom: 6px;">
      <button
        class="mode-toggle-btn"
        class:block-active={showScratchpad}
        style="width: 100%; justify-content: center; gap: 8px;"
        onclick={toggleScratchpad}
        title="Toggle persistent Scratchpad"
      >
        <span class="btn-icon" style="display: inline-flex; align-items: center;">
          <svg class="lucide-icon" xmlns="http://www.w3.org/2000/svg" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 20h9"/><path d="M16.5 3.5a2.12 2.12 0 0 1 3 3L7 19l-4 1 1-4Z"/></svg>
        </span>
        <span class="lbl-text">{showScratchpad ? 'Close Scratchpad' : 'Open Scratchpad'}</span>
      </button>
    </div>

    <!-- PERSISTENT TALLY TOGGLE BUTTON -->
    <div class="sidebar-section" style="padding-top: 0; padding-bottom: 12px;">
      <button
        class="mode-toggle-btn"
        class:block-active={showTally}
        style="width: 100%; justify-content: center; gap: 8px;"
        onclick={toggleTally}
        title="Toggle persistent Tally Page"
      >
        <span class="btn-icon" style="display: inline-flex; align-items: center;">
          <svg class="lucide-icon" xmlns="http://www.w3.org/2000/svg" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="4" x2="20" y1="9" y2="9"/><line x1="4" x2="20" y1="15" y2="15"/><line x1="10" x2="8" y1="3" y2="21"/><line x1="16" x2="14" y1="3" y2="21"/></svg>
        </span>
        <span class="lbl-text">{showTally ? 'Close Tally Page' : 'Open Tally Page'}</span>
      </button>
    </div>

    <div class="sidebar-section">
      <div class="section-label">Link Devices</div>
      <div class="devices-box">
        {#if connectionStatus === "connected"}
          <div class="connected-info">
            <span class="conn-label">Connected to</span>
            <span class="conn-name"
              >{connectedDevice?.deviceName || "Mobile"}</span
            >
          </div>
          <button class="btn-sm full" onclick={handleDisconnect}
            >Disconnect</button
          >
        {:else if discoveredDevices.length === 0}
          <div class="no-dev">Scanning WiFi...</div>
        {:else}
          {#each discoveredDevices as d}
            <div class="dev-row">
              <div>
                <span class="dev-name">{d.deviceName}</span><span class="dev-ip"
                  >{d.ip}:{d.port}</span
                >
              </div>
              <button class="btn-sm primary" onclick={() => handleConnect(d)}
                >Link</button
              >
            </div>
          {/each}
        {/if}
      </div>
    </div>

    <div class="sidebar-section grow">
      <div class="section-label">
        <span>Pages</span>
        {#if connectionStatus === "connected"}
          <button class="icon-btn" onclick={() => createPage("")}>+</button>
        {/if}
      </div>

      <!-- Root Page Search Bar -->
      {#if connectionStatus === "connected"}
        <div class="sidebar-search-container">
          <input
            type="text"
            bind:value={sidebarSearchQuery}
            placeholder="Search root pages..."
            class="sidebar-search-input"
          />
        </div>
      {/if}

      <div class="tree-scroll">
        {#if connectionStatus !== "connected"}
          <div class="tree-empty">Connect to phone to view pages.</div>
        {:else if filteredSidebarRootPages.length === 0}
          <div class="tree-empty">No matching pages found.</div>
        {:else}
          {#each filteredSidebarRootPages as page}
            <div class="tree-node">
              <div class="node-row" class:active={selectedPage && selectedPage.id === page.id}>
                <button class="node-content" onclick={() => selectPage(page)} style="padding-left: 4px;">
                  <span class="emoji"><PageIcon emoji={page.emoji} size={13} /></span>
                  <span class="title" title={page.title}>{page.title || "Untitled"}</span>
                </button>

                <div class="node-actions">
                  <button
                    class="action-btn"
                    title="Add subpage"
                    onclick={(e) => { e.stopPropagation(); createPage(page.id) }}>+</button
                  >
                  <button
                    class="action-btn"
                    title="Move to..."
                    onclick={(e) => { e.stopPropagation(); movePage(page.id) }}>↗</button
                  >
                  <button
                    class="action-btn delete"
                    title="Archive page"
                    onclick={(e) => { e.stopPropagation(); deletePage(page.id) }}>×</button
                  >
                </div>
              </div>
            </div>
          {/each}
        {/if}
      </div>
    </div>

    <!-- Improved Manual Link Section -->
    {#if connectionStatus !== "connected"}
      <div class="sidebar-section manual-connect-card">
        <div class="section-label">Manual Link</div>
        <div class="manual-form">
          <div class="form-group">
            <label for="manual-ip">IP Address</label>
            <input
              type="text"
              id="manual-ip"
              bind:value={manualIp}
              placeholder="192.168.1.X"
            />
          </div>
          <div class="form-group">
            <label for="manual-port">Port</label>
            <input
              type="number"
              id="manual-port"
              bind:value={manualPort}
              placeholder="9090"
            />
          </div>
          <div class="form-group">
            <label for="manual-pin">Auth PIN (from mobile)</label>
            <input
              type="text"
              id="manual-pin"
              bind:value={manualPin}
              placeholder="••••"
              maxlength="4"
              class="pin-input"
            />
          </div>
          <button class="connect-btn" onclick={handleConnectManually}>
            <span class="btn-icon" style="display: inline-flex; align-items: center;">
              <svg class="lucide-icon" xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/></svg>
            </span> Link Device
          </button>
          {#if connectionError}
            <div class="connection-error-box">
              <span class="err-icon" style="display: inline-flex; align-items: center; color: #f87171;">
                <svg class="lucide-icon" xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3Z"/><line x1="12" x2="12" y1="9" y2="13"/><line x1="12" x2="12.01" y1="17" y2="17"/></svg>
              </span>
              <span class="err-msg">{connectionError}</span>
            </div>
          {/if}
        </div>
      </div>
    {/if}
  </aside>

  <!-- LEFT DRAG HANDLE -->
  <div
    class="drag-handle left-handle"
    onmousedown={(e) => onDragStart("left", e)}
    class:active={dragging === "left"}
  >
    <div class="handle-line"></div>
  </div>

  <!-- EDITOR WORKSPACE -->
  <section class="editor-workspace">
    {#if connectionStatus !== "connected"}
      <div class="welcome">
        <div class="welcome-card">
          <div class="pulse">
            <svg class="logo-welcome-svg" viewBox="0 0 100 100" fill="none" xmlns="http://www.w3.org/2000/svg">
              <defs>
                <linearGradient id="welcome-logo-grad" x1="0%" y1="0%" x2="100%" y2="100%">
                  <stop offset="0%" stop-color="#818cf8"/>
                  <stop offset="100%" stop-color="#c084fc"/>
                </linearGradient>
              </defs>
              <circle cx="50" cy="50" r="32" stroke="url(#welcome-logo-grad)" stroke-width="12" stroke-linecap="round"/>
              <path d="M41 28 L59 72" stroke="url(#welcome-logo-grad)" stroke-width="12" stroke-linecap="round"/>
            </svg>
          </div>
          <h1>Cero Journal</h1>
          <p>Connect to your mobile phone to sync notes in real time.</p>
          <div class="steps">
            <div class="step">
              <span class="num">1</span>
              <p>Start <strong>Cero</strong> on phone</p>
            </div>
            <div class="step">
              <span class="num">2</span>
              <p>Enable <strong>Sync Hub</strong></p>
            </div>
            <div class="step">
              <span class="num">3</span>
              <p>Link device on left</p>
            </div>
          </div>
        </div>
      </div>
    {:else if !selectedPage}
      <div class="welcome">
        <div class="empty-page">
          <span class="big-icon"><PageIcon emoji="" size={38} /></span>
          <h2>Select or Create a Page</h2>
          <p>Choose from sidebar or create a new page.</p>
          <button class="btn primary" onclick={() => createPage("")}
            >Create New Page</button
          >
        </div>
      </div>
    {:else}
      <div class="editor-header">
        <div class="breadcrumbs">
          {#if navigationHistory.length > 0}
            <button class="back-btn" onclick={goBack}>← Back</button>
            <span class="sep">|</span>
          {/if}
          <span class="bc-root">{activeWorkspace}</span>
          <span class="sep">/</span>
          <span class="bc-current">
            <PageIcon emoji={selectedPage.emoji} size={12} />
            {selectedPage.title || "Untitled"}
          </span>
          {#if isSyncing}
            <span class="sync-pill" style="margin-left: 10px; display: inline-flex; align-items: center; gap: 4px; background: rgba(129, 140, 248, 0.1); border: 1px solid rgba(129, 140, 248, 0.2); padding: 2px 8px; border-radius: 12px; font-size: 9px; color: #818cf8; font-weight: bold;">
              <span class="sync-spinner-mini"></span> Syncing...
            </span>
          {/if}
        </div>
        <div class="header-actions">
          <button
            class="btn-sm"
            style="background: {showScratchpad ? 'rgba(129, 140, 248, 0.15)' : 'rgba(255, 255, 255, 0.02)'}; border-color: {showScratchpad ? '#818cf8' : 'rgba(255, 255, 255, 0.05)'}; color: {showScratchpad ? '#818cf8' : '#cbd5e1'}; font-weight: bold;"
            onclick={toggleScratchpad}
          >
            <svg class="lucide-icon" xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="margin-right: 4px;"><path d="M12 20h9"/><path d="M16.5 3.5a2.12 2.12 0 0 1 3 3L7 19l-4 1 1-4Z"/></svg>
            Scratchpad
          </button>
          <button
            class="btn-sm"
            style="background: {showTally ? 'rgba(129, 140, 248, 0.15)' : 'rgba(255, 255, 255, 0.02)'}; border-color: {showTally ? '#818cf8' : 'rgba(255, 255, 255, 0.05)'}; color: {showTally ? '#818cf8' : '#cbd5e1'}; font-weight: bold; margin-left: 4px;"
            onclick={toggleTally}
          >
            <svg class="lucide-icon" xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="margin-right: 4px;"><line x1="4" x2="20" y1="9" y2="9"/><line x1="4" x2="20" y1="15" y2="15"/><line x1="10" x2="8" y1="3" y2="21"/><line x1="16" x2="14" y1="3" y2="21"/></svg>
            Tallies
          </button>
          <button class="btn-sm" onclick={() => movePage(selectedPage.id)}
            >Move</button
          >
          <button
            class="btn-sm danger"
            onclick={() => deletePage(selectedPage.id)}>Archive</button
          >
          <button
            class="btn-sm sidebar-toggle"
            title="{showRightSidebar ? 'Hide' : 'Show'} context panel (Ctrl+\)"
            onclick={() => (showRightSidebar = !showRightSidebar)}
          >
            {showRightSidebar ? '✕ Close Context' : '☰ Context'}
          </button>
        </div>
      </div>

      <!-- SIDE-BY-SIDE RESPONSIVE WRAPPER CONTAINER -->
      <div class="workspace-body-container" style="display: flex; flex: 1; overflow: hidden; position: relative; width: 100%; flex-direction: {(showScratchpad ? scratchpadLayout : tallyLayout) === 'bottom' ? 'column' : 'row'};">
        <div class="main-editor-pane" style="display: flex; flex-direction: column; flex: 1; min-height: 0; min-width: 0; overflow: hidden; position: relative; height: {(showScratchpad ? scratchpadLayout : tallyLayout) === 'bottom' ? 'auto' : '100%'};">
          <div class="title-bar">
            <!-- Interactive Page Emoji Picker Trigger -->
            <button
              class="emoji-input-btn"
              onclick={() => (showEmojiPickerModal = true)}
            >
              <PageIcon emoji={editorEmoji || ""} size={44} />
            </button>
            <input
              id="editor-title-input"
              class="title-input"
              type="text"
              bind:value={editorTitle}
              oninput={savePageDebounced}
              placeholder="Untitled"
            />
          </div>

          <div class="card-column" bind:this={cardColumnContainer} onscroll={handleCardColumnScroll}>
            {#if isPaginatedView && pageCards.length > 0}
              {@const card = pageCards[currentBlockIndex]}
              <div class="card-slot">
                <CardBlock
                  {card}
                  isSelected={selectedCardId === card.id}
                  index={currentBlockIndex + 1}
                  {allPages}
                  {activeLiveCardId}
                  {activeSitesLocalUrl}
                  onSelect={selectCard}
                  onNavigate={selectPage}
                  onDeleted={(id) => {
                    pageCards = pageCards.filter((c) => c.id !== id);
                    if (currentBlockIndex >= pageCards.length) {
                      currentBlockIndex = pageCards.length > 0 ? pageCards.length - 1 : 0;
                      blockNumberInput = (currentBlockIndex + 1).toString();
                    }
                  }}
                  oneditMarkdown={(content) => openMarkdownFullscreen(card, content)}
                  oneditCode={(content) => openCodeFullscreen(card, content)}
                  oneditSites={(name, desc, html) => openSitesFullscreen(card, name, desc, html)}
                  onpreviewSites={(name, html) => openSitesPreview(name, html)}
                  onopenLinkModal={(c) => {
                    linkingCard = c;
                    pageSearchQuery = "";
                    showLinkPageModal = true;
                  }}
                  onopenCommentsModal={(c) => openCommentsModal(c)}
                  onopenMoveBlockModal={(c) => openMoveBlockModal(c)}
                  onReadFullscreen={(c) => {
                    readingCard = c;
                    readingViewTab = "read";
                    showReadingFullscreenModal = true;
                  }}
                  onmoveUp={() => moveCard(currentBlockIndex, -1)}
                  onmoveDown={() => moveCard(currentBlockIndex, 1)}
                />
              </div>
            {:else if pageCards.length > 0}
              {#each pageCards as card, index (card.id)}
                <div
                  class="card-slot"
                  draggable="true"
                  ondragstart={(e) => {
                    e.dataTransfer.setData("text/plain", index.toString());
                    e.dataTransfer.effectAllowed = "move";
                  }}
                  ondragover={(e) => { e.preventDefault(); {
                    e.dataTransfer.dropEffect = "move";
                   }}}
                  ondrop={(e) => { e.preventDefault(); handleCardDrop(e, index) }}
                >
                  <CardBlock
                    {card}
                    isSelected={selectedCardId === card.id}
                    index={index + 1}
                    {allPages}
                    {activeLiveCardId}
                    {activeSitesLocalUrl}
                    onSelect={selectCard}
                    onNavigate={selectPage}
                    onDeleted={(id) => {
                      pageCards = pageCards.filter((c) => c.id !== id);
                    }}
                    oneditMarkdown={(content) => openMarkdownFullscreen(card, content)}
                    oneditCode={(content) => openCodeFullscreen(card, content)}
                    oneditSites={(name, desc, html) => openSitesFullscreen(card, name, desc, html)}
                    onpreviewSites={(name, html) => openSitesPreview(name, html)}
                    onopenLinkModal={(c) => {
                      linkingCard = c;
                      pageSearchQuery = "";
                      showLinkPageModal = true;
                    }}
                    onopenCommentsModal={(c) => openCommentsModal(c)}
                    onopenMoveBlockModal={(c) => openMoveBlockModal(c)}
                    onReadFullscreen={(c) => {
                      readingCard = c;
                      readingViewTab = "read";
                      showReadingFullscreenModal = true;
                    }}
                    onmoveUp={() => moveCard(index, -1)}
                    onmoveDown={() => moveCard(index, 1)}
                  />
                </div>
                <div
                  class="insert-slot"
                  ondragover={(e) => e.preventDefault()}
                  ondrop={(e) => { e.preventDefault(); handleCardDrop(e, index + 1) }}
                >
                  <button
                    class="insert-btn"
                    onclick={() => {
                      blockInsertIndex = index + 1;
                      showBlockSelectorModal = true;
                    }}>+</button
                  >
                </div>
              {/each}
            {/if}

            {#if pageCards.length === 0}
              <div class="empty-cards">
                <span class="empty-icon"><PageIcon emoji="folder" size={24} /></span>
                <h3>This page is empty</h3>
                <p>Add blocks to start writing.</p>
                <div class="empty-actions">
                  <button class="btn secondary" onclick={() => handleAddCardType("markdown")}>
                    <svg class="lucide-icon" xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="margin-right: 4px;"><path d="M12 20h9"/><path d="M16.5 3.5a2.12 2.12 0 0 1 3 3L7 19l-4 1 1-4Z"/></svg>
                    Markdown
                  </button>
                  <button class="btn secondary" onclick={() => handleAddCardType("image")}>🖼️ Image</button>
                  <button class="btn secondary" onclick={() => handleAddCardType("subpage_link")}>🔗 Link</button>
                  <button class="btn secondary" onclick={() => handleAddCardType("code")}>💻 Code Block</button>
                  <button class="btn secondary" onclick={() => handleAddCardType("sites")}>🌐 HTML Site</button>
                  <button class="btn secondary" onclick={() => handleAddCardType("section")}>📋 Section</button>
                </div>
              </div>
            {/if}

            {#if !isPaginatedView}
              <button
                class="add-card-btn"
                onclick={() => {
                  blockInsertIndex = null;
                  showBlockSelectorModal = true;
                }}>+ Add Card</button
              >
            {/if}

            {#if showScrollToBottom && !isPaginatedView}
              <button class="scroll-to-bottom-btn" onclick={scrollToBottom}>
                ↓
              </button>
            {/if}
          </div>

          <!-- Persistent workspace-bottom-bar sits right here inside the workspace parent, below the card-column -->
          <div class="workspace-bottom-bar">
            <!-- Unified Mode Control (Left) -->
            <button
              class="mode-toggle-btn"
              class:block-active={isPaginatedView}
              onclick={() => {
                isPaginatedView = !isPaginatedView;
                if (isPaginatedView) {
                  currentBlockIndex = 0;
                  blockNumberInput = "1";
                }
              }}
              title={isPaginatedView ? "Switch to Scroll Mode" : "Switch to Block Mode"}
            >
              {#if isPaginatedView}
                <span class="btn-icon" style="display: inline-flex; align-items: center;">
                  <svg class="lucide-icon" xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="18" height="18" x="3" y="3" rx="2"/></svg>
                </span> <span class="lbl-text">Block View</span>
              {:else}
                <span class="btn-icon" style="display: inline-flex; align-items: center;">
                  <svg class="lucide-icon" xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="18" height="18" x="3" y="3" rx="2" /><path d="M3 9h18"/><path d="M3 15h18"/></svg>
                </span> <span class="lbl-text">Scroll View</span>
              {/if}
            </button>

            <!-- Super Compact Pagination (Right) with Jump buttons -->
            {#if isPaginatedView && pageCards.length > 0}
              <div class="pagination-compact">
                <!-- Jump to First Block Button -->
                <button
                  class="nav-arrow-btn"
                  disabled={currentBlockIndex === 0}
                  onclick={() => {
                    currentBlockIndex = 0;
                    blockNumberInput = "1";
                  }}
                  title="First Block"
                >
                  &laquo;
                </button>

                <!-- Previous Block Button -->
                <button
                  class="nav-arrow-btn"
                  disabled={currentBlockIndex === 0}
                  onclick={() => {
                    currentBlockIndex = Math.max(0, currentBlockIndex - 1);
                    blockNumberInput = (currentBlockIndex + 1).toString();
                  }}
                  title="Previous Block"
                >
                  &larr;
                </button>
                
                <div class="pagination-info">
                  <input
                    type="number"
                    class="pagination-num-input"
                    min="1"
                    max={pageCards.length}
                    value={blockNumberInput}
                    onchange={(e) => {
                      const parsed = parseInt(e.target.value);
                      if (!isNaN(parsed) && parsed >= 1 && parsed <= pageCards.length) {
                        currentBlockIndex = parsed - 1;
                        blockNumberInput = parsed.toString();
                      } else {
                        blockNumberInput = (currentBlockIndex + 1).toString();
                      }
                    }}
                  />
                  <span class="divider-slash">/</span>
                  <span class="total-blocks">{pageCards.length}</span>
                </div>

                <!-- Next Block Button -->
                <button
                  class="nav-arrow-btn"
                  disabled={currentBlockIndex === pageCards.length - 1}
                  onclick={() => {
                    currentBlockIndex = Math.min(pageCards.length - 1, currentBlockIndex + 1);
                    blockNumberInput = (currentBlockIndex + 1).toString();
                  }}
                  title="Next Block"
                >
                  &rarr;
                </button>

                <!-- Jump to Last Block Button -->
                <button
                  class="nav-arrow-btn"
                  disabled={currentBlockIndex === pageCards.length - 1}
                  onclick={() => {
                    currentBlockIndex = pageCards.length - 1;
                    blockNumberInput = pageCards.length.toString();
                  }}
                  title="Last Block"
                >
                  &raquo;
                </button>
              </div>
            {:else}
              <div class="continuous-scroll-lbl">
                Continuous
              </div>
            {/if}
          </div>
        </div>

        {#if showScratchpad && scratchpadLayout === "side"}
          <!-- SIDE DRAG HANDLE -->
          <!-- svelte-ignore a11y-no-static-element-interactions -->
          <div
            class="drag-handle scratchpad-handle"
            onmousedown={(e) => onDragStart("scratchpad", e)}
            class:active={dragging === "scratchpad"}
            style="width: 4px; cursor: col-resize; display: flex; align-items: center; justify-content: center; flex-shrink: 0; background: transparent; transition: background 0.15s; z-index: 20;"
          >
            <div class="handle-line" style="width: 1px; height: 32px; background: rgba(255, 255, 255, 0.1);"></div>
          </div>

          <!-- SIDE SCRATCHPAD PANE -->
          <div
            class="scratchpad-pane"
            style="width: {scratchpadWidth}px; min-width: 220px; max-width: 600px; display: flex; flex-direction: column; background: #0c0c0e; border-left: 1px solid rgba(255, 255, 255, 0.05); overflow: hidden; flex-shrink: 0; height: 100%;"
          >
            {@render scratchpadContentTemplate()}
          </div>
        {/if}

        {#if showScratchpad && scratchpadLayout === "bottom"}
          <!-- VERTICAL DRAG HANDLE FOR HEIGHT RESIZING -->
          <!-- svelte-ignore a11y-no-static-element-interactions -->
          <div
            class="drag-handle scratchpad-height-handle"
            onmousedown={(e) => onDragStart("scratchpad_height", e)}
            class:active={dragging === "scratchpad_height"}
            style="height: 4px; cursor: row-resize; display: flex; align-items: center; justify-content: center; flex-shrink: 0; background: transparent; transition: background 0.15s; z-index: 20; width: 100%;"
          >
            <div class="handle-line" style="width: 32px; height: 1px; background: rgba(255, 255, 255, 0.15);"></div>
          </div>

          <!-- BOTTOM SCRATCHPAD PANE -->
          <div
            class="scratchpad-pane bottom-dock"
            style="height: {scratchpadHeight}px; min-height: 120px; max-height: 500px; display: flex; flex-direction: column; background: #0c0c0e; border-top: 1px solid rgba(255, 255, 255, 0.05); overflow: hidden; flex-shrink: 0; width: 100%;"
          >
            {@render scratchpadContentTemplate()}
          </div>
        {/if}

        {#if showTally && tallyLayout === "side"}
          <!-- svelte-ignore a11y-no-static-element-interactions -->
          <div
            class="drag-handle scratchpad-handle"
            onmousedown={(e) => onDragStart("scratchpad", e)}
            class:active={dragging === "scratchpad"}
            style="width: 4px; cursor: col-resize; display: flex; align-items: center; justify-content: center; flex-shrink: 0; background: transparent; transition: background 0.15s; z-index: 20;"
          >
            <div class="handle-line" style="width: 1px; height: 32px; background: rgba(255, 255, 255, 0.1);"></div>
          </div>

          <div
            class="scratchpad-pane"
            style="width: {tallyWidth}px; min-width: 220px; max-width: 600px; display: flex; flex-direction: column; background: #0c0c0e; border-left: 1px solid rgba(255, 255, 255, 0.05); overflow: hidden; flex-shrink: 0; height: 100%;"
          >
            {@render tallyContentTemplate()}
          </div>
        {/if}

        {#if showTally && tallyLayout === "bottom"}
          <!-- svelte-ignore a11y-no-static-element-interactions -->
          <div
            class="drag-handle scratchpad-height-handle"
            onmousedown={(e) => onDragStart("scratchpad_height", e)}
            class:active={dragging === "scratchpad_height"}
            style="height: 4px; cursor: row-resize; display: flex; align-items: center; justify-content: center; flex-shrink: 0; background: transparent; transition: background 0.15s; z-index: 20; width: 100%;"
          >
            <div class="handle-line" style="width: 32px; height: 1px; background: rgba(255, 255, 255, 0.15);"></div>
          </div>

          <div
            class="scratchpad-pane bottom-dock"
            style="height: {tallyHeight}px; min-height: 120px; max-height: 500px; display: flex; flex-direction: column; background: #0c0c0e; border-top: 1px solid rgba(255, 255, 255, 0.05); overflow: hidden; flex-shrink: 0; width: 100%;"
          >
            {@render tallyContentTemplate()}
          </div>
        {/if}
      </div>
    {/if}

  </section>

  {#if showRightSidebar}
    <!-- RIGHT DRAG HANDLE -->
    <div
      class="drag-handle right-handle"
      onmousedown={(e) => onDragStart("right", e)}
      class:active={dragging === "right"}
    >
      <div class="handle-line"></div>
    </div>

    <!-- RIGHT SIDEBAR -->
    <aside
      class="sidebar right-sidebar"
      style="width: {rightSidebarWidth}px; min-width: {rightSidebarWidth}px;"
    >
      <RightSidebar
        {sidePages}
        {selectedPage}
        onSelectPage={selectPage}
        onCreateSidePage={createSidePage}
        onDeletePage={deletePage}
      />
    </aside>
  {/if}

  <!-- GLOBAL SNIPPETS (Placed at root level for global component access) -->
  {#snippet scratchpadContentTemplate()}
    <div class="scratchpad-header" style="display: flex; align-items: center; justify-content: space-between; padding: 10px 14px; border-bottom: 1px solid rgba(255,255,255,0.05); background: #121215; flex-shrink: 0;">
      <span style="font-size: 11px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.5px; color: #818cf8; display: flex; align-items: center; gap: 4px;">
        <svg class="lucide-icon" xmlns="http://www.w3.org/2000/svg" width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 20h9"/><path d="M16.5 3.5a2.12 2.12 0 0 1 3 3L7 19l-4 1 1-4Z"/></svg>
        Scratchpad</span>
      <div style="display: flex; align-items: center; gap: 4px;">
        <button
          class="btn-sm"
          style="padding: 2px 6px; font-size: 9px;"
          onclick={() => (scratchpadLayout = scratchpadLayout === "side" ? "bottom" : "side")}
          title="Toggle Dock Layout (Side / Bottom)"
        >
          {scratchpadLayout === "side" ? "📥 Bottom" : "🔲 Side"}
        </button>
        <button
          class="btn-sm"
          style="padding: 2px 6px; font-size: 9px; {scratchpadTab === 'write' ? 'background: rgba(129, 140, 248, 0.1); color: #818cf8; border-color: rgba(129,140,248,0.2);' : ''}"
          onclick={() => (scratchpadTab = "write")}
        >
          Write
        </button>
        <button
          class="btn-sm"
          style="padding: 2px 6px; font-size: 9px; {scratchpadTab === 'preview' ? 'background: rgba(129, 140, 248, 0.1); color: #818cf8; border-color: rgba(129,140,248,0.2);' : ''}"
          onclick={() => (scratchpadTab = "preview")}
        >
          Preview
        </button>
        <button
          class="close-btn"
          style="font-size: 14px; padding: 2px 6px; margin-left: 4px;"
          onclick={() => (showScratchpad = false)}
        >
          &times;
        </button>
      </div>
    </div>
    
    <div class="scratchpad-content-area" style="flex: 1; display: flex; flex-direction: column; overflow: hidden; position: relative;">
      {#if scratchpadTab === "write"}
        <textarea
          id="scratchpad-textarea"
          style="flex: 1; background: transparent; border: none; resize: none; outline: none; color: #e4e4e7; font-family: inherit; font-size: 12px; line-height: 1.6; padding: 14px; box-sizing: border-box; text-align: left;"
          bind:value={scratchpadContent}
          oninput={saveScratchpadDebounced}
          placeholder="Take quick notes here while reading your main page..."
        ></textarea>
      {:else}
        <div class="scratchpad-preview-rendered markdown-rendered" style="flex: 1; overflow-y: auto; padding: 14px; background: #09090b; text-align: left; height: 100%; overflow-wrap: break-word; word-break: break-word;">
          {#if scratchpadContent.trim()}
            {#snippet codeSnippet({ lang, text })}
              <pre class="markdown-code-block"><div class="code-lang-badge">{(lang || '').toUpperCase() || 'CODE'}</div><code>{@html highlightCode(text, lang || '')}</code></pre>
            {/snippet}
            <SvelteMarkdown source={scratchpadContent} extensions={[markedKatex({ singleDollarInline: true })]} renderers={{ inlineKatex: KatexRenderer, blockKatex: KatexRenderer }} code={codeSnippet} />
          {:else}
            <span class="empty-hint" style="font-style: italic; font-size: 11px; opacity: 0.5;">No content to preview. Type something in the Write tab!</span>
          {/if}
        </div>
      {/if}
    </div>
  {/snippet}

  {#snippet tallyContentTemplate()}
    <div class="scratchpad-header" style="display: flex; align-items: center; justify-content: space-between; padding: 10px 14px; border-bottom: 1px solid rgba(255,255,255,0.05); background: #121215; flex-shrink: 0;">
      <span style="font-size: 11px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.5px; color: #818cf8; display: flex; align-items: center; gap: 4px;">
        <svg class="lucide-icon" xmlns="http://www.w3.org/2000/svg" width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="4" x2="20" y1="9" y2="9"/><line x1="4" x2="20" y1="15" y2="15"/><line x1="10" x2="8" y1="3" y2="21"/><line x1="16" x2="14" y1="3" y2="21"/></svg>
        Tallies</span>
      <div style="display: flex; align-items: center; gap: 4px;">
        <button
          class="btn-sm"
          style="padding: 2px 6px; font-size: 9px;"
          onclick={() => (tallyLayout = tallyLayout === "side" ? "bottom" : "side")}
          title="Toggle Dock Layout (Side / Bottom)"
        >
          {tallyLayout === "side" ? "📥 Bottom" : "🔲 Side"}
        </button>
        <button
          class="close-btn"
          style="font-size: 14px; padding: 2px 6px; margin-left: 4px;"
          onclick={() => (showTally = false)}
        >
          &times;
        </button>
      </div>
    </div>
    
    <div class="scratchpad-content-area" style="flex: 1; display: flex; flex-direction: column; overflow: hidden; position: relative;">
      <div style="display: flex; padding: 10px; gap: 6px; border-bottom: 1px solid rgba(255, 255, 255, 0.05); background: #141416;">
        <input
          type="text"
          placeholder="New Tally Name..."
          bind:value={newTallyName}
          onkeydown={(e) => { if (e.key === "Enter") addTally(); }}
          style="flex: 1; background: #1c1c1e; border: 1px solid rgba(255,255,255,0.05); border-radius: 6px; padding: 6px 10px; color: white; font-size: 12px; outline: none;"
        />
        <button
          class="btn primary"
          style="padding: 6px 12px; font-size: 12px;"
          onclick={addTally}
        >
          Add
        </button>
      </div>

      <div style="flex: 1; overflow-y: auto; padding: 10px; display: flex; flex-direction: column; gap: 8px;">
        {#if tallyCards.length === 0}
          <div style="padding: 24px; text-align: center; color: #52525b; font-size: 12px; font-style: italic;">
            No tallies yet. Add one above!
          </div>
        {:else}
          {#each tallyCards as card (card.id)}
            {@const data = (() => {
              try {
                return JSON.parse(card.content);
              } catch (_) {
                return { name: card.comment || "Tally", count: parseInt(card.content) || 0 };
              }
            })()}
            <div style="display: flex; align-items: center; justify-content: space-between; background: #18181b; border: 1px solid rgba(255, 255, 255, 0.04); border-radius: 8px; padding: 8px 12px; gap: 12px;">
              <span style="font-size: 12.5px; font-weight: 700; color: #e4e4e7; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; flex: 1; text-align: left;">
                {data.name}
              </span>
              <div style="display: flex; align-items: center; gap: 4px;">
                <button
                  class="btn-sm"
                  style="padding: 2px 8px; font-size: 13px; color: #f87171; border-color: rgba(239, 68, 68, 0.15);"
                  disabled={data.count <= 0}
                  onclick={() => updateTallyCount(card, data.name, data.count - 1)}
                >
                  -
                </button>
                <span style="font-size: 13px; font-weight: 700; color: white; min-width: 24px; text-align: center;">
                  {data.count}
                </span>
                <button
                  class="btn-sm"
                  style="padding: 2px 8px; font-size: 13px; color: #4ade80; border-color: rgba(74, 222, 128, 0.15);"
                  onclick={() => updateTallyCount(card, data.name, data.count + 1)}
                >
                  +
                </button>
                <button
                  class="close-btn"
                  style="font-size: 14px; padding: 2px 6px; margin-left: 8px; color: #71717a;"
                  onclick={() => deleteTally(card.id)}
                  title="Delete Tally"
                >
                  &times;
                </button>
              </div>
            </div>
          {/each}
        {/if}
      </div>
    </div>
  {/snippet}
</main>

<svelte:window onkeydown={(e) => {
  if (e.key === '\\' && (e.ctrlKey || e.metaKey)) {
    e.preventDefault();
    showRightSidebar = !showRightSidebar;
  }
}} />

<!-- Reusable Sleek Dialog Portal (Alert & Confirm Modal) -->
{#if modalConfig.show}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <div 
    class="modal-backdrop" 
    onclick={(e) => { if (e.target === e.currentTarget) { { if (modalConfig.type === 'alert') modalConfig.onConfirm();  } }}} 
    role="button" 
    tabindex="-1"
  >
    <div class="modal-container alert-confirm-modal">
      <div class="modal-header">
        <h3>{modalConfig.title}</h3>
        {#if modalConfig.type === 'alert'}
          <button class="close-btn" onclick={modalConfig.onConfirm}>&times;</button>
        {:else}
          <button class="close-btn" onclick={modalConfig.onCancel}>&times;</button>
        {/if}
      </div>
      <div class="modal-body alert-confirm-body">
        <p>{modalConfig.message}</p>
      </div>
      <div class="modal-footer">
        {#if modalConfig.type === 'confirm'}
          <button class="btn secondary" onclick={modalConfig.onCancel}>{modalConfig.cancelText}</button>
        {/if}
        <button class="btn primary" onclick={modalConfig.onConfirm}>{modalConfig.confirmText}</button>
      </div>
    </div>
  </div>
{/if}

<!-- Block Selector Modal -->
{#if showBlockSelectorModal}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <div
    class="modal-backdrop"
    onclick={(e) => { if (e.target === e.currentTarget) { (showBlockSelectorModal = false) } }}
    role="button"
    tabindex="-1"
  >
    <div class="modal-container block-selector-modal">
      <div class="modal-header">
        <h3>Add Block</h3>
        <button
          class="close-btn"
          onclick={() => (showBlockSelectorModal = false)}>&times;</button
        >
      </div>
      <div class="modal-body block-options-grid">
        <button
          class="block-option-row"
          onclick={() => handleAddCardType("markdown")}
        >
          <span class="block-icon" style="display: inline-flex; align-items: center; color: #818cf8;">
            <svg class="lucide-icon" xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 20h9"/><path d="M16.5 3.5a2.12 2.12 0 0 1 3 3L7 19l-4 1 1-4Z"/></svg>
          </span>
          <div class="block-desc">
            <span class="block-title">Markdown</span>
            <span class="block-subtitle"
              >Write formatted text, headers, checklist items, or code blocks.</span
            >
          </div>
        </button>
        <button
          class="block-option-row"
          onclick={() => handleAddCardType("image")}
        >
          <span class="block-icon" style="display: inline-flex; align-items: center; color: #818cf8;">
            <svg class="lucide-icon" xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="18" height="18" x="3" y="3" rx="2" ry="2"/><circle cx="9" cy="9" r="2"/><path d="m21 15-3.08-3.08a2 2 0 0 0-2.83 0L12 15"/></svg>
          </span>
          <div class="block-desc">
            <span class="block-title">Image</span>
            <span class="block-subtitle"
              >Embed an image from a web URL or upload directly from your
              computer.</span
            >
          </div>
        </button>
        <button
          class="block-option-row"
          onclick={() => handleAddCardType("subpage_link")}
        >
          <span class="block-icon" style="display: inline-flex; align-items: center; color: #818cf8;">
            <svg class="lucide-icon" xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/></svg>
          </span>
          <div class="block-desc">
            <span class="block-title">Subpage Link</span>
            <span class="block-subtitle"
              >Link directly to another nested page in your journal hierarchy.</span
            >
          </div>
        </button>
        <button
          class="block-option-row"
          onclick={() => handleAddCardType("code")}
        >
          <span class="block-icon" style="display: inline-flex; align-items: center; color: #818cf8;">
            <svg class="lucide-icon" xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/></svg>
          </span>
          <div class="block-desc">
            <span class="block-title">Code Block</span>
            <span class="block-subtitle"
              >Syntax-highlighted editor with multi-programming language
              options.</span
            >
          </div>
        </button>
        <button
          class="block-option-row"
          onclick={() => handleAddCardType("sites")}
        >
          <span class="block-icon" style="display: inline-flex; align-items: center; color: #818cf8;">
            <svg class="lucide-icon" xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="2" x2="22" y1="12" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>
          </span>
          <div class="block-desc">
            <span class="block-title">HTML Site block</span>
            <span class="block-subtitle"
              >Creates customized templates with embedded sandbox HTML code
              rendering.</span
            >
          </div>
        </button>
        <button
          class="block-option-row"
          onclick={() => handleAddCardType("section")}
        >
          <span class="block-icon" style="display: inline-flex; align-items: center; color: #818cf8;">
            <svg class="lucide-icon" xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="3" x2="21" y1="6" y2="6"/><line x1="3" x2="21" y1="12" y2="12"/><line x1="3" x2="21" y1="18" y2="18"/></svg>
          </span>
          <div class="block-desc">
            <span class="block-title">Section Divider</span>
            <span class="block-subtitle"
              >Clean typography header spanning full page width to segment content areas.</span
            >
          </div>
        </button>
      </div>
    </div>
  </div>
{/if}

<!-- Page Icon Picker Modal -->
{#if showEmojiPickerModal}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <div
    class="modal-backdrop"
    onclick={(e) => { if (e.target === e.currentTarget) { (showEmojiPickerModal = false) } }}
    role="button"
    tabindex="-1"
  >
    <div class="modal-container emoji-picker-modal" style="height: auto; max-height: 90vh; width: 400px;">
      <div class="modal-header">
        <h3>Select Page Icon</h3>
        <button
          class="close-btn"
          onclick={() => (showEmojiPickerModal = false)}>&times;</button
        >
      </div>
      <input type="file" bind:this={iconImageInput} accept="image/*" style="display:none" onchange={handleIconUpload} />
      <div class="modal-body">
        <div class="custom-icon-section">
          <div class="custom-icon-header">
            <span class="custom-icon-icon">🖼️</span>
            <h4>Use Custom Page Icon</h4>
            <p>Import an image from your computer or use an online URL.</p>
          </div>
          <button class="btn primary" style="width:100%; margin-bottom: 12px;" onclick={() => iconImageInput.click()}>
            Pick Image from Computer
          </button>
          <div class="custom-icon-url-row" style="margin-bottom: 12px;">
            <input type="text" bind:value={customIconUrl} placeholder="https://example.com/icon.png" />
            <button class="btn primary" onclick={applyCustomIconUrl}>Apply</button>
          </div>
          {#if parentPage}
            <button class="btn secondary" style="width:100%; margin-bottom: 12px; color: #818cf8;" onclick={() => {
              editorEmoji = parentPage.emoji || "";
              showEmojiPickerModal = false;
              savePageImmediate();
            }}>
              Inherit Parent Icon ({parentPage.title || "Untitled"})
            </button>
          {/if}
          <button class="btn secondary" style="width:100%; color: #f87171;" onclick={() => {
            editorEmoji = "";
            showEmojiPickerModal = false;
            savePageImmediate();
          }}>
            Remove Custom Icon
          </button>
        </div>
      </div>
    </div>
  </div>
{/if}

<!-- Connection PIN Entry Modal -->
{#if showPinModal && pendingDeviceToConnect}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <div
    class="modal-backdrop"
    onclick={(e) => { if (e.target === e.currentTarget) { {
      if (!isConnecting) showPinModal = false;
     } }}}
    role="button"
    tabindex="-1"
  >
    <div class="modal-container pin-entry-modal">
      <div class="modal-header">
        <h3>🔑 Link Mobile Device</h3>
        <button
          class="close-btn"
          onclick={() => (showPinModal = false)}
          disabled={isConnecting}>&times;</button
        >
      </div>
      <div class="modal-body pin-modal-body">
        <div class="device-info-bar">
          <span class="device-icon">📱</span>
          <div class="device-details">
            <span class="device-name">{pendingDeviceToConnect.deviceName}</span>
            <span class="device-ip"
              >{pendingDeviceToConnect.ip}:{pendingDeviceToConnect.port}</span
            >
          </div>
        </div>

        <p class="pin-instructions">
          Please enter the 4-digit <strong>Auth PIN</strong> shown on your phone's
          Cero Sync Hub dashboard.
        </p>

        <div class="pin-input-container">
          <!-- svelte-ignore a11y-no-autofocus -->
          <input
            type="text"
            class="pin-display-input"
            bind:value={pinInput}
            placeholder="••••"
            maxlength="4"
            pattern="[0-9]*"
            inputmode="numeric"
            onkeydown={(e) => {
              if (e.key === "Enter") submitPin();
            }}
            disabled={isConnecting}
            autofocus
          />
        </div>

        {#if pinModalError}
          <div class="modal-error-box">
            <span class="err-icon">⚠️</span>
            <span class="err-msg">{pinModalError}</span>
          </div>
        {/if}
      </div>
      <div class="modal-footer">
        <button
          class="btn secondary"
          onclick={() => (showPinModal = false)}
          disabled={isConnecting}
        >
          Cancel
        </button>
        <button
          class="btn primary"
          onclick={submitPin}
          disabled={isConnecting || pinInput.length < 4}
        >
          {#if isConnecting}
            <span class="spinner"></span> Connecting...
          {:else}
            Link Device
          {/if}
        </button>
      </div>
    </div>
  </div>
{/if}

<!-- Immersive Markdown Fullscreen Modal -->
{#if showMarkdownFullscreenModal}
  <div class="fullscreen-editor-overlay markdown-fullscreen-theme">
    <div class="fullscreen-header">
      <div class="header-info">
        <div class="header-title-row">
          <span class="header-badge">MARKDOWN</span>
          <h3>Immersive Workspace</h3>
        </div>
        <span class="sub-desc"
          >Writing canvas with live split-screen preview</span
        >
      </div>

      <!-- Live Typing Stats Indicator -->
      <div class="editor-stats">
        <div class="stat-pill">
          <span class="stat-val">{markdownWordCount}</span>
          <span class="stat-lbl">words</span>
        </div>
        <div class="stat-pill">
          <span class="stat-val">{markdownCharCount}</span>
          <span class="stat-lbl">chars</span>
        </div>
      </div>

      <!-- Action Toggles for Scratchpad & Tallies inside Fullscreen mode -->
      <div style="display: flex; gap: 8px; margin-right: 12px; margin-left: auto;">
        <button
          class="btn-sm"
          style="background: {showScratchpad ? 'rgba(129, 140, 248, 0.15)' : 'rgba(255, 255, 255, 0.02)'}; border-color: {showScratchpad ? '#818cf8' : 'rgba(255, 255, 255, 0.05)'}; color: {showScratchpad ? '#818cf8' : '#cbd5e1'}; font-weight: bold;"
          onclick={toggleScratchpad}
        >
          <svg class="lucide-icon" xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="margin-right: 4px;"><path d="M12 20h9"/><path d="M16.5 3.5a2.12 2.12 0 0 1 3 3L7 19l-4 1 1-4Z"/></svg>
          Scratchpad
        </button>
        <button
          class="btn-sm"
          style="background: {showTally ? 'rgba(129, 140, 248, 0.15)' : 'rgba(255, 255, 255, 0.02)'}; border-color: {showTally ? '#818cf8' : 'rgba(255, 255, 255, 0.05)'}; color: {showTally ? '#818cf8' : '#cbd5e1'}; font-weight: bold; margin-left: 4px;"
          onclick={toggleTally}
        >
          <svg class="lucide-icon" xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="margin-right: 4px;"><line x1="4" x2="20" y1="9" y2="9"/><line x1="4" x2="20" y1="15" y2="15"/><line x1="10" x2="8" y1="3" y2="21"/><line x1="16" x2="14" y1="3" y2="21"/></svg>
          Tallies
        </button>
      </div>

      <div class="header-buttons">
        <button
          class="btn secondary"
          onclick={() => (showMarkdownFullscreenModal = false)}>Cancel</button
        >
        <button class="btn primary" onclick={saveMarkdownFullscreen}
          >Save Changes</button
        >
      </div>
    </div>

    <div class="fullscreen-workspace" style="display: flex; flex: 1; overflow: hidden; position: relative; width: 100%; flex-direction: {(showScratchpad ? scratchpadLayout : tallyLayout) === 'bottom' ? 'column' : 'row'};">
      <div class="fullscreen-editor-columns-wrap" style="display: flex; flex: 1; min-height: 0; min-width: 0; overflow: hidden; position: relative; width: 100%; height: {(showScratchpad ? scratchpadLayout : tallyLayout) === 'bottom' ? 'auto' : '100%'};">
        <div class="editor-pane-container" style="flex: 1;">
          <textarea
            bind:this={markdownTextarea}
            class="fullscreen-textarea"
            bind:value={fullscreenEditContent}
            placeholder="Start typing your thoughts in markdown..."
            spellcheck="true"
          ></textarea>

          <!-- Interactive Formatting Toolbar -->
          <div class="floating-markdown-toolbar">
            <button
              class="tool-btn"
              title="Bold"
              onclick={() => insertMarkdownSymbol("**", "**")}
              ><strong>B</strong></button
            >
            <button
              class="tool-btn"
              title="Italic"
              onclick={() => insertMarkdownSymbol("*", "*")}><em>I</em></button
            >
            <button
              class="tool-btn"
              title="Header"
              onclick={() => insertMarkdownSymbol("### ")}>H3</button
            >
            <button
              class="tool-btn"
              title="List"
              onclick={() => insertMarkdownSymbol("- ")}>• List</button
            >
            <button
              class="tool-btn"
              title="Checklist"
              onclick={() => insertMarkdownSymbol("- [ ] ")}>☑ Todo</button
            >
            <button
              class="tool-btn"
              title="Inline Code"
              onclick={() => insertMarkdownSymbol("`", "`")}>&lt;/&gt;</button
            >
            <button
              class="tool-btn"
              title="Code Block"
              onclick={() => insertMarkdownSymbol("```\n", "\n```")}
              >Block</button
            >
          </div>
        </div>

        <div class="fullscreen-preview markdown-rendered" style="flex: 1; overflow-y: auto;">
          {#snippet codeSnippet({ lang, text })}
            <pre class="markdown-code-block"><div class="code-lang-badge">{(lang || '').toUpperCase() || 'CODE'}</div><code>{@html highlightCode(text, lang || '')}</code></pre>
          {/snippet}
          <SvelteMarkdown source={fullscreenEditContent || ""} extensions={[markedKatex({ singleDollarInline: true })]} renderers={{ inlineKatex: KatexRenderer, blockKatex: KatexRenderer }} code={codeSnippet} />
        </div>
      </div>

      {#if showScratchpad && scratchpadLayout === "side"}
        <!-- svelte-ignore a11y-no-static-element-interactions -->
        <div
          class="drag-handle scratchpad-handle"
          onmousedown={(e) => onDragStart("scratchpad", e)}
          class:active={dragging === "scratchpad"}
          style="width: 4px; cursor: col-resize; display: flex; align-items: center; justify-content: center; flex-shrink: 0; background: transparent; transition: background 0.15s; z-index: 20;"
        >
          <div class="handle-line" style="width: 1px; height: 32px; background: rgba(255, 255, 255, 0.1);"></div>
        </div>

        <div
          class="scratchpad-pane"
          style="width: {scratchpadWidth}px; min-width: 220px; max-width: 600px; display: flex; flex-direction: column; background: #0c0c0e; border-left: 1px solid rgba(255, 255, 255, 0.05); overflow: hidden; flex-shrink: 0; height: 100%;"
        >
          {@render scratchpadContentTemplate()}
        </div>
      {/if}

      {#if showScratchpad && scratchpadLayout === "bottom"}
        <!-- svelte-ignore a11y-no-static-element-interactions -->
        <div
          class="drag-handle scratchpad-height-handle"
          onmousedown={(e) => onDragStart("scratchpad_height", e)}
          class:active={dragging === "scratchpad_height"}
          style="height: 4px; cursor: row-resize; display: flex; align-items: center; justify-content: center; flex-shrink: 0; background: transparent; transition: background 0.15s; z-index: 20; width: 100%;"
        >
          <div class="handle-line" style="width: 32px; height: 1px; background: rgba(255, 255, 255, 0.15);"></div>
        </div>

        <div
          class="scratchpad-pane bottom-dock"
          style="height: {scratchpadHeight}px; min-height: 120px; max-height: 500px; display: flex; flex-direction: column; background: #0c0c0e; border-top: 1px solid rgba(255, 255, 255, 0.05); overflow: hidden; flex-shrink: 0; width: 100%;"
        >
          {@render scratchpadContentTemplate()}
        </div>
      {/if}

      {#if showTally && tallyLayout === "side"}
        <!-- svelte-ignore a11y-no-static-element-interactions -->
        <div
          class="drag-handle scratchpad-handle"
          onmousedown={(e) => onDragStart("scratchpad", e)}
          class:active={dragging === "scratchpad"}
          style="width: 4px; cursor: col-resize; display: flex; align-items: center; justify-content: center; flex-shrink: 0; background: transparent; transition: background 0.15s; z-index: 20;"
        >
          <div class="handle-line" style="width: 1px; height: 32px; background: rgba(255, 255, 255, 0.1);"></div>
        </div>

        <div
          class="scratchpad-pane"
          style="width: {tallyWidth}px; min-width: 220px; max-width: 600px; display: flex; flex-direction: column; background: #0c0c0e; border-left: 1px solid rgba(255, 255, 255, 0.05); overflow: hidden; flex-shrink: 0; height: 100%;"
        >
          {@render tallyContentTemplate()}
        </div>
      {/if}

      {#if showTally && tallyLayout === "bottom"}
        <!-- svelte-ignore a11y-no-static-element-interactions -->
        <div
          class="drag-handle scratchpad-height-handle"
          onmousedown={(e) => onDragStart("scratchpad_height", e)}
          class:active={dragging === "scratchpad_height"}
          style="height: 4px; cursor: row-resize; display: flex; align-items: center; justify-content: center; flex-shrink: 0; background: transparent; transition: background 0.15s; z-index: 20; width: 100%;"
        >
          <div class="handle-line" style="width: 32px; height: 1px; background: rgba(255, 255, 255, 0.15);"></div>
        </div>

        <div
          class="scratchpad-pane bottom-dock"
          style="height: {tallyHeight}px; min-height: 120px; max-height: 500px; display: flex; flex-direction: column; background: #0c0c0e; border-top: 1px solid rgba(255, 255, 255, 0.05); overflow: hidden; flex-shrink: 0; width: 100%;"
        >
          {@render tallyContentTemplate()}
        </div>
      {/if}
    </div>
  </div>
{/if}

<!-- Immersive Code Fullscreen Modal -->
{#if showCodeFullscreenModal}
  <div class="fullscreen-editor-overlay code-fullscreen-theme">
    <div class="fullscreen-header">
      <div
        class="header-info"
        style="display: flex; align-items: center; gap: 16px;"
      >
        <!-- Language Selector -->
        <div class="custom-select-wrapper">
          <select bind:value={fullscreenCodeLang} class="lang-dropdown-select">
            <option value="javascript">JAVASCRIPT</option>
            <option value="rust">RUST</option>
            <option value="python">PYTHON</option>
            <option value="html">HTML</option>
            <option value="css">CSS</option>
            <option value="go">GO</option>
            <option value="dart">DART</option>
            <option value="json">JSON</option>
            <option value="bash">BASH</option>
          </select>
        </div>
      </div>

      <!-- Code Stats -->
      <div class="editor-stats">
        <div class="stat-pill">
          <span class="stat-val">{lineNumbers}</span>
          <span class="stat-lbl">lines</span>
        </div>
        <div class="stat-pill">
          <span class="stat-val"
            >{fullscreenCodeContent ? fullscreenCodeContent.length : 0}</span
          > <span class="stat-lbl">bytes</span>
        </div>
      </div>

      <div class="header-buttons">
        <button
          class="btn secondary"
          onclick={() => (showCodeFullscreenModal = false)}>Cancel</button
        >
        <button class="btn primary" onclick={saveCodeFullscreen}
          >Save Changes</button
        >
      </div>
    </div>

    <div class="fullscreen-workspace single-pane">
      <div class="interactive-code-editor-viewport">
        <!-- Interactive Vertical Line Numbers Panel -->
        <div class="editor-line-numbers" bind:this={lineNumbersElement}>
          {#each Array(lineNumbers) as _, i}
            <div class="line-number-row">{i + 1}</div>
          {/each}
        </div>

        <!-- Textarea and Syntax Highlighter Canvas Container -->
        <div class="editor-canvas">
          <pre
            class="editor-pre"
            bind:this={preElement}>{@html highlightedCodeHtml}</pre>
          <textarea
            class="editor-textarea"
            bind:value={fullscreenCodeContent}
            onscroll={handleEditorScroll}
            placeholder="Write your code snippet here..."
            spellcheck="false"
            autofocus
          ></textarea>
        </div>
      </div>
    </div>
  </div>
{/if}

<!-- Fullscreen Reading Modal (Tabbed pages) -->
{#if showReadingFullscreenModal && readingCard}
  <div class="fullscreen-editor-overlay" style="display: flex; flex-direction: column; background: #09090b; height: 100vh; width: 100vw;">
    <div class="fullscreen-header">
      <div class="header-info">
        <span class="header-title">Reading Workspace — {readingCard.title || "Untitled"}</span>
      </div>
      <div class="editor-stats">
        {#if readingCard.type === "markdown"}
          <div class="stat-pill"><span class="stat-val">{readingCard.content ? readingCard.content.length : 0}</span><span class="stat-lbl">chars</span></div>
        {/if}
      </div>
      <div class="header-buttons">
        <button class="btn secondary" onclick={() => { showReadingFullscreenModal = false; }}>Close Workspace</button>
      </div>
    </div>

    <!-- Cohesive Segmented Tabs -->
    <div class="sites-tabs" style="display: flex; gap: 4px; background: #121212; border-bottom: 1px solid #2e2e2e; padding: 6px 12px; flex-shrink: 0;">
      <button
        class="emoji-tab-btn"
        class:active={readingViewTab === "read"}
        onclick={() => (readingViewTab = "read")}
        style="display: inline-flex; align-items: center; gap: 6px;"
      >
        <svg class="lucide-icon" xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M2 3h6a4 4 0 0 1 4 4v14a3 3 0 0 0-3-3H2Z"/><path d="M22 3h-6a4 4 0 0 0-4 4v14a3 3 0 0 1 3-3h7Z"/></svg>
        Reading View
      </button>
      <button
        class="emoji-tab-btn"
        class:active={readingViewTab === "scratchpad"}
        onclick={() => (readingViewTab = "scratchpad")}
        style="display: inline-flex; align-items: center; gap: 6px;"
      >
        <svg class="lucide-icon" xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 20h9"/><path d="M16.5 3.5a2.12 2.12 0 0 1 3 3L7 19l-4 1 1-4Z"/></svg>
        Scratchpad
      </button>
      <button
        class="emoji-tab-btn"
        class:active={readingViewTab === "tallies"}
        onclick={() => {
          readingViewTab = "tallies";
          FetchCards("global-tally");
        }}
        style="display: inline-flex; align-items: center; gap: 6px;"
      >
        <svg class="lucide-icon" xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="4" x2="20" y1="9" y2="9"/><line x1="4" x2="20" y1="15" y2="15"/><line x1="10" x2="8" y1="3" y2="21"/><line x1="16" x2="14" y1="3" y2="21"/></svg>
        Tallies
      </button>
    </div>

    <!-- Render active page view -->
    {#if readingViewTab === "read"}
      <div class="fullscreen-workspace single-pane" style="flex: 1; overflow-y: auto; padding: 32px; background: #09090b; display: flex; flex-direction: column;">
        <div class="markdown-rendered" style="max-width: 800px; width: 100%; margin: 0 auto; text-align: left; flex: 1; overflow-wrap: break-word; word-break: break-word;">
          {#snippet codeSnippet({ lang, text })}
            <pre class="markdown-code-block"><div class="code-lang-badge">{(lang || '').toUpperCase() || 'CODE'}</div><code>{@html highlightCode(text, lang || '')}</code></pre>
          {/snippet}
          {#if readingCard.type === "code"}
            <pre class="markdown-code-block" style="background: #09090b; border: 1px solid rgba(255, 255, 255, 0.08);"><div class="code-lang-badge">{(readingCodeLang || '').toUpperCase()}</div><code>{@html highlightCode(readingCodeContent, readingCodeLang)}</code></pre>
          {:else}
            <SvelteMarkdown source={readingCard.content || ""} extensions={[markedKatex({ singleDollarInline: true })]} renderers={{ inlineKatex: KatexRenderer, blockKatex: KatexRenderer }} code={codeSnippet} />
          {/if}
        </div>
      </div>
    {:else if readingViewTab === "scratchpad"}
      <div class="fullscreen-workspace" style="flex: 1; display: flex; flex-direction: column; background: #09090b; overflow: hidden;">
        <div class="scratchpad-header" style="display: flex; align-items: center; justify-content: space-between; padding: 10px 14px; border-bottom: 1px solid rgba(255,255,255,0.05); background: #121215; flex-shrink: 0;">
          <span style="font-size: 11px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.5px; color: #818cf8; display: flex; align-items: center; gap: 4px;">
            <svg class="lucide-icon" xmlns="http://www.w3.org/2000/svg" width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 20h9"/><path d="M16.5 3.5a2.12 2.12 0 0 1 3 3L7 19l-4 1 1-4Z"/></svg>
            Scratchpad Canvas</span>
          <div style="display: flex; align-items: center; gap: 4px;">
            <button
              class="btn-sm"
              style="padding: 4px 10px; font-size: 11px; {scratchpadTab === 'write' ? 'background: rgba(129, 140, 248, 0.1); color: #818cf8; border-color: rgba(129,140,248,0.2);' : ''}"
              onclick={() => (scratchpadTab = "write")}
            >
              Write
            </button>
            <button
              class="btn-sm"
              style="padding: 4px 10px; font-size: 11px; {scratchpadTab === 'preview' ? 'background: rgba(129, 140, 248, 0.1); color: #818cf8; border-color: rgba(129,140,248,0.2);' : ''}"
              onclick={() => (scratchpadTab = "preview")}
            >
              Preview
            </button>
          </div>
        </div>
        
        <div style="flex: 1; display: flex; flex-direction: column; overflow: hidden; position: relative;">
          {#if scratchpadTab === "write"}
            <textarea
              id="fullscreen-scratchpad-textarea"
              style="flex: 1; background: transparent; border: none; resize: none; outline: none; color: #e4e4e7; font-family: inherit; font-size: 14px; line-height: 1.7; padding: 24px; box-sizing: border-box; text-align: left;"
              bind:value={scratchpadContent}
              oninput={saveScratchpadDebounced}
              placeholder="Take fullscreen scratchpad notes here..."
            ></textarea>
          {:else}
            <div class="markdown-rendered" style="flex: 1; overflow-y: auto; padding: 32px; background: #09090b; text-align: left; height: 100%;">
              <div style="max-width: 800px; margin: 0 auto;">
                {#if scratchpadContent.trim()}
                  {#snippet codeSnippet({ lang, text })}
                    <pre class="markdown-code-block"><div class="code-lang-badge">{(lang || '').toUpperCase() || 'CODE'}</div><code>{@html highlightCode(text, lang || '')}</code></pre>
                  {/snippet}
                  <SvelteMarkdown source={scratchpadContent} extensions={[markedKatex({ singleDollarInline: true })]} renderers={{ inlineKatex: KatexRenderer, blockKatex: KatexRenderer }} code={codeSnippet} />
                {:else}
                  <span class="empty-hint" style="font-style: italic; font-size: 12px; opacity: 0.5;">No content to preview. Type something in the Write tab!</span>
                {/if}
              </div>
            </div>
          {/if}
        </div>
      </div>
    {:else if readingViewTab === "tallies"}
      <div class="fullscreen-workspace" style="flex: 1; display: flex; flex-direction: column; background: #09090b; overflow: hidden;">
        <div class="scratchpad-header" style="display: flex; align-items: center; justify-content: space-between; padding: 10px 14px; border-bottom: 1px solid rgba(255,255,255,0.05); background: #121215; flex-shrink: 0;">
          <span style="font-size: 11px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.5px; color: #818cf8; display: flex; align-items: center; gap: 4px;">
            <svg class="lucide-icon" xmlns="http://www.w3.org/2000/svg" width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="4" x2="20" y1="9" y2="9"/><line x1="4" x2="20" y1="15" y2="15"/><line x1="10" x2="8" y1="3" y2="21"/><line x1="16" x2="14" y1="3" y2="21"/></svg>
            Fullscreen Tallies</span>
        </div>

        <div style="max-width: 600px; width: 100%; margin: 0 auto; padding: 24px; display: flex; flex-direction: column; gap: 16px; flex: 1; overflow-y: auto;">
          <div style="display: flex; padding: 12px; gap: 8px; border: 1px solid rgba(255,255,255,0.05); border-radius: 8px; background: #141416;">
            <input
              type="text"
              placeholder="New Tally Name..."
              bind:value={newTallyName}
              onkeydown={(e) => { if (e.key === "Enter") addTally(); }}
              style="flex: 1; background: #1c1c1e; border: 1px solid rgba(255,255,255,0.05); border-radius: 6px; padding: 8px 12px; color: white; font-size: 13px; outline: none;"
            />
            <button
              class="btn primary"
              style="padding: 8px 16px; font-size: 13px;"
              onclick={addTally}
            >
              Add Tally
            </button>
          </div>

          <div style="display: flex; flex-direction: column; gap: 10px;">
            {#if tallyCards.length === 0}
              <div style="padding: 48px 24px; text-align: center; color: #52525b; font-size: 13px; font-style: italic;">
                No tallies yet. Add one above!
              </div>
            {:else}
              {#each tallyCards as card (card.id)}
                {@const data = (() => {
                  try {
                    return JSON.parse(card.content);
                  } catch (_) {
                    return { name: card.comment || "Tally", count: parseInt(card.content) || 0 };
                  }
                })()}
                <div style="display: flex; align-items: center; justify-content: space-between; background: #18181b; border: 1px solid rgba(255, 255, 255, 0.05); border-radius: 8px; padding: 12px 18px; gap: 16px;">
                  <span style="font-size: 14px; font-weight: 700; color: #e4e4e7; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; flex: 1; text-align: left;">
                    {data.name}
                  </span>
                  <div style="display: flex; align-items: center; gap: 4px;">
                    <button
                      class="btn-sm"
                      style="padding: 4px 12px; font-size: 16px; color: #f87171; border-color: rgba(239, 68, 68, 0.15);"
                      disabled={data.count <= 0}
                      onclick={() => updateTallyCount(card, data.name, data.count - 1)}
                    >
                      -
                    </button>
                    <span style="font-size: 16px; font-weight: 700; color: white; min-width: 32px; text-align: center;">
                      {data.count}
                    </span>
                    <button
                      class="btn-sm"
                      style="padding: 4px 12px; font-size: 16px; color: #4ade80; border-color: rgba(74, 222, 128, 0.15);"
                      onclick={() => updateTallyCount(card, data.name, data.count + 1)}
                    >
                      +
                    </button>
                    <button
                      class="close-btn"
                      style="font-size: 18px; padding: 4px 8px; margin-left: 12px; color: #71717a;"
                      onclick={() => deleteTally(card.id)}
                      title="Delete Tally"
                    >
                      &times;
                    </button>
                  </div>
                </div>
              {/each}
            {/if}
          </div>
        </div>
      </div>
    {/if}
  </div>
{/if}

<!-- Immersive Sites Sandbox Modal (At very top of stacking context) -->
{#if showSitesModal}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
  <div
    class="modal-backdrop"
    onclick={(e) => { if (e.target === e.currentTarget) closeSitesModal(e); }}
    role="button"
    tabindex="-1"
  >
    <div
      class="modal-container"
      style="width: 750px; max-width: 95%; height: 600px; max-height: 90vh; display: flex; flex-direction: column;"
    >
      <div class="modal-header">
        <div style="display: flex; flex-direction: column; gap: 2px;">
          <h3 style="margin:0;">🌐 HTML Site block Sandbox</h3>
          <span style="font-size: 11px; color: #64748b;"
            >Renders templates seamlessly using sandbox environment</span
          >
        </div>
        <button class="close-btn" onclick={closeSitesModal}>&times;</button>
      </div>

      <div
        class="sites-tabs"
        style="display: flex; gap: 4px; background: #121212; border-bottom: 1px solid #2e2e2e; padding: 6px 12px;"
      >
        <button
          class="emoji-tab-btn"
          class:active={sitesTab === "edit"}
          onclick={() => (sitesTab = "edit")}
        >
          🛠️ Code Editor
        </button>
        <button
          class="emoji-tab-btn"
          class:active={sitesTab === "preview"}
          onclick={() => (sitesTab = "preview")}
        >
          📡 Local Server
        </button>
      </div>

      <div class="modal-body" style="flex:1; overflow-y:auto; padding: 18px;">
        {#if sitesTab === "edit"}
          <div style="display: flex; flex-direction: column; gap: 14px;">
            <div style="display: flex; flex-direction: column; gap: 4px;">
              <label
                style="font-size: 10px; font-weight: 700; color: #818cf8; letter-spacing: 0.5px;"
                >SITE NAME</label
              >
              <input
                type="text"
                bind:value={sitesName}
                oninput={handleSaveSites}
                placeholder="Enter name of site widget"
                style="background:#1e1e1e; border: 1px solid #2e2e2e; border-radius: 6px; padding: 8px 12px; color: white;"
              />
            </div>
            <div style="display: flex; flex-direction: column; gap: 4px;">
              <label
                style="font-size: 10px; font-weight: 700; color: #818cf8; letter-spacing: 0.5px;"
                >SITE SHORT DESCRIPTION</label
              >
              <input
                type="text"
                bind:value={sitesDesc}
                oninput={handleSaveSites}
                placeholder="Renders templates cleanly"
                style="background:#1e1e1e; border: 1px solid #2e2e2e; border-radius: 6px; padding: 8px 12px; color: white;"
              />
            </div>
            <div style="display: flex; flex-direction: column; gap: 4px;">
              <label
                style="font-size: 10px; font-weight: 700; color: #818cf8; letter-spacing: 0.5px;"
                >HTML CODE</label
              >
              <textarea
                bind:value={sitesHtml}
                oninput={handleSaveSites}
                placeholder="Write HTML tags here..."
                style="background:#121212; border: 1px solid #2e2e2e; border-radius: 6px; color: #e2e8f0; font-family: monospace; font-size: 13px; padding: 12px; height: 260px; resize: vertical; line-height: 1.5;"
              ></textarea>
            </div>
          </div>
        {:else}
          <div class="server-workspace-pane">
            <div class="server-status-card" class:active={isSitesLive}>
              <div class="server-status-indicator">
                <span class="pulse-indicator" class:active={isSitesLive}></span>
                <span class="status-text"
                  >{isSitesLive
                    ? "Local Server is Active"
                    : "Local Server is Offline"}</span
                >
              </div>

              {#if isSitesLive}
                <div class="server-address-box">
                  <span class="address-label">Local Port Address</span>
                  <span class="address-value"
                    >{sitesLocalUrl || "Generating Link..."}</span
                  >
                </div>
              {:else}
                <p class="server-offline-hint">
                  Serve your custom HTML locally to preview in real-time or view
                  in external system browsers.
                </p>
              {/if}
            </div>

            <div class="server-actions-container">
              {#if !isSitesLive}
                <button class="btn-serve start" onclick={toggleSitesLive}>
                  <span class="btn-icon">▶</span> Start Local Server
                </button>
              {:else}
                <div class="active-server-buttons">
                  <button
                    class="btn primary"
                    onclick={() => {
                      if (sitesLocalUrl) window.open(sitesLocalUrl, "_blank");
                    }}
                  >
                    Open in Browser
                  </button>
                  <button
                    class="btn secondary"
                    onclick={() => {
                      if (sitesLocalUrl) {
                        navigator.clipboard.writeText(sitesLocalUrl);
                        showAlert("Copied", "Address copied to clipboard!");
                      }
                    }}
                  >
                    Copy Address
                  </button>
                  <button class="btn stop-btn" onclick={toggleSitesLive}>
                    ⏹ Stop Serving
                  </button>
                </div>
              {/if}
            </div>
          </div>
        {/if}
      </div>
    </div>
  </div>
{/if}

<!-- Immersive Sites Sandbox Preview Modal (Consistent with Flutter separate preview) -->
{#if showSitesPreviewModal}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
  <div
    class="modal-backdrop"
    onclick={(e) => { if (e.target === e.currentTarget) { (showSitesPreviewModal = false) } }}
    role="button"
    tabindex="-1"
  >
    <div
      class="modal-container"
      style="width: 800px; max-width: 95%; height: 600px; max-height: 90vh; display: flex; flex-direction: column;"
    >
      <div class="modal-header">
        <div style="display: flex; flex-direction: column; gap: 2px;">
          <h3 style="margin:0;">🎬 Live Sandbox Preview - {previewSitesName}</h3>
          <span style="font-size: 11px; color: #64748b;"
            >Interacting in a secure sandbox webview container</span
          >
        </div>
        <button class="close-btn" onclick={() => (showSitesPreviewModal = false)}>&times;</button>
      </div>
      <div class="modal-body" style="flex:1; overflow:hidden; padding: 12px; background: #09090b;">
        <iframe
          title="Sandbox Preview Webview"
          srcdoc={previewSitesHtml || "<html><body><p style='color: grey; font-family: sans-serif; text-align: center; margin-top: 100px;'>Empty HTML Content</p></body></html>"}
          style="width: 100%; height: 100%; border: none; background: transparent;"
          sandbox="allow-scripts"
        ></iframe>
      </div>
    </div>
  </div>
{/if}

<!-- Link Subpage Modal (Rendered Globally at Root Level) -->
{#if showLinkPageModal && linkingCard}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
  <div
    class="modal-backdrop"
    onclick={(e) => { if (e.target === e.currentTarget) { (showLinkPageModal = false) } }}
    role="button"
    tabindex="-1"
  >
    <div class="modal-container command-palette-modal">
      <div class="modal-header">
        <div class="modal-title-group">
          <h3>Link a Subpage</h3>
          <span class="modal-subtitle"
            >Search or instantly generate nested references</span
          >
        </div>
        <button class="close-btn" onclick={() => (showLinkPageModal = false)}
          >&times;</button
        >
      </div>

      <div class="modal-search">
        <span class="search-icon">🔍</span>
        <input
          type="text"
          bind:value={pageSearchQuery}
          placeholder="Type to filter journal pages..."
          autofocus
        />
        {#if pageSearchQuery}
          <button class="clear-btn" onclick={() => (pageSearchQuery = "")}
            >&times;</button
          >
        {/if}
      </div>

      <!-- Linear/Raycast-style Command Trigger -->
      <button
        class="cmd-action-btn"
        onclick={(e) => { e.stopPropagation(); handleCreateNewPageAndLink(e); }}
      >
        <span class="cmd-plus-icon">+</span>
        <div class="cmd-meta">
          <span class="cmd-title">Create New Page & Link</span>
          <span class="cmd-subtitle"
            >Instantly instantiate and point this card to a new workspace note</span
          >
        </div>
        <span class="cmd-kbd">Enter ↵</span>
      </button>

      <div class="modal-body list-body">
        {#if filteredLinkCandidates.length === 0}
          <div class="empty-results">
            <span>📭</span>
            <p>
              {pageSearchQuery
                ? "No matching pages found"
                : "No other pages available to link"}
            </p>
          </div>
        {:else}
          <div class="candidates-list">
            {#each filteredLinkCandidates as p}
              <button
                class="candidate-row"
                onclick={() => selectPageToLink(p)}
              >
                <span class="cand-emoji">
                  <PageIcon emoji={p.emoji} size={14} />
                </span>
                <span class="cand-title">{p.title || "Untitled"}</span>
                <span class="cand-action-hint">Link Page ➔</span>
              </button>
            {/each}
          </div>
        {/if}
      </div>
    </div>
  </div>
{/if}

<!-- Stateful Parent-Browsing Move Page Modal -->
{#if showMovePageModal && movingPage}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
  <div
    class="modal-backdrop"
    onclick={(e) => { if (e.target === e.currentTarget) { (showMovePageModal = false) } }}
    role="button"
    tabindex="-1"
  >
    <div class="modal-container command-palette-modal" style="height: 380px;">
      <div class="modal-header">
        <div class="modal-title-group">
          <h3 style="margin:0;">Move Page</h3>
          <span class="modal-subtitle">Moving "{movingPage.title || 'Untitled'}"</span>
        </div>
        <button class="close-btn" onclick={() => (showMovePageModal = false)}>&times;</button>
      </div>

      <div class="modal-search" style="background: #121215; justify-content: space-between; display: flex; align-items: center; padding: 12px 16px;">
        <div style="display: flex; align-items: center; gap: 8px;">
          <span class="search-icon">📂</span>
          <span style="font-size: 12px; font-weight: 700; color: #cbd5e1;">
            {moveBrowsingId === null ? "Root Level (No Parent)" : (allPages.find(p => p.id === moveBrowsingId)?.title || "Untitled")}
          </span>
        </div>
        {#if moveBrowsingId !== null}
          <button
            class="btn-sm"
            onclick={() => {
              const current = allPages.find(p => p.id === moveBrowsingId);
              moveBrowsingId = current ? (current.parent_id || current.parentId) : null;
            }}
          >
            ↑ Go Up
          </button>
        {/if}
      </div>

      <div class="modal-body list-body" style="overflow-y: auto; padding: 8px;">
        {#if allPages.filter(p => p.id !== movingPage.id && p.relation_type !== "sidepage" && (p.parent_id || p.parentId) === moveBrowsingId).length === 0}
          <div style="padding: 24px; text-align: center; color: #52525b; font-size: 11px; line-height: 1.5;">
            No subpages under this level.<br/>Click "Move Here" to choose this destination.
          </div>
        {:else}
          <div class="candidates-list">
            {#each allPages.filter(p => p.id !== movingPage.id && p.relation_type !== "sidepage" && (p.parent_id || p.parentId) === moveBrowsingId) as candidate}
              <button
                class="candidate-row"
                style="display: flex; align-items: center; gap: 8px; width: 100%; border: none; background: transparent; padding: 6px 12px; border-radius: 6px;"
                onclick={() => (moveBrowsingId = candidate.id)}
              >
                <span class="cand-emoji"><PageIcon emoji={candidate.emoji} size={14} /></span>
                <span class="cand-title" style="flex:1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 12px; text-align:left;">{candidate.title || "Untitled"}</span>
                <span style="font-size: 10px; color: #52525b;">Open →</span>
              </button>
            {/each}
          </div>
        {/if}
      </div>

      <div class="modal-footer" style="padding: 10px 16px; border-top: 1px solid rgba(255, 255, 255, 0.05); background: #121215; display: flex; justify-content: flex-end; gap: 8px;">
        <button class="btn secondary" onclick={() => (showMovePageModal = false)}>Cancel</button>
        <button
          class="btn primary"
          onclick={() => {
            MovePage(movingPage.id, moveBrowsingId || "")
              .then(() => {
                showMovePageModal = false;
                movingPage = null;
              })
              .catch(err => showAlert("Move Failed", err.toString()));
          }}
        >
          Move Here
        </button>
      </div>
    </div>
  </div>
{/if}

<!-- Stateful Prioritized Move Block Modal -->
{#if showMoveBlockModal && movingCard}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
  <div
    class="modal-backdrop"
    onclick={(e) => { if (e.target === e.currentTarget) { (showMoveBlockModal = false) } }}
    role="button"
    tabindex="-1"
  >
    <div class="modal-container command-palette-modal" style="height: 480px; display: flex; flex-direction: column;">
      <div class="modal-header">
        <div class="modal-title-group">
          <h3 style="margin:0;">Move Block to Page</h3>
          <span class="modal-subtitle">Choose a destination page for this block</span>
        </div>
        <button class="close-btn" onclick={() => (showMoveBlockModal = false)}>&times;</button>
      </div>

      <div class="modal-body list-body" style="overflow-y: auto; flex: 1; padding: 12px 16px;">
        {#if moveBlockSubpages.length > 0}
          <div class="category-header" style="font-size: 10px; font-weight: bold; color: #818cf8; text-transform: uppercase; letter-spacing: 0.5px; margin-bottom: 6px;">Subpages & Links Inside Current Page</div>
          <div class="candidates-list" style="margin-bottom: 16px;">
            {#each moveBlockSubpages as page}
              <button
                class="candidate-row"
                style="display: flex; align-items: center; gap: 8px; width: 100%; border: none; background: transparent; padding: 8px 12px; border-radius: 6px; text-align: left; cursor: pointer;"
                onclick={() => handleMoveBlockToPage(page)}
              >
                <PageIcon emoji={page.emoji} size={14} />
                <span style="font-size: 12px; color: #cbd5e1;">{page.title || "Untitled"}</span>
              </button>
            {/each}
          </div>
        {/if}

        {#if moveBlockNeighbors.length > 0}
          <div class="category-header" style="font-size: 10px; font-weight: bold; color: #71717a; text-transform: uppercase; letter-spacing: 0.5px; margin-bottom: 6px;">Neighbor Pages</div>
          <div class="candidates-list" style="margin-bottom: 16px;">
            {#each moveBlockNeighbors as page}
              <button
                class="candidate-row"
                style="display: flex; align-items: center; gap: 8px; width: 100%; border: none; background: transparent; padding: 8px 12px; border-radius: 6px; text-align: left; cursor: pointer;"
                onclick={() => handleMoveBlockToPage(page)}
              >
                <PageIcon emoji={page.emoji} size={14} />
                <span style="font-size: 12px; color: #cbd5e1;">{page.title || "Untitled"}</span>
              </button>
            {/each}
          </div>
        {/if}

        {#if moveBlockOthers.length > 0}
          <div class="category-header" style="font-size: 10px; font-weight: bold; color: #71717a; text-transform: uppercase; letter-spacing: 0.5px; margin-bottom: 6px;">Other Pages</div>
          <div class="candidates-list">
            {#each moveBlockOthers as page}
              <button
                class="candidate-row"
                style="display: flex; align-items: center; gap: 8px; width: 100%; border: none; background: transparent; padding: 8px 12px; border-radius: 6px; text-align: left; cursor: pointer;"
                onclick={() => handleMoveBlockToPage(page)}
              >
                <PageIcon emoji={page.emoji} size={14} />
                <span style="font-size: 12px; color: #cbd5e1;">{page.title || "Untitled"}</span>
              </button>
            {/each}
          </div>
        {/if}
      </div>

      <div class="modal-footer" style="padding: 10px 16px; border-top: 1px solid rgba(255, 255, 255, 0.05); background: #121215; display: flex; justify-content: flex-end; gap: 8px;">
        <button class="btn secondary" onclick={() => (showMoveBlockModal = false)}>Cancel</button>
      </div>
    </div>
  </div>
{/if}

<!-- Comments Modal -->
{#if showCommentsModal && commentCard}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
  <div
    class="modal-backdrop"
    onclick={(e) => { if (e.target === e.currentTarget) { (showCommentsModal = false) } }}
    role="button"
    tabindex="-1"
  >
    <div class="modal-container comments-modal-container">
      <div class="modal-header">
        <div class="modal-title-group">
          <h3>Comments / Notes</h3>
          <span class="modal-subtitle"
            >Block context: {commentCard.content ? commentCard.content.substring(0, 120) : '(no content)'}</span
          >
        </div>
        <button class="close-btn" onclick={() => (showCommentsModal = false)}>&times;</button>
      </div>
      <div class="comments-modal-body">
        {#if parseMetadata(commentCard.comment).comments.length === 0}
          <div class="comments-empty">No comments yet. Add one below.</div>
        {:else}
          <div class="comments-timeline">
            {#each parseMetadata(commentCard.comment).comments as c, i}
              <div class="comment-item">
                <span class="comment-bullet">•</span>
                <span class="comment-text">{c}</span>
                <button class="comment-delete-btn" onclick={() => handleDeleteComment(i)}>×</button>
              </div>
            {/each}
          </div>
        {/if}
      </div>
      <div class="comments-modal-footer">
        <input
          type="text"
          bind:value={commentInput}
          placeholder="Add a comment or note..."
          onkeydown={(e) => { if (e.key === "Enter") handleAddComment(); }}
        />
        <button class="btn primary" onclick={handleAddComment} disabled={!commentInput.trim()}>Add</button>
      </div>
    </div>
  </div>
{/if}

<style>
  /* Globally hide scrollbars for all scrollable containers while maintaining scroll functionality */
  :global(::-webkit-scrollbar) {
    display: none !important;
    width: 0 !important;
    height: 0 !important;
  }
  :global(*) {
    scrollbar-width: none !important; /* Firefox */
    -ms-overflow-style: none !important; /* IE/Edge */
    box-sizing: border-box;
  }

  :global(body) {
    background-color: #09090b;
    margin: 0;
    padding: 0;
  }

  /* Force all modals and overlays to sit at the absolute top of Wails' rendering layers */
  .modal-backdrop {
    position: fixed;
    top: 0;
    left: 0;
    width: 100vw;
    height: 100vh;
    background: rgba(9, 9, 11, 0.7);
    backdrop-filter: blur(8px);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 99999999 !important;
    cursor: default;
  }

  .fullscreen-editor-overlay {
    position: fixed;
    top: 0;
    left: 0;
    width: 100vw;
    height: 100vh;
    background: #09090b;
    z-index: 100000000 !important;
    display: flex;
    flex-direction: column;
  }

  .app-layout {
    display: flex;
    width: 100vw;
    height: 100vh;
    overflow: hidden;
    background-color: #09090b;
    color: #f4f4f5;
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto,
      Helvetica, Arial, sans-serif;
  }
  .app-layout.dragging {
    cursor: col-resize;
    user-select: none;
  }

  .sidebar {
    display: flex;
    flex-direction: column;
    height: 100%;
    overflow: hidden;
    flex-shrink: 0;
    background: #121215;
  }
  .left-sidebar {
    border-right: 1px solid rgba(255, 255, 255, 0.05);
  }
  .right-sidebar {
    background: #121215;
    border-left: 1px solid rgba(255, 255, 255, 0.05);
  }

  /* Drag handles */
  .drag-handle {
    width: 4px;
    cursor: col-resize;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    background: transparent;
    transition: background 0.15s;
    z-index: 20;
  }
  .drag-handle:hover,
  .drag-handle.active {
    background: rgba(129, 140, 248, 0.1);
  }
  .handle-line {
    width: 1px;
    height: 32px;
    border-radius: 1px;
    background: rgba(255, 255, 255, 0.1);
    transition: background 0.15s;
  }
  .drag-handle:hover .handle-line,
  .drag-handle.active .handle-line {
    background: #818cf8;
  }

  .sidebar-header {
    padding: 16px 20px;
    border-bottom: 1px solid rgba(255, 255, 255, 0.05);
    display: flex;
    justify-content: space-between;
    align-items: center;
    flex-shrink: 0;
  }
  .logo-section {
    display: flex;
    align-items: center;
    gap: 8px;
    font-weight: 700;
  }
  .logo-svg {
    width: 20px;
    height: 20px;
    flex-shrink: 0;
  }
  .logo-welcome-svg {
    width: 28px;
    height: 28px;
    flex-shrink: 0;
  }
  .logo-text {
    font-size: 13px;
    letter-spacing: -0.3px;
    background: linear-gradient(135deg, #a5b4fc, #818cf8);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
  }

  .status-badge {
    display: flex;
    align-items: center;
    gap: 5px;
    padding: 3px 8px;
    border-radius: 12px;
    font-size: 8px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }
  .status-badge.disconnected {
    background: rgba(239, 68, 68, 0.08);
    border: 1px solid rgba(239, 68, 68, 0.15);
    color: #f87171;
  }
  .status-badge.connected {
    background: rgba(34, 197, 94, 0.08);
    border: 1px solid rgba(34, 197, 94, 0.15);
    color: #4ade80;
  }
  .status-badge.connecting,
  .status-badge.reconnecting {
    background: rgba(251, 191, 36, 0.08);
    border: 1px solid rgba(251, 191, 36, 0.15);
    color: #fbbf24;
  }
  .dot {
    width: 4px;
    height: 4px;
    border-radius: 50%;
  }
  .disconnected .dot {
    background: #f87171;
  }
  .connected .dot {
    background: #4ade80;
  }
  .connecting .dot,
  .reconnecting .dot {
    background: #fbbf24;
    animation: pulse 1.5s infinite;
  }
  @keyframes pulse {
    0%,
    100% {
      opacity: 1;
    }
    50% {
      opacity: 0.3;
    }
  }

  .sidebar-section {
    padding: 16px 20px;
    border-bottom: 1px solid rgba(255, 255, 255, 0.05);
  }
  .sidebar-section.grow {
    flex: 1;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
  }

  .section-label {
    font-size: 9px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 1px;
    color: #71717a;
    margin-bottom: 10px;
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  /* Read-Only Active Workspace Display */
  .active-workspace-card {
    background: transparent;
    padding: 12px 20px;
  }
  .active-ws-display {
    display: flex;
    align-items: center;
    gap: 8px;
    background: rgba(255, 255, 255, 0.02);
    border: 1px solid rgba(255, 255, 255, 0.05);
    border-radius: 6px;
    padding: 6px 12px;
    color: #cbd5e1;
    font-size: 11px;
    font-weight: 600;
  }
  .ws-icon {
    font-size: 12px;
  }
  .ws-name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .devices-box {
    background: rgba(255, 255, 255, 0.01);
    border: 1px solid rgba(255, 255, 255, 0.04);
    border-radius: 8px;
    padding: 12px;
  }
  .connected-info {
    display: flex;
    justify-content: space-between;
    font-size: 11px;
    margin-bottom: 8px;
  }
  .conn-label {
    color: #71717a;
  }
  .conn-name {
    color: #4ade80;
    font-weight: 700;
  }
  .no-dev {
    font-size: 11px;
    color: #52525b;
    text-align: center;
    padding: 6px 0;
  }
  .dev-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 8px;
  }
  .dev-row:last-child {
    margin-bottom: 0;
  }
  .dev-name {
    font-size: 11px;
    font-weight: 600;
    display: block;
    color: #e4e4e7;
  }
  .dev-ip {
    font-size: 9px;
    color: #52525b;
    font-family: monospace;
  }

  .tree-scroll {
    flex: 1;
    overflow-y: auto;
    margin-left: -4px;
  }
  .tree-empty {
    font-size: 11px;
    color: #52525b;
    padding: 8px 4px;
  }

  /* Root Page Search Input */
  .sidebar-search-container {
    margin-bottom: 12px;
  }
  .sidebar-search-input {
    width: 100%;
    background: rgba(255,255,255,0.02);
    border: 1px solid rgba(255,255,255,0.05);
    border-radius: 6px;
    padding: 6px 10px;
    color: white;
    font-size: 11px;
    outline: none;
    box-sizing: border-box;
  }
  .sidebar-search-input:focus {
    border-color: rgba(129,140,248,0.4);
  }

  /* Sidebar List Item Styles */
  .tree-node {
    display: flex;
    flex-direction: column;
    width: 100%;
  }
  .node-row {
    display: flex;
    align-items: center;
    padding: 4px 8px;
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
    display: flex;
    align-items: center;
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

  .icon-btn {
    background: transparent;
    border: none;
    color: #71717a;
    cursor: pointer;
    font-size: 14px;
    padding: 0 4px;
    transition: color 0.15s;
  }
  .icon-btn:hover {
    color: #f4f4f5;
  }

  /* Manual Connect Form UI styling */
  .manual-connect-card {
    background: rgba(255, 255, 255, 0.01);
    border-radius: 8px;
    border: 1px solid rgba(255, 255, 255, 0.04);
    margin: 12px 20px;
    padding: 12px;
  }
  .manual-form {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .form-group {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .form-group label {
    font-size: 8px;
    font-weight: 700;
    color: #52525b;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }
  .form-group input {
    background: rgba(255, 255, 255, 0.02) !important;
    border: 1px solid rgba(255, 255, 255, 0.05) !important;
    border-radius: 6px !important;
    color: #f4f4f5 !important;
    font-size: 11px !important;
    padding: 6px 10px !important;
    outline: none !important;
    margin-bottom: 0 !important;
    transition: all 0.2s ease;
  }
  .form-group input:focus {
    border-color: rgba(129, 140, 248, 0.4) !important;
    background: rgba(255, 255, 255, 0.03) !important;
    box-shadow: 0 0 0 1px rgba(129, 140, 248, 0.1) !important;
  }
  .pin-input {
    letter-spacing: 4px !important;
    text-align: center !important;
    font-weight: 700 !important;
    font-size: 12px !important;
  }
  .connect-btn {
    width: 100%;
    background: #818cf8;
    color: white;
    border: none;
    border-radius: 6px;
    padding: 8px;
    font-size: 11px;
    font-weight: 600;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    transition:
      background 0.15s,
      transform 0.1s;
  }
  .connect-btn:hover {
    background: #6366f1;
  }
  .connect-btn:active {
    transform: translateY(0.5px);
  }
  .connection-error-box {
    display: flex;
    align-items: center;
    gap: 6px;
    background: rgba(239, 68, 68, 0.05);
    border: 1px solid rgba(239, 68, 68, 0.1);
    border-radius: 6px;
    padding: 6px 10px;
  }
  .err-icon {
    font-size: 11px;
  }
  .err-msg {
    color: #f87171;
    font-size: 10px;
    font-weight: 500;
    line-height: 1.3;
  }

  /* Buttons */
  .btn-sm {
    border-radius: 6px;
    font-size: 10px;
    font-weight: 600;
    padding: 4px 10px;
    cursor: pointer;
    border: 1px solid rgba(255, 255, 255, 0.05);
    background: rgba(255, 255, 255, 0.02);
    color: #cbd5e1;
    transition: all 0.15s;
  }
  .btn-sm:hover {
    background: rgba(255, 255, 255, 0.06);
    color: #f4f4f5;
  }
  .btn-sm.primary {
    background: #818cf8;
    border-color: rgba(129, 140, 248, 0.2);
    color: white;
  }
  .btn-sm.primary:hover {
    background: #6366f1;
  }
  .btn-sm.danger {
    background: rgba(239, 68, 68, 0.05);
    border-color: rgba(239, 68, 68, 0.1);
    color: #f87171;
  }
  .btn-sm.danger:hover {
    background: rgba(239, 68, 68, 0.1);
  }
  .sidebar-toggle {
    background: #818cf8 !important;
    border-color: rgba(129, 140, 248, 0.3) !important;
    color: white !important;
    margin-left: 4px;
  }
  .sidebar-toggle:hover {
    background: #6366f1 !important;
  }
  .btn-sm.full {
    width: 100%;
  }

  .btn {
    border-radius: 6px;
    font-size: 11px;
    font-weight: 600;
    padding: 8px 16px;
    cursor: pointer;
    border: none;
    transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
  }
  .btn.primary {
    background: #818cf8;
    color: #ffffff;
    border: 1px solid rgba(129, 140, 248, 0.2);
    box-shadow: 0 2px 8px rgba(129, 140, 248, 0.15);
  }
  .btn.primary:hover {
    background: #6366f1;
    box-shadow: 0 4px 12px rgba(99, 102, 241, 0.25);
    transform: translateY(-0.5px);
  }
  .btn.secondary {
    background: rgba(255, 255, 255, 0.02);
    border: 1px solid rgba(255, 255, 255, 0.05);
    color: #cbd5e1;
  }
  .btn.secondary:hover {
    background: rgba(255, 255, 255, 0.06);
    border-color: rgba(255, 255, 255, 0.1);
    color: #ffffff;
  }
  .btn.stop-btn {
    background: rgba(239, 68, 68, 0.02);
    border: 1px solid rgba(239, 68, 68, 0.15);
    color: #f87171;
  }
  .btn.stop-btn:hover {
    background: rgba(239, 68, 68, 0.08);
    border-color: rgba(239, 68, 68, 0.3);
  }

  /* Editor Workspace */
  .editor-workspace {
    flex: 1;
    display: flex;
    flex-direction: column;
    height: 100%;
    overflow: hidden;
    background: #09090b;
  }

  .welcome {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 40px;
    background: #09090b;
  }
  .welcome-card {
    max-width: 460px;
    text-align: center;
    background: #121215;
    border: 1px solid rgba(255, 255, 255, 0.05);
    border-radius: 16px;
    padding: 40px;
    box-shadow: 0 10px 30px rgba(0, 0, 0, 0.3);
  }
  .pulse {
    width: 56px;
    height: 56px;
    border-radius: 50%;
    background: rgba(129, 140, 248, 0.04);
    border: 1px solid rgba(129, 140, 248, 0.15);
    display: flex;
    align-items: center;
    justify-content: center;
    margin: 0 auto 20px;
    font-size: 24px;
  }
  .welcome-card h1 {
    font-size: 20px;
    font-weight: 800;
    margin: 0 0 8px;
    letter-spacing: -0.3px;
  }
  .welcome-card p {
    font-size: 12px;
    color: #a1a1aa;
    margin: 0 0 28px;
    line-height: 1.5;
  }
  .steps {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 12px;
    text-align: left;
  }
  .step {
    background: rgba(255, 255, 255, 0.01);
    border: 1px solid rgba(255, 255, 255, 0.03);
    border-radius: 10px;
    padding: 12px;
  }
  .step .num {
    width: 18px;
    height: 18px;
    border-radius: 50%;
    background: #818cf8;
    color: white;
    font-weight: 700;
    font-size: 9px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    margin-bottom: 8px;
  }
  .step p {
    font-size: 10px;
    color: #cbd5e1;
    margin: 0;
    line-height: 1.4;
  }

  .empty-page {
    text-align: center;
  }
  .big-icon {
    font-size: 40px;
    display: block;
    margin-bottom: 16px;
  }
  .empty-page h2 {
    font-size: 16px;
    margin: 0 0 6px;
    font-weight: 700;
  }
  .empty-page p {
    font-size: 12px;
    color: #71717a;
    margin: 0 0 20px;
  }

  .editor-header {
    background: #09090b;
    border-bottom: 1px solid rgba(255, 255, 255, 0.05);
    padding: 12px 28px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    flex-shrink: 0;
  }
  .breadcrumbs {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 11px;
  }
  .back-btn {
    background: transparent;
    border: none;
    color: #71717a;
    cursor: pointer;
    font-size: 10px;
    font-weight: 600;
    padding: 3px 6px;
    border-radius: 4px;
    transition: all 0.15s;
  }
  .back-btn:hover {
    background: rgba(255, 255, 255, 0.05);
    color: #f4f4f5;
  }
  .sep {
    color: rgba(255, 255, 255, 0.1);
  }
  .bc-root {
    color: #71717a;
  }
  .bc-current {
    color: #818cf8;
    font-weight: 600;
  }
  .header-actions {
    display: flex;
    gap: 8px;
  }

  .title-bar {
    display: flex;
    align-items: center;
    gap: 20px;
    padding: 24px 28px 12px;
    flex-shrink: 0;
  }
  .emoji-input-btn {
    width: 64px;
    height: 64px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 32px;
    background: rgba(255, 255, 255, 0.02);
    border: 2px solid rgba(255, 255, 255, 0.1);
    border-radius: 12px;
    color: #e2e8f0;
    cursor: pointer;
    flex-shrink: 0;
    transition: all 0.15s;
    overflow: hidden;
    padding: 0;
  }
  .emoji-input-btn:hover {
    border-color: rgba(129, 140, 248, 0.5);
    background: rgba(129, 140, 248, 0.06);
  }
  .title-input {
    flex: 1;
    background: transparent;
    border: none;
    outline: none;
    font-size: 20px;
    font-weight: 700;
    color: #f4f4f5;
    padding: 4px 0;
    border-bottom: 2px solid transparent;
    letter-spacing: -0.3px;
  }
  .title-input:focus {
    border-bottom-color: rgba(129, 140, 248, 0.3);
  }
  .title-input::placeholder {
    color: #3f3f46;
  }

  .card-column {
    flex: 1;
    overflow-y: auto;
    padding: 16px 28px 60px;
    display: flex;
    flex-direction: column;
    gap: 12px;
    align-items: stretch; /* Stretch cards to full width */
    text-align: left; /* Guarantee default left-alignment for all children */
  }

  .empty-cards {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 48px 32px;
    text-align: center;
    background: rgba(255, 255, 255, 0.01);
    border: 1px dashed rgba(255, 255, 255, 0.05);
    border-radius: 12px;
    margin: 8px 0;
  }
  .empty-cards .empty-icon {
    margin-bottom: 12px;
    opacity: 0.8;
  }
  .empty-cards h3 {
    font-size: 14px;
    font-weight: 700;
    margin: 0 0 6px;
    color: #e4e4e7;
  }
  .empty-cards p {
    font-size: 12px;
    color: #71717a;
    margin: 0 0 20px;
  }
  .empty-actions {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    justify-content: center;
  }

  .add-card-btn {
    background: transparent;
    border: 1px dashed rgba(255, 255, 255, 0.06);
    border-radius: 8px;
    padding: 10px;
    color: #52525b;
    cursor: pointer;
    font-size: 11px;
    font-weight: 600;
    transition: all 0.15s;
    margin-top: 10px;
  }
  .add-card-btn:hover {
    border-color: rgba(129, 140, 248, 0.25);
    color: #818cf8;
    background: rgba(129, 140, 248, 0.01);
  }

  .scroll-to-bottom-btn {
    position: sticky;
    bottom: 16px;
    align-self: flex-end;
    width: 36px;
    height: 36px;
    border-radius: 50%;
    background: #818cf8;
    color: white;
    border: none;
    font-size: 18px;
    font-weight: 700;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    box-shadow: 0 4px 12px rgba(129, 140, 248, 0.3);
    transition: all 0.15s;
    z-index: 10;
    margin-top: -56px;
    flex-shrink: 0;
  }
  .scroll-to-bottom-btn:hover {
    background: #6366f1;
    transform: translateY(-1px);
    box-shadow: 0 6px 16px rgba(129, 140, 248, 0.4);
  }

  .card-slot {
    transition: opacity 0.15s;
    width: 100%; /* Force full container width */
    text-align: left; /* Re-enforce left-alignment */
  }
  .card-slot[draggable="true"] {
    cursor: grab;
  }
  .card-slot[draggable="true"]:active {
    cursor: grabbing;
  }

  .insert-slot {
    height: 4px;
    margin: 0 16px;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.2s ease;
  }
  .insert-slot:hover {
    height: 28px;
    background: rgba(129, 140, 248, 0.02);
    border-radius: 4px;
  }
  .insert-btn {
    opacity: 0;
    background: #18181b;
    border: 1px solid rgba(255, 255, 255, 0.08);
    color: #71717a;
    font-size: 11px;
    width: 18px;
    height: 18px;
    border-radius: 4px;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.15s;
  }
  .insert-slot:hover .insert-btn {
    opacity: 1;
  }
  .insert-btn:hover {
    border-color: #818cf8;
    color: #818cf8;
  }

  /* Block Selector Modal Styles */
  .block-selector-modal {
    width: 400px;
    max-width: 90%;
  }
  .block-options-grid {
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding: 16px !important;
  }
  .block-option-row {
    display: flex;
    align-items: center;
    gap: 12px;
    background: rgba(255, 255, 255, 0.01);
    border: 1px solid rgba(255, 255, 255, 0.04);
    border-radius: 8px;
    padding: 10px 14px;
    text-align: left;
    cursor: pointer;
    color: #f4f4f5;
    transition: all 0.15s ease;
    width: 100%;
  }
  .block-option-row:hover {
    background: rgba(129, 140, 248, 0.04);
    border-color: rgba(129, 140, 248, 0.3);
    transform: translateY(-0.5px);
  }
  .block-icon {
    flex-shrink: 0;
    font-size: 18px;
    line-height: 1;
  }
  .block-desc {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .block-title {
    font-size: 12px;
    font-weight: 700;
    color: #f4f4f5;
  }
  .block-subtitle {
    font-size: 10px;
    color: #71717a;
    line-height: 1.3;
  }

  /* Emoji Picker Modal Styles */
  .emoji-picker-modal {
    width: 440px;
    max-width: 95%;
    height: 420px;
    max-height: 80vh;
  }
  .emoji-picker-tabs {
    display: flex;
    gap: 4px;
    background: #09090b;
    border-bottom: 1px solid rgba(255, 255, 255, 0.05);
    padding: 6px 12px;
    overflow-x: auto;
  }
  .emoji-tab-btn {
    background: transparent;
    border: none;
    color: #71717a;
    font-size: 11px;
    font-weight: 600;
    padding: 6px 10px;
    border-radius: 4px;
    cursor: pointer;
    white-space: nowrap;
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
  .emoji-picker-body {
    padding: 12px !important;
    overflow-y: auto !important;
  }
  .emoji-grid {
    display: grid;
    grid-template-columns: repeat(8, 1fr);
    gap: 4px;
  }
  .emoji-select-btn {
    background: transparent;
    border: none;
    font-size: 20px;
    aspect-ratio: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 6px;
    cursor: pointer;
    transition: background 0.12s;
  }
  .emoji-select-btn:hover {
    background: rgba(255, 255, 255, 0.04);
  }

  .custom-icon-section {
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 24px 16px;
    gap: 12px;
    text-align: center;
  }
  .custom-icon-header {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 6px;
    margin-bottom: 8px;
  }
  .custom-icon-icon {
    font-size: 36px;
    opacity: 0.8;
  }
  .custom-icon-header h4 {
    margin: 0;
    font-size: 14px;
    font-weight: 700;
    color: #f4f4f5;
  }
  .custom-icon-header p {
    margin: 0;
    font-size: 11px;
    color: #71717a;
    line-height: 1.4;
  }
  .custom-icon-url-row {
    display: flex;
    gap: 6px;
    width: 100%;
  }
  .custom-icon-url-row input {
    flex: 1;
    background: rgba(255, 255, 255, 0.02);
    border: 1px solid rgba(255, 255, 255, 0.05);
    border-radius: 6px;
    color: #f4f4f5;
    font-size: 12px;
    padding: 8px 10px;
    outline: none;
  }
  .custom-icon-url-row input:focus {
    border-color: rgba(129, 140, 248, 0.4);
  }

  /* PIN Modal Specific Styles */
  .pin-entry-modal {
    width: 340px;
    max-width: 90%;
  }
  .pin-modal-body {
    padding: 20px !important;
    display: flex;
    flex-direction: column;
    gap: 16px;
  }
  .device-info-bar {
    display: flex;
    align-items: center;
    gap: 10px;
    background: #09090b;
    border: 1px solid rgba(255, 255, 255, 0.05);
    border-radius: 8px;
    padding: 8px 12px;
  }
  .device-icon {
    font-size: 20px;
  }
  .device-details {
    display: flex;
    flex-direction: column;
  }
  .pin-instructions {
    font-size: 11px;
    color: #a1a1aa;
    line-height: 1.5;
    margin: 0;
  }
  .pin-input-container {
    display: flex;
    justify-content: center;
    margin: 4px 0;
  }
  .pin-display-input {
    background: #09090b !important;
    border: 1px solid rgba(255, 255, 255, 0.08) !important;
    border-radius: 8px !important;
    color: #818cf8 !important;
    font-size: 24px !important;
    font-weight: 700 !important;
    letter-spacing: 8px !important;
    text-align: center !important;
    width: 140px !important;
    padding: 6px 0 6px 8px !important;
    outline: none !important;
    font-family: monospace;
    transition: all 0.2s ease;
    margin-bottom: 0 !important;
  }
  .pin-display-input:focus {
    border-color: rgba(129, 140, 248, 0.5) !important;
    box-shadow: 0 0 0 1px rgba(129, 140, 248, 0.2) !important;
  }
  .modal-error-box {
    display: flex;
    align-items: center;
    gap: 8px;
    background: rgba(239, 68, 68, 0.05);
    border: 1px solid rgba(239, 68, 68, 0.1);
    border-radius: 8px;
    padding: 8px 12px;
  }
  .modal-footer {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    padding: 12px 16px;
    border-top: 1px solid rgba(255, 255, 255, 0.05);
    background: #121215;
  }
  .spinner {
    display: inline-block;
    width: 10px;
    height: 10px;
    border: 2px solid rgba(255, 255, 255, 0.3);
    border-radius: 50%;
    border-top-color: white;
    animation: spin 0.8s linear infinite;
    margin-right: 6px;
  }
  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }

  .modal-container {
    background: #1a1a1d;
    border: 1px solid rgba(255, 255, 255, 0.05);
    border-radius: 12px;
    width: 380px;
    max-width: 90%;
    max-height: 400px;
    display: flex;
    flex-direction: column;
    box-shadow: 0 12px 32px rgba(0, 0, 0, 0.4);
    overflow: hidden;
    animation: modal-fade-in 0.15s cubic-bezier(0.16, 1, 0.3, 1);
  }
  @keyframes modal-fade-in {
    from {
      opacity: 0;
      transform: scale(0.97) translateY(8px);
    }
    to {
      opacity: 1;
      transform: scale(1) translateY(0);
    }
  }
  .modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 12px 16px;
    border-bottom: 1px solid rgba(255, 255, 255, 0.05);
  }
  .modal-header h3 {
    margin: 0;
    font-size: 13px;
    font-weight: 700;
    color: #f4f4f5;
  }
  .close-btn {
    background: transparent;
    border: none;
    color: #71717a;
    font-size: 18px;
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
    background: rgba(239, 68, 68, 0.08);
  }
  .modal-search {
    padding: 10px 16px;
    display: flex;
    align-items: center;
    gap: 8px;
    background: #09090b;
    border-bottom: 1px solid rgba(255, 255, 255, 0.05);
    position: relative;
  }
  .search-icon {
    font-size: 12px;
    color: #52525b;
  }
  .modal-search input {
    flex: 1;
    background: transparent;
    border: none;
    color: #f4f4f5;
    font-size: 12px;
    outline: none;
    padding: 2px 0;
  }
  .clear-btn {
    background: transparent;
    border: none;
    color: #52525b;
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
    gap: 2px;
  }
  .candidate-row {
    display: flex;
    align-items: center;
    gap: 8px;
    background: transparent;
    border: none;
    text-align: left;
    color: #cbd5e1;
    padding: 6px 10px;
    cursor: pointer;
    border-radius: 6px;
    width: 100%;
    transition: all 0.12s;
  }
  .candidate-row:hover {
    background: rgba(129, 140, 248, 0.06);
    color: #818cf8;
  }
  .cand-emoji {
    font-size: 14px;
    flex-shrink: 0;
  }
  .cand-title {
    font-size: 12px;
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
    padding: 24px 16px;
    color: #52525b;
  }
  .empty-results span {
    font-size: 20px;
    margin-bottom: 6px;
  }

  .fullscreen-workspace {
    flex: 1;
    display: flex;
    flex-direction: row;
    overflow: hidden;
  }
  .fullscreen-textarea {
    background: #09090b;
    border: none;
    border-right: 1px solid rgba(255, 255, 255, 0.05);
    color: #f4f4f5;
    font-size: 13px;
    font-family: inherit;
    line-height: 1.6;
    padding: 24px;
    resize: none;
    outline: none;
    text-align: left !important;
  }
  .fullscreen-textarea.monospace {
    font-family: "Fira Code", monospace;
    font-size: 12px;
  }
  .fullscreen-preview {
    background: #09090b !important;
    padding: 32px !important;
    overflow-y: auto;
    text-align: left !important; /* Enforces left alignment on container */
  }

  .fullscreen-preview.markdown-rendered {
    text-align: left !important;
  }

  .fullscreen-preview.markdown-rendered :global(p),
  .fullscreen-preview.markdown-rendered :global(li),
  .fullscreen-preview.markdown-rendered :global(h1),
  .fullscreen-preview.markdown-rendered :global(h2),
  .fullscreen-preview.markdown-rendered :global(h3),
  .fullscreen-preview.markdown-rendered :global(h4),
  .fullscreen-preview.markdown-rendered :global(h5),
  .fullscreen-preview.markdown-rendered :global(h6),
  .fullscreen-preview.markdown-rendered :global(blockquote),
  .fullscreen-preview.markdown-rendered :global(span) {
    text-align: left !important; /* Ensures raw blocks do not justify alignment */
  }

  .server-workspace-pane {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 28px 16px;
    gap: 16px;
  }
  .server-status-card {
    background: rgba(255, 255, 255, 0.01);
    border: 1px solid rgba(255, 255, 255, 0.04);
    border-radius: 12px;
    padding: 20px;
    width: 100%;
    max-width: 440px;
    text-align: center;
    transition: all 0.3s ease;
  }
  .server-status-card.active {
    background: rgba(16, 185, 129, 0.01);
    border-color: rgba(16, 185, 129, 0.1);
  }
  .server-status-indicator {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    margin-bottom: 12px;
  }
  .pulse-indicator {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: #ef4444;
    transition: all 0.3s ease;
  }
  .pulse-indicator.active {
    background: #10b981;
    box-shadow: 0 0 8px rgba(16, 185, 129, 0.5);
    animation: pulse-active 2s infinite;
  }
  .status-text {
    font-size: 12px;
    font-weight: 600;
    color: #71717a;
  }
  .server-status-card.active .status-text {
    color: #cbd5e1;
  }
  .server-address-box {
    display: flex;
    flex-direction: column;
    gap: 4px;
    background: #09090b;
    border: 1px solid rgba(255, 255, 255, 0.04);
    padding: 10px;
    border-radius: 6px;
  }
  .address-label {
    font-size: 8px;
    font-weight: 700;
    color: #52525b;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }
  .address-value {
    font-size: 12px;
    color: #818cf8;
    font-family: monospace;
    font-weight: 600;
    word-break: break-all;
  }
  .server-offline-hint {
    font-size: 11px;
    color: #52525b;
    line-height: 1.5;
    margin: 0;
  }
  .server-actions-container {
    display: flex;
    justify-content: center;
    width: 100%;
  }
  .active-server-buttons {
    display: flex;
    gap: 8px;
    width: 100%;
    max-width: 440px;
  }
  .active-server-buttons .btn {
    flex: 1;
  }
  .btn-serve.start {
    background: rgba(16, 185, 129, 0.03);
    border: 1px solid rgba(16, 185, 129, 0.15);
    color: #34d399;
    font-weight: 600;
    font-size: 11px;
    padding: 10px 20px;
    border-radius: 8px;
    cursor: pointer;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    transition: all 0.2s ease;
  }
  .btn-serve.start:hover {
    background: rgba(16, 185, 129, 0.08);
    border-color: rgba(16, 185, 129, 0.3);
    transform: translateY(-0.5px);
  }

  @keyframes pulse-active {
    0% {
      box-shadow: 0 0 6px rgba(16, 185, 129, 0.3);
    }
    50% {
      box-shadow: 0 0 12px rgba(16, 185, 129, 0.5);
    }
    100% {
      box-shadow: 0 0 6px rgba(16, 185, 129, 0.3);
    }
  }

  /* Immersive Editors CSS Layout */
  .markdown-fullscreen-theme,
  .code-fullscreen-theme {
    background-color: #09090b !important;
  }

  .fullscreen-header {
    height: 56px;
    padding: 0 20px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    background: #121215;
    border-bottom: 1px solid rgba(255, 255, 255, 0.05);
    box-sizing: border-box;
  }

  .header-info {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .header-title-row {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .header-badge {
    font-size: 8px;
    font-weight: 800;
    padding: 2px 6px;
    border-radius: 4px;
    background: rgba(129, 140, 248, 0.08);
    border: 1px solid rgba(129, 140, 248, 0.15);
    color: #818cf8;
    letter-spacing: 0.5px;
  }

  .header-badge.code {
    background: rgba(244, 114, 182, 0.08);
    border-color: rgba(244, 114, 182, 0.15);
    color: #f472b6;
  }

  .fullscreen-header h3 {
    margin: 0;
    font-size: 13px;
    font-weight: 700;
    color: #ffffff;
  }

  .fullscreen-header .sub-desc {
    font-size: 10px;
    color: #71717a;
  }

  .editor-stats {
    display: flex;
    gap: 6px;
  }

  .stat-pill {
    background: rgba(255, 255, 255, 0.02);
    border: 1px solid rgba(255, 255, 255, 0.05);
    padding: 4px 10px;
    border-radius: 20px;
    font-size: 10px;
    display: flex;
    gap: 4px;
    align-items: center;
  }

  .stat-val {
    font-weight: 700;
    color: #e4e4e7;
  }

  .stat-lbl {
    color: #52525b;
    font-size: 9px;
  }

  .header-buttons {
    display: flex;
    gap: 8px;
  }

  /* Markdown Editor Split Panel Canvas */
  .editor-pane-container {
    position: relative;
    display: flex;
    flex-direction: column;
    height: 100%;
    background: #09090b;
    border-right: 1px solid rgba(255, 255, 255, 0.05);
  }

  .editor-pane-container textarea {
    flex: 1;
    background: transparent;
    border: none;
    resize: none;
    outline: none;
    color: #e4e4e7;
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
    font-size: 13px;
    line-height: 1.6;
    padding: 24px 24px 80px 24px;
    box-sizing: border-box;
    text-align: left !important;
  }

  /* Floating Styling toolbar */
  .floating-markdown-toolbar {
    position: absolute;
    bottom: 20px;
    left: 50%;
    transform: translateX(-50%);
    background: rgba(18, 18, 21, 0.8);
    backdrop-filter: blur(8px);
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: 30px;
    padding: 4px;
    display: flex;
    gap: 2px;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.5);
    z-index: 10;
  }

  .tool-btn {
    background: transparent;
    border: none;
    color: #a1a1aa;
    width: 28px;
    height: 28px;
    border-radius: 50%;
    cursor: pointer;
    font-size: 11px;
    font-weight: 600;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.15s;
  }

  .tool-btn:hover {
    background: rgba(255, 255, 255, 0.06);
    color: #ffffff;
  }

  /* Split preview screen */
  .fullscreen-preview {
    background: #09090b !important;
    padding: 24px !important;
  }

  /* Interactive Code Highlighting Canvas Viewport */
  .interactive-code-editor-viewport {
    display: flex;
    background: #09090b;
    width: 100%;
    height: 100%;
    overflow: hidden;
    position: relative;
    box-sizing: border-box;
  }

  .editor-line-numbers {
    width: 48px;
    background: rgba(255, 255, 255, 0.01);
    border-right: 1px solid rgba(255, 255, 255, 0.04);
    color: #3f3f46;
    font-family: "Fira Code", "Cascadia Code", Consolas, monospace;
    font-size: 11px;
    line-height: 20px;
    text-align: right;
    padding: 24px 12px 24px 0;
    overflow-y: hidden;
    user-select: none;
    box-sizing: border-box;
  }

  .line-number-row {
    height: 20px;
  }

  .editor-canvas {
    flex: 1;
    height: 100%;
    position: relative;
    overflow: hidden;
    background: transparent;
  }

  .editor-textarea,
  .editor-pre {
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    padding: 24px;
    font-family: "Fira Code", "Cascadia Code", Consolas, monospace;
    font-size: 11px;
    line-height: 20px;
    margin: 0;
    border: 0;
    box-sizing: border-box;
    tab-size: 4;
    -moz-tab-size: 4;
    white-space: pre-wrap;
    word-wrap: break-word;
    text-align: left !important;
  }

  .editor-textarea {
    background: transparent;
    color: transparent !important;
    text-fill-color: transparent !important;
    -webkit-text-fill-color: transparent !important;
    caret-color: #818cf8;
    resize: none;
    outline: none;
    z-index: 2;
    overflow-y: auto;
    overflow-x: hidden;
  }

  .editor-pre {
    background: transparent;
    color: #a1a1aa;
    z-index: 1;
    pointer-events: none;
    overflow: hidden;
  }

  /* Dropdown language component */
  .custom-select-wrapper {
    position: relative;
  }

  .lang-dropdown-select {
    background: rgba(255, 255, 255, 0.02);
    color: #818cf8;
    border: 1px solid rgba(255, 255, 255, 0.05);
    padding: 5px 10px;
    border-radius: 6px;
    font-size: 10px;
    font-weight: 700;
    outline: none;
    cursor: pointer;
    transition: border-color 0.15s;
    appearance: none;
    -webkit-appearance: none;
    padding-right: 24px;
    letter-spacing: 0.5px;
  }

  .lang-dropdown-select:hover {
    border-color: rgba(129, 140, 248, 0.3);
  }

  .custom-select-wrapper::after {
    content: "▼";
    font-size: 6px;
    color: #818cf8;
    position: absolute;
    right: 8px;
    top: 50%;
    transform: translateY(-50%);
    pointer-events: none;
  }

  /* Token Highlight Styles */
  :global(.tok-keyword) {
    color: #f472b6 !important;
    font-weight: bold;
  }

  :global(.tok-string) {
    color: #34d399 !important;
  }

  :global(.tok-comment) {
    color: #52525b !important;
    font-style: italic;
  }

  :global(.tok-number) {
    color: #fb923c !important;
  }

  :global(.tok-builtin) {
    color: #38bdf8 !important;
  }

  :global(.tok-function) {
    color: #60a5fa !important;
  }

  /* Redesigned Sleek Command Palette Modal */
  .command-palette-modal {
    background: #18181b !important;
    border: 1px solid rgba(255, 255, 255, 0.06) !important;
    width: 440px !important;
    max-width: 95% !important;
    max-height: 480px !important;
    box-shadow: 0 16px 36px rgba(0, 0, 0, 0.6) !important;
    display: flex;
    flex-direction: column;
  }

  .modal-title-group {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .modal-subtitle {
    font-size: 10px;
    color: #71717a;
    font-weight: 500;
  }

  .modal-search {
    background: #09090b !important;
    border-bottom: 1px solid rgba(255, 255, 255, 0.05) !important;
    padding: 12px 16px !important;
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .modal-search input {
    background: transparent !important;
    border: none !important;
    outline: none !important;
    color: #f4f4f5 !important;
    font-size: 12px !important;
    width: 100% !important;
    padding: 0 !important;
  }

  /* Linear-style Action Row Button */
  .cmd-action-btn {
    display: flex;
    align-items: center;
    gap: 12px;
    background: rgba(129, 140, 248, 0.02);
    border: none;
    border-bottom: 1px solid rgba(255, 255, 255, 0.04);
    padding: 12px 16px;
    text-align: left;
    cursor: pointer;
    width: 100%;
    transition: all 0.15s ease;
  }

  .cmd-action-btn:hover {
    background: rgba(129, 140, 248, 0.08);
  }

  .cmd-plus-icon {
    font-size: 16px;
    font-weight: 700;
    color: #818cf8;
    background: rgba(129, 140, 248, 0.1);
    width: 24px;
    height: 24px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 6px;
    border: 1px solid rgba(129, 140, 248, 0.2);
    flex-shrink: 0;
  }

  .cmd-meta {
    display: flex;
    flex-direction: column;
    gap: 2px;
    flex: 1;
    min-width: 0;
  }

  .cmd-title {
    font-size: 12px;
    font-weight: 700;
    color: #818cf8;
  }

  .cmd-subtitle {
    font-size: 10px;
    color: #71717a;
    line-height: 1.3;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .cmd-kbd {
    font-size: 9px;
    font-weight: 600;
    background: rgba(255, 255, 255, 0.03);
    border: 1px solid rgba(255, 255, 255, 0.05);
    color: #52525b;
    padding: 2px 6px;
    border-radius: 4px;
    flex-shrink: 0;
  }

  .cmd-action-btn:hover .cmd-kbd {
    color: #818cf8;
    border-color: rgba(129, 140, 248, 0.2);
  }

  .list-body {
    padding: 6px !important;
  }

  /* Interactive Row Hint Styling */
  .candidate-row {
    position: relative;
    padding: 8px 12px !important;
    border-radius: 6px !important;
    display: flex;
    align-items: center;
    gap: 10px;
    background: transparent;
    border: none;
    cursor: pointer;
    transition: all 0.15s ease;
  }

  .candidate-row:hover {
    background: rgba(255, 255, 255, 0.02) !important;
    padding-right: 80px !important; /* Make room for the sliding hint */
  }

  .cand-action-hint {
    position: absolute;
    right: 12px;
    font-size: 9px;
    font-weight: 700;
    color: #818cf8;
    opacity: 0;
    transform: translateX(4px);
    transition: all 0.15s ease;
    pointer-events: none;
  }

  .candidate-row:hover .cand-action-hint {
    opacity: 1;
    transform: translateX(0);
  }

  /* Compact Persistent Bottom Bar */
  .workspace-bottom-bar {
    background: #121215;
    border-top: 1px solid rgba(255, 255, 255, 0.05);
    padding: 8px 16px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    flex-shrink: 0;
    height: 40px;
  }

  /* Segmented Control Pill styling */
  .segmented-control {
    display: inline-flex;
    background: rgba(255, 255, 255, 0.02);
    border: 1px solid rgba(255, 255, 255, 0.05);
    border-radius: 6px;
    padding: 2px;
  }

  .segment-btn {
    background: transparent;
    border: none;
    color: #71717a;
    font-size: 11px;
    font-weight: 600;
    padding: 4px 8px;
    border-radius: 4px;
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 4px;
    transition: all 0.1s ease;
  }

  .segment-btn:hover {
    color: #cbd5e1;
  }

  .segment-btn.active {
    color: #818cf8;
    background: rgba(129, 140, 248, 0.08);
  }

  /* Hide label text on narrower screens */
  @media (max-width: 600px) {
    .lbl-text {
      display: none;
    }
  }

  /* Super Compact Pagination Control */
  .pagination-compact {
    display: flex;
    align-items: center;
    gap: 6px;
    background: rgba(255, 255, 255, 0.02);
    border: 1px solid rgba(255, 255, 255, 0.05);
    border-radius: 8px;
    padding: 4px 8px;
  }

  .nav-arrow-btn {
    background: transparent;
    border: none;
    color: #818cf8;
    cursor: pointer;
    font-size: 18px;
    font-weight: bold;
    padding: 6px 10px;
    border-radius: 6px;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.1s ease;
    line-height: 1;
  }

  .nav-arrow-btn:hover:not(:disabled) {
    background: rgba(255, 255, 255, 0.04);
    color: #a5b4fc;
  }

  .nav-arrow-btn:disabled {
    color: #3f3f46;
    cursor: not-allowed;
  }

  .pagination-info {
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 0 6px;
  }

  .pagination-num-input {
    background: transparent !important;
    border: none !important;
    color: #ffffff !important;
    font-size: 14px !important;
    font-weight: 700 !important;
    text-align: center !important;
    width: 32px !important;
    padding: 0 !important;
    outline: none !important;
    margin-bottom: 0 !important;
  }

  .divider-slash {
    color: #3f3f46;
    font-size: 14px;
    user-select: none;
  }

  .total-blocks {
    font-size: 14px;
    color: #71717a;
    font-weight: 600;
    user-select: none;
  }

  .continuous-scroll-lbl {
    font-size: 10px;
    color: #52525b;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  /* Unified View Mode Toggle Button */
  .mode-toggle-btn {
    background: rgba(255, 255, 255, 0.02);
    border: 1px solid rgba(255, 255, 255, 0.05);
    color: #cbd5e1;
    font-size: 11px;
    font-weight: 600;
    padding: 6px 10px;
    border-radius: 6px;
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 6px;
    transition: all 0.15s ease;
  }
  .mode-toggle-btn:hover {
    background: rgba(255, 255, 255, 0.05);
    color: #ffffff;
  }
  .mode-toggle-btn.block-active {
    color: #818cf8;
    background: rgba(129, 140, 248, 0.08);
    border-color: rgba(129, 140, 248, 0.2);
  }
  .mode-toggle-btn.block-active:hover {
    background: rgba(129, 140, 248, 0.12);
    border-color: rgba(129, 140, 248, 0.3);
  }

  .alert-confirm-modal {
    max-width: 420px;
    width: 90%;
  }
  .alert-confirm-body {
    padding: 24px 20px;
    text-align: center;
  }
  .alert-confirm-body p {
    font-size: 14px;
    line-height: 1.6;
    color: #d4d4d8;
    margin: 0;
  }

  /* Markdown Typography & Layout inside Fullscreen Split Preview */
  .fullscreen-preview.markdown-rendered {
    color: #e4e4e7;
    font-size: 13.5px;
    line-height: 1.65;
    letter-spacing: -0.05px;
    text-align: left !important;
    overflow-wrap: break-word;
    word-break: break-word;
  }
  .fullscreen-preview.markdown-rendered :global(p) {
    margin: 0 0 12px 0;
    text-align: left !important;
    overflow-wrap: break-word;
    word-break: break-word;
  }
  .fullscreen-preview.markdown-rendered :global(p:last-child) {
    margin-bottom: 0;
  }
  .fullscreen-preview.markdown-rendered :global(h1) {
    font-size: 1.35rem;
    margin: 20px 0 10px 0;
    color: #ffffff;
    font-weight: 800;
    letter-spacing: -0.3px;
    border-bottom: 1px solid rgba(255, 255, 255, 0.06);
    padding-bottom: 4px;
    text-align: left !important;
  }
  .fullscreen-preview.markdown-rendered :global(h2) {
    font-size: 1.15rem;
    margin: 18px 0 8px 0;
    color: #ffffff;
    font-weight: 700;
    letter-spacing: -0.2px;
    text-align: left !important;
  }
  .fullscreen-preview.markdown-rendered :global(h3) {
    font-size: 1rem;
    margin: 14px 0 6px 0;
    color: #818cf8;
    font-weight: 600;
    letter-spacing: -0.1px;
    text-align: left !important;
  }
  .fullscreen-preview.markdown-rendered :global(h4), 
  .fullscreen-preview.markdown-rendered :global(h5), 
  .fullscreen-preview.markdown-rendered :global(h6) {
    font-size: 0.9rem;
    margin: 12px 0 4px 0;
    color: #cbd5e1;
    font-weight: 600;
    text-align: left !important;
  }
  .fullscreen-preview.markdown-rendered :global(code:not(pre code)) {
    background: rgba(129, 140, 248, 0.08);
    border: 1px solid rgba(129, 140, 248, 0.15);
    padding: 2px 5px;
    border-radius: 4px;
    font-size: 11.5px;
    font-family: "Fira Code", Consolas, Monaco, monospace;
    color: #f472b6;
  }
  .fullscreen-preview.markdown-rendered :global(.markdown-code-block) {
    position: relative;
    background: #09090b;
    border: 1px solid rgba(255, 255, 255, 0.06);
    border-radius: 8px;
    margin: 14px 0;
    padding: 12px 14px;
    overflow-x: auto;
  }
  .fullscreen-preview.markdown-rendered :global(.markdown-code-block pre) {
    margin: 0;
    background: transparent;
    border: none;
    padding: 0;
  }
  .fullscreen-preview.markdown-rendered :global(.markdown-code-block code) {
    background: transparent !important;
    border: none !important;
    padding: 0 !important;
    font-size: 12px;
    line-height: 1.5;
    font-family: "Fira Code", "Cascadia Code", Consolas, monospace;
    color: #e4e4e7;
    display: block;
  }
  .fullscreen-preview.markdown-rendered :global(.code-lang-badge) {
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
  .fullscreen-preview.markdown-rendered :global(ul), 
  .fullscreen-preview.markdown-rendered :global(ol) {
    padding-left: 20px;
    margin: 0 0 12px 0;
    text-align: left !important;
  }
  .fullscreen-preview.markdown-rendered :global(li) {
    margin-bottom: 4px;
    text-align: left !important;
  }
  .fullscreen-preview.markdown-rendered :global(li::marker) {
    color: #818cf8;
  }
  .fullscreen-preview.markdown-rendered :global(.task-list-item) {
    list-style-type: none;
    margin-left: -20px;
    margin-bottom: 4px;
  }
  .fullscreen-preview.markdown-rendered :global(.task-list-label) {
    display: flex;
    align-items: flex-start;
    gap: 8px;
    cursor: default;
    user-select: none;
  }
  .fullscreen-preview.markdown-rendered :global(.task-list-checkbox) {
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
  .fullscreen-preview.markdown-rendered :global(.task-list-checkbox:checked) {
    background-color: #818cf8;
    border-color: #818cf8;
  }
  .fullscreen-preview.markdown-rendered :global(.task-list-checkbox:checked::before) {
    content: "✓";
    color: white;
    font-size: 10px;
    font-weight: bold;
  }
  .fullscreen-preview.markdown-rendered :global(.task-list-checkbox:checked + .task-list-text) {
    color: #71717a;
    text-decoration: line-through;
  }
  .fullscreen-preview.markdown-rendered :global(.markdown-blockquote) {
    border-left: 4px solid #818cf8;
    background: rgba(129, 140, 248, 0.03);
    padding: 8px 16px;
    margin: 14px 0;
    border-radius: 0 6px 6px 0;
    color: #a1a1aa;
    font-style: italic;
    text-align: left !important;
  }
  .fullscreen-preview.markdown-rendered :global(.markdown-blockquote p) {
    margin: 0;
  }
  .fullscreen-preview.markdown-rendered :global(.markdown-table-wrapper) {
    overflow-x: auto;
    margin: 14px 0;
    border: 1px solid rgba(255, 255, 255, 0.06);
    border-radius: 8px;
    background: #09090b;
  }
  .fullscreen-preview.markdown-rendered :global(.markdown-table) {
    width: 100%;
    border-collapse: collapse;
    font-size: 12px;
    text-align: left;
  }
  .fullscreen-preview.markdown-rendered :global(.markdown-table th) {
    background: rgba(255, 255, 255, 0.02);
    font-weight: 700;
    color: #ffffff;
    padding: 10px 14px;
    border-bottom: 1px solid rgba(255, 255, 255, 0.06);
  }
  .fullscreen-preview.markdown-rendered :global(.markdown-table td) {
    padding: 10px 14px;
    border-bottom: 1px solid rgba(255, 255, 255, 0.04);
    color: #cbd5e1;
  }
  .fullscreen-preview.markdown-rendered :global(.markdown-table tbody tr:last-child td) {
    border-bottom: none;
  }
  .fullscreen-preview.markdown-rendered :global(.markdown-table tbody tr:nth-child(even)) {
    background: rgba(255, 255, 255, 0.005);
  }
  .fullscreen-preview.markdown-rendered :global(hr) {
    border: none;
    border-top: 1px solid rgba(255, 255, 255, 0.06);
    margin: 18px 0;
  }
  .fullscreen-preview.markdown-rendered :global(a) {
    color: #818cf8;
    text-decoration: none;
    border-bottom: 1px dotted transparent;
    transition: all 0.15s ease;
  }
  .fullscreen-preview.markdown-rendered :global(a:hover) {
    color: #a5b4fc;
    border-bottom-color: #a5b4fc;
  }
  .fullscreen-preview.markdown-rendered :global(img) {
    max-width: 100%;
    border-radius: 6px;
    margin: 10px 0;
    border: 1px solid rgba(255, 255, 255, 0.05);
  }
  .fullscreen-preview.markdown-rendered :global(kbd) {
    background: rgba(255, 255, 255, 0.04);
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: 4px;
    padding: 2px 4px;
    font-size: 10px;
    font-family: inherit;
    color: #e4e4e7;
    box-shadow: 0 1px 0 rgba(0,0,0,0.2);
  }
  
  :global(.tok-operator) {
    color: #cbd5e1 !important;
    opacity: 0.8;
  }

  /* Comments Modal */
  .comments-modal-container {
    width: 500px;
    max-width: 95%;
  }

  .comments-modal-body {
    padding: 16px;
    max-height: 50vh;
    overflow-y: auto;
  }

  .comments-modal-body .comments-empty {
    color: #52525b;
    font-size: 13px;
    text-align: center;
    padding: 20px 0;
  }

  .comments-modal-body .comments-timeline {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .comments-modal-body .comment-item {
    display: flex;
    align-items: flex-start;
    gap: 8px;
    font-size: 13px;
    line-height: 1.4;
  }

  .comments-modal-body .comment-bullet {
    color: #71717a;
    flex-shrink: 0;
  }

  .comments-modal-body .comment-text {
    flex: 1;
    color: #d4d4d8;
    word-break: break-word;
  }

  .comments-modal-body .comment-delete-btn {
    background: transparent;
    border: none;
    cursor: pointer;
    color: #71717a;
    font-size: 16px;
    padding: 0 2px;
    flex-shrink: 0;
    line-height: 1;
  }

  .comments-modal-body .comment-delete-btn:hover {
    color: #f87171;
  }

  .comments-modal-footer {
    display: flex;
    gap: 8px;
    padding: 8px 16px 16px;
    border-top: 1px solid rgba(255, 255, 255, 0.06);
  }

  .comments-modal-footer input {
    flex: 1;
    background: rgba(255, 255, 255, 0.04);
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: 6px;
    padding: 8px 12px;
    color: #e4e4e7;
    font-size: 13px;
    outline: none;
  }

  .comments-modal-footer input::placeholder {
    color: #52525b;
  }

  .comments-modal-footer input:focus {
    border-color: #818cf8;
  }

  .sync-spinner {
    display: inline-block;
    width: 10px;
    height: 10px;
    border: 2px solid rgba(129, 140, 248, 0.3);
    border-top-color: #818cf8;
    border-radius: 50%;
    animation: sync-spin 0.6s linear infinite;
  }

  .sync-spinner-mini {
    display: inline-block;
    width: 8px;
    height: 8px;
    border: 1.5px solid rgba(129, 140, 248, 0.3);
    border-top-color: #818cf8;
    border-radius: 50%;
    animation: sync-spin 0.6s linear infinite;
  }

  .sync-pill {
    animation: sync-pulse 1.5s ease-in-out infinite;
  }

  @keyframes sync-spin {
    to { transform: rotate(360deg); }
  }

  @keyframes sync-pulse {
    0%, 100% { opacity: 0.7; }
    50% { opacity: 1; }
  }
</style>
