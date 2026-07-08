<script context="module">
  import { default as TreeRender } from "./TreeRender.svelte";
  import { default as CardBlock } from "./CardBlock.svelte";
  import { default as RightSidebar } from "./RightSidebar.svelte";
</script>

<script>
  import { onMount, tick } from "svelte";
  import { EventsOn } from "../wailsjs/runtime/runtime.js";
  import { marked } from "marked";
  import PageIcon from "./PageIcon.svelte";
  import {
    GetConnectionStatus,
    GetDbPages,
    GetDiscoveredDevices,
    ConnectToDevice,
    Disconnect,
    AddPage,
    UpdatePage,
    DeletePage,
    RestorePage,
    HardDeletePage,
    MovePage,
    GetCards,
    FetchCards,
    AddCard,
    UpdateCard,
    DeleteCard,
    ReorderCards,
    StartHTMLServer,
    StopHTMLServer,
    SaveImage,
  } from "../wailsjs/go/main/App.js";

  let connectionStatus = "disconnected";
  let discoveredDevices = [];
  let connectedDevice = null;
  let manualIp = "";
  let manualPort = 9090;
  let manualPin = "";
  let connectionError = "";

  let activeWorkspace = "";

  let allPages = [];
  let selectedPage = null;
  let navigationHistory = [];

  let pageCards = [];
  let selectedCardId = null;

  let editorTitle = "";
  let editorEmoji = "📓";
  let saveTimeout;

  let expandedPageIds = {};

  // Custom Modal States
  let showBlockSelectorModal = false;
  let blockInsertIndex = null;
  let cardColumnContainer;

  let showEmojiPickerModal = false;
  let selectedCategory = "smileys";
  let emojiSearchQuery = "";
  let recentEmojis = JSON.parse(localStorage.getItem("recent_emojis") || "[]");

  let showScrollToBottom = false;

  let iconImageInput;
  let customIconUrl = "";

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
          .catch((err) => alert("Failed to save icon image: " + err));
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
  let showPinModal = false;
  let pinInput = "";
  let pendingDeviceToConnect = null;
  let pinModalError = "";
  let isConnecting = false;

  // Link a Subpage Modal States (Moved to root level)
  let showLinkPageModal = false;
  let linkingCard = null;
  let pageSearchQuery = "";

  // Immersive Fullscreen States at App Level
  let editingCard = null;
  let showMarkdownFullscreenModal = false;
  let showCodeFullscreenModal = false;
  let showSitesModal = false;

  // Real-time server live tracking
  let activeLiveCardId = null;
  let activeSitesLocalUrl = "";
  let isSitesLive = false;

  // Fullscreen edit copy
  let fullscreenEditContent = "";
  let fullscreenCodeLang = "javascript";
  let fullscreenCodeContent = "";

  // Sites Card variables inside sandbox modal
  let sitesTab = "edit";
  let sitesName = "";
  let sitesDesc = "";
  let sitesHtml = "";
  let sitesLocalUrl = "";

  // Reactive Fullscreen Markdown Parser
  $: interactivePreviewMarkdown = marked.parse(fullscreenEditContent || "");

  // Immersive Markdown Helpers
  let markdownTextarea;
  $: markdownCharCount = fullscreenEditContent
    ? fullscreenEditContent.length
    : 0;
  $: markdownWordCount = fullscreenEditContent
    ? fullscreenEditContent.trim().split(/\s+/).filter(Boolean).length
    : 0;

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
  let lineNumbersElement = null;
  let preElement = null;

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

  // Custom regex syntax highlighter supporting JS, Go, Python, HTML/CSS, Dart, etc.
  function highlightCode(code, lang) {
    if (!code)
      return '<span style="color: #4b5563; font-style: italic;">// Start writing your code here...</span>';

    let escaped = code
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
      "export",
      "class",
      "void",
      "final",
      "def",
      "package",
      "func",
      "interface",
      "from",
      "default",
      "as",
      "new",
      "this",
      "extends",
      "super",
      "try",
      "catch",
      "finally",
      "async",
      "await",
      "break",
      "continue",
      "switch",
      "case",
      "throw",
      "true",
      "false",
      "null",
    ];
    const builtins = [
      "console",
      "window",
      "document",
      "process",
      "Object",
      "Array",
      "String",
      "Number",
      "Boolean",
      "Math",
      "JSON",
    ];

    const stringRegex =
      /("(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*'|`(?:\\.|[^`\\])*`)/g;
    const commentRegex = /(\/\/[^\n]*|#[^\n]*)/g;
    const multiLineCommentRegex = /(\/\*[\s\S]*?\*\/)/g;

    const tokens = [];
    let counter = 0;
    function saveToken(text, className) {
      const placeholder = `___TOK_PLCHLDR_${counter++}___`;
      tokens.push({
        placeholder,
        html: `<span class="${className}">${text}</span>`,
      });
      return placeholder;
    }

    // 1. Multi-line comments
    escaped = escaped.replace(multiLineCommentRegex, (m) =>
      saveToken(m, "tok-comment"),
    );
    // 2. Single-line comments
    escaped = escaped.replace(commentRegex, (m) => saveToken(m, "tok-comment"));
    // 3. Strings
    escaped = escaped.replace(stringRegex, (m) => saveToken(m, "tok-string"));
    // 4. Numbers
    escaped = escaped.replace(/\b(\d+(?:\.\d+)?)\b/g, (m) =>
      saveToken(m, "tok-number"),
    );
    // 5. Keywords
    const keywordsRegex = new RegExp(`\\b(${keywords.join("|")})\\b`, "g");
    escaped = escaped.replace(keywordsRegex, (m) =>
      saveToken(m, "tok-keyword"),
    );
    // 6. Built-ins
    const builtinsRegex = new RegExp(`\\b(${builtins.join("|")})\\b`, "g");
    escaped = escaped.replace(builtinsRegex, (m) =>
      saveToken(m, "tok-builtin"),
    );
    // 7. Functions
    escaped = escaped.replace(/\b(\w+)(?=\s*\()/g, (m) =>
      saveToken(m, "tok-function"),
    );

    // Restore placeholder tokens
    for (let i = tokens.length - 1; i >= 0; i--) {
      escaped = escaped.replace(tokens[i].placeholder, tokens[i].html);
    }

    if (escaped.endsWith("\n") || escaped === "") {
      escaped += " ";
    }

    return escaped;
  }

  $: lineNumbers = (fullscreenCodeContent.match(/\n/g) || []).length + 1;
  $: highlightedCodeHtml = highlightCode(
    fullscreenCodeContent,
    fullscreenCodeLang,
  );

  // Rich Keyboard-Matching Emojis Catalog grouped by category
  const emojiCategories = [
    {
      id: "custom_image",
      label: "🖼️ Custom Image",
    },
    {
      id: "smileys",
      label: "😀 Smileys & People",
      emojis: [
        "😀",
        "😃",
        "😄",
        "😁",
        "😆",
        "😅",
        "😂",
        "🤣",
        "😊",
        "😇",
        "🙂",
        "🙃",
        "😉",
        "😌",
        "😍",
        "🥰",
        "😘",
        "😗",
        "😙",
        "😚",
        "😋",
        "😛",
        "😝",
        "😜",
        "🤪",
        "🤨",
        "🧐",
        "🤓",
        "😎",
        "🥸",
        "🤩",
        "🥳",
        "😏",
        "😒",
        "😞",
        "😔",
        "😟",
        "😕",
        "🙁",
        "☹️",
        "😣",
        "😖",
        "😫",
        "😩",
        "🥺",
        "😢",
        "😭",
        "😤",
        "😠",
        "😡",
        "🤬",
        "🤯",
        "😳",
        "🥵",
        "🥶",
        "😱",
        "😨",
        "😰",
        "😥",
        "😓",
        "🤔",
        "🫣",
        "🤭",
        "🫢",
        "🫡",
        "🤫",
        "🫠",
        "🤥",
        "😶",
        "😐",
        "😑",
        "😬",
        "🫨",
        "😴",
        "🤤",
        "😪",
        "😵",
        "😵💫",
        "🤐",
        "🥴",
        "🤢",
        "🤮",
        "🤧",
        "😷",
        "🤒",
        "🤕",
        "😈",
        "👿",
        "👹",
        "👺",
        "💀",
        "☠️",
        "👻",
        "👽",
        "👾",
        "🤖",
        "🎃",
        "😺",
        "😸",
        "😹",
        "😻",
        "😼",
        "😽",
        "😾",
        "😿",
        "🙀",
        "👋",
        "🤚",
        "🖐️",
        "✋",
        "🖖",
        "👌",
        "🤌",
        "🤏",
        "✌️",
        "🤞",
        "🫰",
        "🤟",
        "🤘",
        "🤙",
        "👈",
        "👉",
        "👆",
        "🖕",
        "👇",
        "☝️",
        "👍",
        "👎",
        "✊",
        "👊",
        "🤛",
        "🤜",
        "👏",
        "🙌",
        "🫶",
        "👐",
        "🤲",
        "🤝",
        "🙏",
        "✍️",
        "💅",
        "🤳",
        "💪",
        "🦾",
        "🦿",
        "🦵",
        "🦶",
        "👂",
        "🦻",
        "👃",
        "🧠",
        "🫀",
        "🫁",
        "🦷",
        "🦴",
        "👀",
        "👁️",
        "👅",
        "👄",
        "💋",
        "🩸",
        "👤",
        "👥",
        "🫂",
      ],
    },
    {
      id: "animals",
      label: "🐱 Animals & Nature",
      emojis: [
        "🐶",
        "🐱",
        "🐭",
        "🐹",
        "🐰",
        "🦊",
        "🐻",
        "🐼",
        "🐨",
        "🐯",
        "🦁",
        "🐮",
        "🐷",
        "🐽",
        "🐸",
        "🐵",
        "🙈",
        "🙉",
        "🙊",
        "🐒",
        "🐔",
        "🐧",
        "🐦",
        "🐤",
        "🐣",
        "🐥",
        "🦆",
        "🦅",
        "🦉",
        "🪱",
        "🐛",
        "🦋",
        "🐌",
        "🐞",
        "🐜",
        "🪰",
        "🪲",
        "🪳",
        "🦗",
        "🕷️",
        "🕸️",
        "🐢",
        "🐍",
        "🦎",
        "🐙",
        "🦑",
        "🦞",
        "🦀",
        "🐡",
        "🐠",
        "🐟",
        "🐬",
        "🐳",
        "🐋",
        "🦈",
        "🐊",
        "🐅",
        "🐆",
        "🦓",
        "🦍",
        "🦧",
        "🦣",
        "🐘",
        "🦛",
        "🦏",
        "🐪",
        "🐫",
        "🦒",
        "🦘",
        "🦬",
        "🐃",
        "🐂",
        "🐄",
        "🐎",
        "🐖",
        "🐏",
        "🐑",
        "🦙",
        "🐐",
        "🦌",
        "🐕",
        "🐩",
        "🦮",
        "🐈",
        "🐓",
        "🦃",
        "🦚",
        "🦜",
        "🕊️",
        "🐇",
        "🦝",
        "🦨",
        "🦡",
        "🦦",
        "🦥",
        "🐿️",
        "🦔",
        "🐾",
        "🐉",
        "🐲",
        "🌵",
        "🎄",
        "🌲",
        "🌳",
        "🌴",
        "🪵",
        "🌱",
        "🌿",
        "☘️",
        "🍀",
        "🍁",
        "🍂",
        "🍃",
        "🍄",
        "🐚",
        "🪸",
        "🪨",
        "🌾",
        "💐",
        "🌷",
        "🌹",
        "🥀",
        "🌺",
        "🌸",
        "🌼",
        "🌻",
        "🌞",
        "🌝",
        "🌛",
        "🌜",
        "🌚",
        "🌕",
        "🌖",
        "🌗",
        "🌘",
        "🌑",
        "🌒",
        "🌓",
        "🌔",
        "🌙",
        "🌎",
        "🌍",
        "🌏",
        "🪐",
        "💫",
        "⭐️",
        "🌟",
        "✨",
        "⚡️",
        "☄️",
        "💥",
        "🔥",
        "🌪️",
        "🌈",
        "☀️",
        "🌤️",
        "⛅️",
        "🌥️",
        "🌦️",
        "☁️",
        "🌧️",
        "⛈️",
        "🌩️",
        "🌨️",
        "❄️",
        "☃️",
        "⛄️",
        "🌬️",
        "💨",
        "💧",
        "💦",
        "🫧",
        "☔️",
        "🌊",
        "🌫️",
      ],
    },
    {
      id: "food",
      label: "🍏 Food & Drink",
      emojis: [
        "🍏",
        "🍎",
        "🍐",
        "🍊",
        "🍋",
        "🍌",
        "🍉",
        "🍇",
        "🍓",
        "🫐",
        "🍈",
        "🍒",
        "🍑",
        "🥭",
        "🍍",
        "🥥",
        "🥝",
        "🍅",
        "🍆",
        "🥑",
        "🥦",
        "🥬",
        "🥒",
        "🌶️",
        "🫑",
        "🌽",
        "🥕",
        "🫒",
        "🧄",
        "🧅",
        "🥔",
        "🍠",
        "🥐",
        "🍞",
        "🥖",
        "🥨",
        "🧀",
        "🍳",
        "🥞",
        "🥓",
        "🥩",
        "🍗",
        "🍖",
        "🌭",
        "🍔",
        "🍟",
        "🍕",
        "🥪",
        "🥙",
        "🫓",
        "🌮",
        "🌯",
        "🫔",
        "🥗",
        "🥘",
        "🍲",
        "🫕",
        "🥫",
        "🍝",
        "🍜",
        "🍛",
        "🍣",
        "🍱",
        "🥟",
        "🍤",
        "🍙",
        "🍘",
        "🍥",
        "🥠",
        "🥮",
        "🍢",
        "🍡",
        "🍧",
        "🍨",
        "🍦",
        "🥧",
        "🍰",
        "🎂",
        "🧁",
        "🍮",
        "🍭",
        "🍬",
        "🍫",
        "🍿",
        "🍩",
        "🍪",
        "🌰",
        "🥜",
        "🫘",
        "🍯",
        "🥛",
        "🍼",
        "☕️",
        "🍵",
        "🧃",
        "🥤",
        "🧋",
        "🍶",
        "🍺",
        "🍻",
        "🥂",
        "🍷",
        "🥃",
        "🍸",
        "🍹",
        "🧉",
        "🍾",
        "🧊",
        "🥢",
        "🍽️",
        "🍴",
        "🥄",
      ],
    },
    {
      id: "activity",
      label: "⚽️ Activity & Travel",
      emojis: [
        "⚽️",
        "🏀",
        "🏈",
        "⚾️",
        "🥎",
        "🎾",
        "🏐",
        "🏉",
        "🥏",
        "🎱",
        "🪀",
        "🏸",
        "🏒",
        "🥍",
        "🏹",
        "🤿",
        "🥊",
        "🥋",
        "🥅",
        "⛳️",
        "⛸️",
        "🎽",
        "🎿",
        "🛷",
        "🥌",
        "🎯",
        "🪗",
        "🪘",
        "🎮",
        "🕹️",
        "🎰",
        "🎲",
        "🧩",
        "🧸",
        "🪅",
        "🪩",
        "🎨",
        "🖼️",
        "🧵",
        "🪡",
        "🧶",
        "🎸",
        "🎹",
        "🎺",
        "🎻",
        "🥁",
        "🪕",
        "🎧",
        "🎤",
        "🎬",
        "🎟️",
        "🎫",
        "🎭",
        "🎪",
        "🧗",
        "🏋️",
        "🚴",
        "🏃",
        "🚶",
        "🚗",
        "🚕",
        "🚙",
        "🚌",
        "🚎",
        "🏎️",
        "🚓",
        "🚑",
        "🚒",
        "🚐",
        "🛻",
        "🚚",
        "🚛",
        "🚜",
        "🛵",
        "🏍️",
        "🛺",
        "🚲",
        "🛴",
        "🛼",
        "🚏",
        "🛣️",
        "🛤️",
        "🚢",
        "⛵️",
        "🚤",
        "🛥️",
        "🛳️",
        "⛴️",
        "🛶",
        "🛸",
        "🚁",
        "🛩️",
        "✈️",
        "🛫",
        "🛬",
        "🚀",
        "🛰️",
        "⚓️",
        "🗺️",
        "🧭",
        "🏔️",
        "⛰️",
        "🌋",
        "🗻",
        "🏕️",
        "🏖️",
        "🏜️",
        "🏝️",
        "🏞️",
        "🏛️",
        "🏗️",
        "🧱",
        "🏘️",
        "🏚️",
        "🏠",
        "🏡",
        "🏢",
        "🏣",
        "🏤",
        "🏥",
        "🏦",
        "🏨",
        "🏩",
        "🏪",
        "🏫",
        "🏬",
        "🏭",
        "🏯",
        "🏰",
        "💒",
        "🗼",
        "🗽",
        "🕌",
        "⛪️",
        "🛕",
        " synagogues",
        "⛩️",
        "🕋",
        "⛲️",
        "⛺️",
        "🌁",
        "🌃",
        "🏙️",
        "🌅",
        "🌄",
        "🌇",
        "🌆",
        "🌉",
        "🎠",
        "🎡",
        "🎢",
        "💈",
      ],
    },
    {
      id: "objects",
      label: "💡 Objects & Symbols",
      emojis: [
        "⌚️",
        "📱",
        "📲",
        "💻",
        "⌨️",
        "🖥️",
        "🖨️",
        "🖱️",
        "🖲️",
        "💽",
        "💾",
        "💿",
        "📀",
        "📼",
        "📷",
        "📸",
        "📹",
        "🎥",
        "📽️",
        "🎞️",
        "📞",
        "☎️",
        "📟",
        "📠",
        "📺",
        "📻",
        "🎙️",
        "🎚️",
        "🎛️",
        "🧭",
        "⏰",
        "⌛️",
        "⏳",
        "🔋",
        "🔌",
        "💡",
        "🕯️",
        "🪔",
        "🗑️",
        "🛢️",
        "💸",
        "💵",
        "💴",
        "💶",
        "💷",
        "🪙",
        "💰",
        "💳",
        "💎",
        "⚖️",
        "🪜",
        "🔧",
        "🔨",
        "⚒️",
        "🛠️",
        "⛏️",
        "🪚",
        "🔩",
        "⚙️",
        "🧱",
        "⛓️",
        "🧲",
        "🧯",
        "🔫",
        "bomb",
        "🧨",
        "🪓",
        "🔪",
        "🗡️",
        "⚔️",
        "🛡️",
        "🚬",
        "⚰️",
        "⚱️",
        "🏺",
        "🔮",
        "📿",
        "🧿",
        "💈",
        "🧫",
        "🧪",
        "🔬",
        "🔭",
        "📡",
        "💉",
        "💊",
        "🩹",
        "🩺",
        "🚪",
        "🛗",
        "🪞",
        "🪟",
        "🛏️",
        "🛋️",
        "🪑",
        "🚽",
        "🪠",
        "🚿",
        "🛁",
        "🧼",
        "🪥",
        "🪮",
        "🧴",
        "🧹",
        "🧺",
        "🧻",
        "🪣",
        "🪟",
        "🗝️",
        "🔑",
        "🪤",
        "📦",
        "🏷️",
        "✉️",
        "📩",
        "📨",
        "📧",
        "📤",
        "📥",
        "📪",
        "📫",
        "📬",
        "📭",
        "📮",
        "🗳️",
        "✏️",
        "✒️",
        "🖋️",
        "🖊️",
        "🖌️",
        "🖍️",
        "📝",
        "📁",
        "📂",
        "🗂️",
        "📅",
        "📆",
        "🗒️",
        "🗓️",
        "🪪",
        "𗃏",
        "𗄀",
        "📋",
        "📌",
        "📍",
        "📎",
        "🖇️",
        "📏",
        "📐",
        "🧮",
        "🔐",
        "🔏",
        "🔒",
        "🔓",
        "❤️",
        "🧡",
        "💛",
        "💚",
        "💙",
        "💜",
        "🖤",
        "🤍",
        "🤎",
        "💔",
        "❣️",
        "💕",
        "💞",
        "💓",
        "💗",
        "💖",
        "💘",
        "💝",
        "💟",
        "☮️",
        "✝️",
        "☪️",
        "🕉️",
        "☸️",
        "✡️",
        "🔯",
        "🕎",
        "☯️",
        "☦️",
        "🛐",
        "♈️",
        "♉️",
        "♊️",
        "♋️",
        "♌️",
        "♍️",
        "♎️",
        "♏️",
        "♐️",
        "♑️",
        "♒️",
        "♓️",
        "🆔",
        "📯",
        "🔔",
        "🔕",
        "📣",
        "📢",
        "💬",
        "💭",
        "🗯️",
        "🏁",
        "🚩",
        "🎌",
        "🏴",
        "🏳️",
        "🏳️🌈",
        "🏴☠️",
      ],
    },
  ];

  // Sidebar widths (pixels)
  let leftSidebarWidth = 260;
  let rightSidebarWidth = 260;
  let dragging = null; // 'left' | 'right' | null
  let dragStartX = 0;
  let dragStartWidth = 0;

  $: rootPages = allPages.filter(
    (p) => !p.parent_id && p.relation_type !== "sidepage",
  );
  $: sidePages = allPages.filter(
    (p) => p.parent_id === selectedPage?.id && p.relation_type === "sidepage",
  );

  $: currentCategoryEmojis = (() => {
    const cat = emojiCategories.find((c) => c.id === selectedCategory);
    return cat ? (cat.emojis || []) : [];
  })();

  $: allEmojisFiltered = (() => {
    if (!emojiSearchQuery) return [];
    const q = emojiSearchQuery.toLowerCase();
    const results = [];
    for (const cat of emojiCategories) {
      if (!cat.emojis) continue;
      for (const e of cat.emojis) {
        if (e.toLowerCase().includes(q)) results.push(e);
      }
    }
    return results;
  })();

  // Filter nested subpages candidates globally for the link modal
  $: filteredLinkCandidates = (() => {
    if (!linkingCard) return [];
    const currentPageId = linkingCard.page_id;
    const currentPage = allPages.find(p => p.id === currentPageId);
    if (!currentPage) return [];

    const directChildrenIds = new Set(
      allPages
        .filter((p) => p.parent_id === currentPage.id && p.relation_type !== "sidepage")
        .map((p) => p.id),
    );

    const candidates = allPages.filter((p) => {
      if (p.id === currentPageId || p.relation_type === "sidepage") return false;
      if (p.parent_id === currentPage.id) return true;
      return directChildrenIds.has(p.parent_id);
    });

    if (!pageSearchQuery) return candidates;
    const query = pageSearchQuery.toLowerCase();
    return candidates.filter(
      (p) =>
        (p.title || "").toLowerCase().includes(query) ||
        (p.emoji || "").includes(query),
    );
  })();

  onMount(async () => {
    try {
      connectionStatus = await GetConnectionStatus();
      discoveredDevices = await GetDiscoveredDevices();
      allPages = await GetDbPages();
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
    });
    EventsOn("workspace-status", (data) => {
      activeWorkspace = data.activeWorkspace;
    });
    EventsOn("cards-update", (data) => {
      const pid = data.pageId || data.page_id;
      if (selectedPage && pid === selectedPage.id) {
        pageCards = data.cards || [];
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
    dragStartWidth = panel === "left" ? leftSidebarWidth : rightSidebarWidth;
    e.preventDefault();
  }

  function onDragMove(e) {
    if (!dragging) return;
    const delta = e.clientX - dragStartX;
    if (dragging === "left") {
      leftSidebarWidth = Math.max(180, Math.min(450, dragStartWidth + delta));
    } else if (dragging === "right") {
      rightSidebarWidth = Math.max(180, Math.min(450, dragStartWidth - delta));
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

  function toggleExpand(pageId) {
    expandedPageIds[pageId] = !expandedPageIds[pageId];
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

  function createPage(parentId = "", relationType = "subpage") {
    AddPage(parentId, relationType, "New Page", "📝")
      .then(() => {
        if (parentId) expandedPageIds[parentId] = true;
      })
      .catch((err) => alert("Failed: " + err));
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
      } else {
        sortOrder =
          pageCards.length > 0
            ? Math.max(...pageCards.map((c) => c.sort_order)) + 1
            : 0;
      }

      let defaultContent = "";
      if (type === "code") {
        defaultContent =
          'javascript\nconsole.log("Hello from Cero Code Block!");\n';
      } else if (type === "sites") {
        defaultContent = JSON.stringify({
          name: "Interactive Site Card",
          description: "Sandboxed iframe renderer preview widget",
          html: "<h1>Welcome directly to Sandboxed Environment!</h1>\n<p>Change this code inside the editor tab to render customized HTML components.</p>",
        });
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
      alert("Failed: " + e);
    }
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

  function deletePage(id) {
    if (!confirm("Archive this page and all its subpages?")) return;
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
    const pages = allPages.filter(
      (p) => p.id !== id && p.relation_type !== "sidepage",
    );
    const options = pages.map(
      (p, i) => `${i + 1}: ${p.emoji} ${p.title || "Untitled"}`,
    );
    options.unshift("0: 📂 Root Level");
    const choice = prompt("Move to:\n" + options.join("\n"));
    if (choice == null) return;
    const idx = parseInt(choice);
    if (idx === 0) MovePage(id, "");
    else if (idx > 0 && idx <= pages.length) MovePage(id, pages[idx - 1].id);
  }

  function getChildrenOf(parentId) {
    return allPages.filter(
      (p) => p.parent_id === parentId && p.relation_type !== "sidepage",
    );
  }

  // Immersive Modal Handlers at App Level
  function openMarkdownFullscreen(c, initialContent) {
    editingCard = c;
    fullscreenEditContent = initialContent;
    showMarkdownFullscreenModal = true;
  }

  function saveMarkdownFullscreen() {
    if (!editingCard) return;
    UpdateCard(editingCard.id, editingCard.page_id, fullscreenEditContent, editingCard.comment || "")
      .then(() => {
        const idx = pageCards.findIndex((c) => c.id === editingCard.id);
        if (idx !== -1) pageCards[idx].content = fullscreenEditContent;
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
    UpdateCard(editingCard.id, editingCard.page_id, combined, editingCard.comment || "")
      .then(() => {
        const idx = pageCards.findIndex((c) => c.id === editingCard.id);
        if (idx !== -1) pageCards[idx].content = combined;
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
    UpdateCard(editingCard.id, editingCard.page_id, combined, editingCard.comment || "")
      .then(() => {
        const idx = pageCards.findIndex((c) => c.id === editingCard.id);
        if (idx !== -1) pageCards[idx].content = combined;
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
    UpdateCard(linkingCard.id, linkingCard.page_id, target.id, linkingCard.comment || "")
      .then(() => {
        const idx = pageCards.findIndex((c) => c.id === linkingCard.id);
        if (idx !== -1) {
          pageCards[idx].content = target.id;
          pageCards = pageCards; // trigger updates
        }
        showLinkPageModal = false;
        linkingCard = null;
      })
      .catch((err) => console.error(err));
  }

  async function handleCreateNewPageAndLink() {
    if (!linkingCard) return;
    try {
      const newPageId = await AddPage(
        linkingCard.page_id,
        "subpage",
        "New Subpage Link",
        "📝",
      );

      if (newPageId) {
        await UpdateCard(linkingCard.id, linkingCard.page_id, newPageId, linkingCard.comment || "");
        const idx = pageCards.findIndex((c) => c.id === linkingCard.id);
        if (idx !== -1) {
          pageCards[idx].content = newPageId;
          pageCards = pageCards;
        }
        showLinkPageModal = false;
        linkingCard = null;
      }
    } catch (err) {
      console.error("Failed to create new page & link:", err);
      alert("Could not create and link subpage: " + err);
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
      <div class="status-badge {connectionStatus}">
        <span class="dot"></span>{connectionStatus}
      </div>
    </div>

    <!-- Read-only Active Workspace Details -->
    <div class="sidebar-section active-workspace-card">
      <div class="section-label">Active Workspace</div>
      <div class="active-ws-display">
        <span class="ws-icon">🗄️</span>
        <span class="ws-name">{activeWorkspace || "Personal"}</span>
      </div>
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
          <button class="btn-sm full" on:click={handleDisconnect}
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
              <button class="btn-sm primary" on:click={() => handleConnect(d)}
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
          <button class="icon-btn" on:click={() => createPage("")}>+</button>
        {/if}
      </div>
      <div class="tree-scroll">
        {#if connectionStatus !== "connected"}
          <div class="tree-empty">Connect to phone to view pages.</div>
        {:else if rootPages.length === 0}
          <div class="tree-empty">No pages yet.</div>
        {:else}
          {#each rootPages as page}
            <svelte:component
              this={TreeRender}
              {page}
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
          <button class="connect-btn" on:click={handleConnectManually}>
            <span class="btn-icon">🔗</span> Link Device
          </button>
          {#if connectionError}
            <div class="connection-error-box">
              <span class="err-icon">⚠️</span>
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
    on:mousedown={(e) => onDragStart("left", e)}
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
          <span class="big-icon">📝</span>
          <h2>Select or Create a Page</h2>
          <p>Choose from sidebar or create a new page.</p>
          <button class="btn primary" on:click={() => createPage("")}
            >Create New Page</button
          >
        </div>
      </div>
    {:else}
      <div class="editor-header">
        <div class="breadcrumbs">
          {#if navigationHistory.length > 0}
            <button class="back-btn" on:click={goBack}>← Back</button>
            <span class="sep">|</span>
          {/if}
          <span class="bc-root">{activeWorkspace}</span>
          <span class="sep">/</span>
          <span class="bc-current">
            <PageIcon emoji={selectedPage.emoji} size={12} />
            {selectedPage.title || "Untitled"}
          </span>
        </div>
        <div class="header-actions">
          <button class="btn-sm" on:click={() => movePage(selectedPage.id)}
            >Move</button
          >
          <button
            class="btn-sm danger"
            on:click={() => deletePage(selectedPage.id)}>Archive</button
          >
        </div>
      </div>

      <div class="title-bar">
        <!-- Interactive Page Emoji Picker Trigger -->
        <button
          class="emoji-input-btn"
          on:click={() => (showEmojiPickerModal = true)}
        >
          <PageIcon emoji={editorEmoji || "📓"} size={44} />
        </button>
        <input
          id="editor-title-input"
          class="title-input"
          type="text"
          bind:value={editorTitle}
          on:input={savePageDebounced}
          placeholder="Untitled"
        />
      </div>

      <div class="card-column" bind:this={cardColumnContainer} on:scroll={handleCardColumnScroll}>
        {#each pageCards as card, index (card.id)}
          <div
            class="card-slot"
            draggable="true"
            on:dragstart={(e) => {
              e.dataTransfer.setData("text/plain", index.toString());
              e.dataTransfer.effectAllowed = "move";
            }}
            on:dragover|preventDefault={(e) => {
              e.dataTransfer.dropEffect = "move";
            }}
            on:drop|preventDefault={(e) => handleCardDrop(e, index)}
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
              on:editMarkdown={(e) =>
                openMarkdownFullscreen(card, e.detail.content)}
              on:editCode={(e) => openCodeFullscreen(card, e.detail.content)}
              on:editSites={(e) =>
                openSitesFullscreen(
                  card,
                  e.detail.name,
                  e.detail.description,
                  e.detail.html,
                )}
              on:openLinkModal={(e) => {
                linkingCard = e.detail.card;
                pageSearchQuery = "";
                showLinkPageModal = true;
              }}
            />
          </div>
          <div
            class="insert-slot"
            on:dragover|preventDefault
            on:drop|preventDefault={(e) => handleCardDrop(e, index + 1)}
          >
            <button
              class="insert-btn"
              on:click={() => {
                blockInsertIndex = index + 1;
                showBlockSelectorModal = true;
              }}>+</button
            >
          </div>
        {/each}

        {#if pageCards.length === 0}
          <div class="empty-cards">
            <span class="empty-icon">📂</span>
            <h3>This page is empty</h3>
            <p>Add blocks to start writing.</p>
            <div class="empty-actions">
              <button
                class="btn secondary"
                on:click={() => handleAddCardType("markdown")}
                >📝 Markdown</button
              >
              <button
                class="btn secondary"
                on:click={() => handleAddCardType("image")}>🖼️ Image</button
              >
              <button
                class="btn secondary"
                on:click={() => handleAddCardType("subpage_link")}
                >🔗 Link</button
              >
              <button
                class="btn secondary"
                on:click={() => handleAddCardType("code")}>💻 Code Block</button
              >
              <button
                class="btn secondary"
                on:click={() => handleAddCardType("sites")}>🌐 HTML Site</button
              >
            </div>
          </div>
        {/if}

        <button
          class="add-card-btn"
          on:click={() => {
            blockInsertIndex = null;
            showBlockSelectorModal = true;
          }}>+ Add Card</button
        >

        {#if showScrollToBottom}
          <button class="scroll-to-bottom-btn" on:click={scrollToBottom}>
            ↓
          </button>
        {/if}
      </div>
    {/if}
  </section>

  <!-- RIGHT DRAG HANDLE -->
  <div
    class="drag-handle right-handle"
    on:mousedown={(e) => onDragStart("right", e)}
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
</main>

<!-- Block Selector Modal -->
{#if showBlockSelectorModal}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <div
    class="modal-backdrop"
    on:click|self={() => (showBlockSelectorModal = false)}
    role="button"
    tabindex="-1"
  >
    <div class="modal-container block-selector-modal">
      <div class="modal-header">
        <h3>Add Block</h3>
        <button
          class="close-btn"
          on:click={() => (showBlockSelectorModal = false)}>&times;</button
        >
      </div>
      <div class="modal-body block-options-grid">
        <button
          class="block-option-row"
          on:click={() => handleAddCardType("markdown")}
        >
          <span class="block-icon">📝</span>
          <div class="block-desc">
            <span class="block-title">Markdown</span>
            <span class="block-subtitle"
              >Write formatted text, headers, checklist items, or code blocks.</span
            >
          </div>
        </button>
        <button
          class="block-option-row"
          on:click={() => handleAddCardType("image")}
        >
          <span class="block-icon">🖼️</span>
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
          on:click={() => handleAddCardType("subpage_link")}
        >
          <span class="block-icon">🔗</span>
          <div class="block-desc">
            <span class="block-title">Subpage Link</span>
            <span class="block-subtitle"
              >Link directly to another nested page in your journal hierarchy.</span
            >
          </div>
        </button>
        <button
          class="block-option-row"
          on:click={() => handleAddCardType("code")}
        >
          <span class="block-icon">💻</span>
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
          on:click={() => handleAddCardType("sites")}
        >
          <span class="block-icon">🌐</span>
          <div class="block-desc">
            <span class="block-title">HTML Site block</span>
            <span class="block-subtitle"
              >Creates customized templates with embedded sandbox HTML code
              rendering.</span
            >
          </div>
        </button>
      </div>
    </div>
  </div>
{/if}

<!-- Keyboard Emoji Picker Modal -->
{#if showEmojiPickerModal}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <div
    class="modal-backdrop"
    on:click|self={() => (showEmojiPickerModal = false)}
    role="button"
    tabindex="-1"
  >
    <div class="modal-container emoji-picker-modal">
      <div class="modal-header">
        <h3>Select Icon</h3>
        <button
          class="close-btn"
          on:click={() => (showEmojiPickerModal = false)}>&times;</button
        >
      </div>
      <div class="emoji-search-bar">
        <span class="search-icon">🔍</span>
        <input
          type="text"
          bind:value={emojiSearchQuery}
          placeholder="Search emojis..."
          autofocus
        />
        {#if emojiSearchQuery}
          <button class="clear-btn" on:click={() => (emojiSearchQuery = "")}
            >&times;</button
          >
        {/if}
      </div>
      <input type="file" bind:this={iconImageInput} accept="image/*" style="display:none" on:change={handleIconUpload} />
      {#if !emojiSearchQuery && recentEmojis.length > 0}
        <div class="emoji-recent-row">
          <span class="recent-label">Recent</span>
          {#each recentEmojis as e}
            <button
              class="emoji-select-btn"
              on:click={() => {
                editorEmoji = e;
                showEmojiPickerModal = false;
                recentEmojis = [e, ...recentEmojis.filter(em => em !== e)].slice(0, 8);
                localStorage.setItem("recent_emojis", JSON.stringify(recentEmojis));
                savePageImmediate();
              }}
            >
              <PageIcon {emoji} size={20} />
            </button>
          {/each}
          <button class="clear-recent-btn" on:click={() => {
            recentEmojis = [];
            localStorage.setItem("recent_emojis", "[]");
          }}>×</button>
        </div>
      {/if}
      {#if !emojiSearchQuery}
        <div class="emoji-picker-tabs">
          {#each emojiCategories as category}
            <button
              class="emoji-tab-btn"
              class:active={selectedCategory === category.id}
              on:click={() => (selectedCategory = category.id)}
            >
              {category.label.split(" ")[0]}
            </button>
          {/each}
        </div>
      {/if}
      <div class="modal-body emoji-picker-body">
        {#if selectedCategory === "custom_image" && !emojiSearchQuery}
          <div class="custom-icon-section">
            <div class="custom-icon-header">
              <span class="custom-icon-icon">🖼️</span>
              <h4>Use Custom Page Icon</h4>
              <p>Import an image from your computer or use an online URL.</p>
            </div>
            <button class="btn primary" style="width:100%; margin-bottom: 8px;" on:click={() => iconImageInput.click()}>
              Pick Image from Computer
            </button>
            <div class="custom-icon-url-row">
              <input type="text" bind:value={customIconUrl} placeholder="https://example.com/icon.png" />
              <button class="btn primary" on:click={applyCustomIconUrl}>Apply</button>
            </div>
          </div>
        {:else}
          <div class="emoji-grid">
            {#each emojiSearchQuery ? allEmojisFiltered : currentCategoryEmojis as emoji}
              <button
                class="emoji-select-btn"
                on:click={() => {
                  editorEmoji = emoji;
                  showEmojiPickerModal = false;
                  recentEmojis = [emoji, ...recentEmojis.filter(e => e !== emoji)].slice(0, 8);
                  localStorage.setItem("recent_emojis", JSON.stringify(recentEmojis));
                  savePageImmediate();
                }}
              >
                <PageIcon {emoji} size={20} />
              </button>
            {/each}
          </div>
        {/if}
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
    on:click|self={() => {
      if (!isConnecting) showPinModal = false;
    }}
    role="button"
    tabindex="-1"
  >
    <div class="modal-container pin-entry-modal">
      <div class="modal-header">
        <h3>🔑 Link Mobile Device</h3>
        <button
          class="close-btn"
          on:click={() => (showPinModal = false)}
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
            on:keydown={(e) => {
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
          on:click={() => (showPinModal = false)}
          disabled={isConnecting}
        >
          Cancel
        </button>
        <button
          class="btn primary"
          on:click={submitPin}
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

      <div class="header-buttons">
        <button
          class="btn secondary"
          on:click={() => (showMarkdownFullscreenModal = false)}>Cancel</button
        >
        <button class="btn primary" on:click={saveMarkdownFullscreen}
          >Save Changes</button
        >
      </div>
    </div>

    <div class="fullscreen-workspace">
      <div class="editor-pane-container">
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
            on:click={() => insertMarkdownSymbol("**", "**")}
            ><strong>B</strong></button
          >
          <button
            class="tool-btn"
            title="Italic"
            on:click={() => insertMarkdownSymbol("*", "*")}><em>I</em></button
          >
          <button
            class="tool-btn"
            title="Header"
            on:click={() => insertMarkdownSymbol("### ")}>H3</button
          >
          <button
            class="tool-btn"
            title="List"
            on:click={() => insertMarkdownSymbol("- ")}>• List</button
          >
          <button
            class="tool-btn"
            title="Checklist"
            on:click={() => insertMarkdownSymbol("- [ ] ")}>☑ Todo</button
          >
          <button
            class="tool-btn"
            title="Inline Code"
            on:click={() => insertMarkdownSymbol("`", "`")}>&lt;/&gt;</button
          >
          <button
            class="tool-btn"
            title="Code Block"
            on:click={() => insertMarkdownSymbol("```\n", "\n```")}
            >Block</button
          >
        </div>
      </div>

      <div class="fullscreen-preview markdown-rendered">
        {@html interactivePreviewMarkdown}
      </div>
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
          on:click={() => (showCodeFullscreenModal = false)}>Cancel</button
        >
        <button class="btn primary" on:click={saveCodeFullscreen}
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
            on:scroll={handleEditorScroll}
            placeholder="Write your code snippet here..."
            spellcheck="false"
            autofocus
          ></textarea>
        </div>
      </div>
    </div>
  </div>
{/if}

<!-- Immersive Sites Sandbox Modal (At very top of stacking context) -->
{#if showSitesModal}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
  <div
    class="modal-backdrop"
    on:click|self={closeSitesModal}
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
        <button class="close-btn" on:click={closeSitesModal}>&times;</button>
      </div>

      <div
        class="sites-tabs"
        style="display: flex; gap: 4px; background: #121212; border-bottom: 1px solid #2e2e2e; padding: 6px 12px;"
      >
        <button
          class="emoji-tab-btn"
          class:active={sitesTab === "edit"}
          on:click={() => (sitesTab = "edit")}
        >
          🛠️ Code Editor
        </button>
        <button
          class="emoji-tab-btn"
          class:active={sitesTab === "preview"}
          on:click={() => (sitesTab = "preview")}
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
                on:input={handleSaveSites}
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
                on:input={handleSaveSites}
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
                on:input={handleSaveSites}
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
                <button class="btn-serve start" on:click={toggleSitesLive}>
                  <span class="btn-icon">▶</span> Start Local Server
                </button>
              {:else}
                <div class="active-server-buttons">
                  <button
                    class="btn primary"
                    on:click={() => {
                      if (sitesLocalUrl) window.open(sitesLocalUrl, "_blank");
                    }}
                  >
                    Open in Browser
                  </button>
                  <button
                    class="btn secondary"
                    on:click={() => {
                      if (sitesLocalUrl) {
                        navigator.clipboard.writeText(sitesLocalUrl);
                        alert("Copied to clipboard!");
                      }
                    }}
                  >
                    Copy Address
                  </button>
                  <button class="btn stop-btn" on:click={toggleSitesLive}>
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

<!-- Link Subpage Modal (Rendered Globally at Root Level) -->
{#if showLinkPageModal && linkingCard}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
  <div
    class="modal-backdrop"
    on:click|self={() => (showLinkPageModal = false)}
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
        <button class="close-btn" on:click={() => (showLinkPageModal = false)}
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
          <button class="clear-btn" on:click={() => (pageSearchQuery = "")}
            >&times;</button
          >
        {/if}
      </div>

      <!-- Linear/Raycast-style Command Trigger -->
      <button
        class="cmd-action-btn"
        on:click|stopPropagation={handleCreateNewPageAndLink}
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
                on:click={() => selectPageToLink(p)}
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
    font-size: 32px;
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
    font-size: 20px;
    flex-shrink: 0;
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
    display: grid;
    grid-template-columns: 1fr 1fr;
    overflow: hidden;
  }
  .fullscreen-workspace.single-pane {
    grid-template-columns: 1fr;
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
</style>
