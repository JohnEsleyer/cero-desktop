<script context="module">
  import { default as TreeRender } from "./TreeRender.svelte";
  import { default as CardBlock } from "./CardBlock.svelte";
  import { default as RightSidebar } from "./RightSidebar.svelte";
</script>

<script>
  import { onMount } from "svelte";
  import { EventsOn } from "../wailsjs/runtime/runtime.js";
  import { marked } from "marked"; // Restored for immersive markdown preview
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

  let showEmojiPickerModal = false;
  let selectedCategory = "smileys";

  // PIN Entry Modal States
  let showPinModal = false;
  let pinInput = "";
  let pendingDeviceToConnect = null;
  let pinModalError = "";
  let isConnecting = false;

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

  // Rich Keyboard-Matching Emojis Catalog grouped by category
  const emojiCategories = [
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
        "🕍",
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
        "💣",
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
        "🗃️",
        "🗄️",
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
    return cat ? cat.emojis : [];
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
    UpdateCard(editingCard.id, editingCard.page_id, fullscreenEditContent)
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
    UpdateCard(editingCard.id, editingCard.page_id, combined)
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
    UpdateCard(editingCard.id, editingCard.page_id, combined)
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
</script>

<main class="app-layout" class:dragging>
  <!-- LEFT SIDEBAR -->
  <aside
    class="sidebar left-sidebar"
    style="width: {leftSidebarWidth}px; min-width: {leftSidebarWidth}px;"
  >
    <div class="sidebar-header">
      <div class="logo-section">
        <span class="logo-icon">📓</span><span class="logo-text">Cero</span>
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
          <div class="pulse"><span>📓</span></div>
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
          <span class="bc-current"
            >{selectedPage.emoji} {selectedPage.title || "Untitled"}</span
          >
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
          {editorEmoji || "📓"}
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

      <div class="card-column">
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
      <div class="modal-body emoji-picker-body">
        <div class="emoji-grid">
          {#each currentCategoryEmojis as emoji}
            <button
              class="emoji-select-btn"
              on:click={() => {
                editorEmoji = emoji;
                showEmojiPickerModal = false;
                savePageImmediate();
              }}
            >
              {emoji}
            </button>
          {/each}
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

<!-- Immersive Markdown Fullscreen Modal (At very top of stacking context) -->
{#if showMarkdownFullscreenModal}
  <div class="fullscreen-editor-overlay">
    <div class="fullscreen-header">
      <div class="header-info">
        <h3>📝 Immersive Markdown Editor</h3>
        <span class="sub-desc"
          >Split-workspace editing with instant rendering preview</span
        >
      </div>
      <div class="header-buttons">
        <button
          class="btn secondary"
          on:click={() => (showMarkdownFullscreenModal = false)}>Cancel</button
        >
        <button class="btn primary" on:click={saveMarkdownFullscreen}
          >Done</button
        >
      </div>
    </div>
    <div class="fullscreen-workspace">
      <textarea
        class="fullscreen-textarea"
        bind:value={fullscreenEditContent}
        placeholder="Start writing markdown content here..."
      ></textarea>
      <div class="fullscreen-preview markdown-rendered">
        {@html interactivePreviewMarkdown}
      </div>
    </div>
  </div>
{/if}

<!-- Immersive Code Fullscreen Modal (At very top of stacking context) -->
{#if showCodeFullscreenModal}
  <div class="fullscreen-editor-overlay">
    <div class="fullscreen-header">
      <div
        class="header-info"
        style="display: flex; align-items: center; gap: 16px;"
      >
        <h3 style="margin: 0;">💻 Immersive Code Editor</h3>
        <select
          bind:value={fullscreenCodeLang}
          style="background: #2a2a2a; color: #818cf8; border: 1px solid #3e3e3e; padding: 4px 8px; border-radius: 4px; font-size: 12px; font-weight: bold; outline: none; cursor: pointer;"
        >
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
      <div class="header-buttons">
        <button
          class="btn secondary"
          on:click={() => (showCodeFullscreenModal = false)}>Cancel</button
        >
        <button class="btn primary" on:click={saveCodeFullscreen}>Done</button>
      </div>
    </div>
    <div class="fullscreen-workspace single-pane">
      <textarea
        class="fullscreen-textarea monospace"
        bind:value={fullscreenCodeContent}
        placeholder="Write code snippet here..."
      ></textarea>
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
  }

  /* Force all modals and overlays to sit at the absolute top of Wails' rendering layers */
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
    z-index: 99999999 !important; /* High layer authority to clear WebKit scrollbars */
    cursor: default;
  }

  .fullscreen-editor-overlay {
    position: fixed;
    top: 0;
    left: 0;
    width: 100vw;
    height: 100vh;
    background: #121212;
    z-index: 100000000 !important; /* Highest layer authority */
    display: flex;
    flex-direction: column;
  }

  .app-layout {
    display: flex;
    width: 100vw;
    height: 100vh;
    overflow: hidden;
    background-color: #121212;
    color: #e2e8f0;
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto,
      sans-serif;
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
  }
  .left-sidebar {
    background: #1a1a1a;
    border-right: 1px solid #2e2e2e;
  }
  .right-sidebar {
    background: #1a1a1a;
    border-left: 1px solid #2e2e2e;
  }

  /* Drag handles */
  .drag-handle {
    width: 5px;
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
    background: rgba(129, 140, 248, 0.15);
  }
  .handle-line {
    width: 2px;
    height: 24px;
    border-radius: 1px;
    background: #3e3e3e;
    transition: background 0.15s;
  }
  .drag-handle:hover .handle-line,
  .drag-handle.active .handle-line {
    background: #818cf8;
  }

  .sidebar-header {
    padding: 14px 16px;
    border-bottom: 1px solid #2e2e2e;
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
  .logo-icon {
    font-size: 18px;
  }
  .logo-text {
    font-size: 14px;
    background: linear-gradient(135deg, #a5b4fc, #c084fc);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
  }

  .status-badge {
    display: flex;
    align-items: center;
    gap: 5px;
    padding: 2px 8px;
    border-radius: 10px;
    font-size: 9px;
    font-weight: 700;
    text-transform: uppercase;
  }
  .status-badge.disconnected {
    background: rgba(239, 68, 68, 0.1);
    color: #f87171;
  }
  .status-badge.connected {
    background: rgba(34, 197, 94, 0.1);
    color: #4ade80;
  }
  .status-badge.connecting {
    background: rgba(251, 191, 36, 0.1);
    color: #fbbf24;
  }
  .status-badge.reconnecting {
    background: rgba(251, 191, 36, 0.1);
    color: #fbbf24;
  }
  .dot {
    width: 5px;
    height: 5px;
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
    animation: pulse 1s infinite;
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
    padding: 12px 14px;
    border-bottom: 1px solid #2e2e2e;
  }
  .sidebar-section.grow {
    flex: 1;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
  }

  .section-label {
    font-size: 10px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.8px;
    color: #8e8e8e;
    margin-bottom: 8px;
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  /* Read-Only Active Workspace Display */
  .active-workspace-card {
    background: transparent;
    padding: 10px 14px;
    border-bottom: 1px solid #2e2e2e;
  }
  .active-ws-display {
    display: flex;
    align-items: center;
    gap: 8px;
    background: #121212;
    border: 1px solid #2e2e2e;
    border-radius: 6px;
    padding: 6px 10px;
    color: #cbd5e1;
    font-size: 12px;
    font-weight: 600;
  }
  .ws-icon {
    font-size: 14px;
  }
  .ws-name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .devices-box {
    background: #121212;
    border: 1px solid #2e2e2e;
    border-radius: 8px;
    padding: 10px;
  }
  .connected-info {
    display: flex;
    justify-content: space-between;
    font-size: 11px;
    margin-bottom: 6px;
  }
  .conn-label {
    color: #8e8e8e;
  }
  .conn-name {
    color: #4ade80;
    font-weight: 700;
  }
  .no-dev {
    font-size: 11px;
    color: #6c6c6c;
    text-align: center;
  }
  .dev-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 6px;
  }
  .dev-name {
    font-size: 12px;
    font-weight: 700;
    display: block;
  }
  .dev-ip {
    font-size: 10px;
    color: #6c6c6c;
    font-family: monospace;
  }

  .tree-scroll {
    flex: 1;
    overflow-y: auto;
  }
  .tree-empty {
    font-size: 12px;
    color: #6c6c6c;
    padding: 8px 0;
  }

  .icon-btn {
    background: transparent;
    border: none;
    color: #8e8e8e;
    cursor: pointer;
    font-size: 16px;
    padding: 0 4px;
  }
  .icon-btn:hover {
    color: white;
  }

  /* Manual Connect Form UI styling */
  .manual-connect-card {
    background: #151515;
    border-radius: 8px;
    border: 1px solid #252525;
    margin: 12px 14px;
    padding: 12px;
  }
  .manual-form {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }
  .form-group {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .form-group label {
    font-size: 9px;
    font-weight: 700;
    color: #64748b;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }
  .form-group input {
    background: #1e1e1e !important;
    border: 1px solid #2e2e2e !important;
    border-radius: 6px !important;
    color: #e2e8f0 !important;
    font-size: 12px !important;
    padding: 6px 10px !important;
    outline: none !important;
    margin-bottom: 0 !important;
    transition: border-color 0.15s;
  }
  .form-group input:focus {
    border-color: #818cf8 !important;
  }
  .pin-input {
    letter-spacing: 6px !important;
    text-align: center !important;
    font-weight: 700 !important;
    font-size: 14px !important;
  }
  .connect-btn {
    width: 100%;
    background: #818cf8;
    color: white;
    border: none;
    border-radius: 6px;
    padding: 8px;
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    transition: background 0.15s;
  }
  .connect-btn:hover {
    background: #6366f1;
  }
  .connection-error-box {
    display: flex;
    align-items: center;
    gap: 6px;
    background: rgba(239, 68, 68, 0.1);
    border: 1px solid rgba(239, 68, 68, 0.2);
    border-radius: 6px;
    padding: 6px 10px;
  }
  .err-icon {
    font-size: 12px;
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
    font-size: 11px;
    font-weight: 600;
    padding: 5px 10px;
    cursor: pointer;
    border: none;
    background: #2e2e2e;
    color: #cbd5e1;
    transition: all 0.15s;
  }
  .btn-sm:hover {
    background: #3e3e3e;
  }
  .btn-sm.primary {
    background: #818cf8;
    color: white;
  }
  .btn-sm.primary:hover {
    background: #6366f1;
  }
  .btn-sm.danger {
    background: rgba(239, 68, 68, 0.1);
    color: #f87171;
  }
  .btn-sm.full {
    width: 100%;
  }

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
  .btn.stop-btn {
    background: rgba(239, 68, 68, 0.04);
    border: 1px solid rgba(239, 68, 68, 0.2);
    color: #f87171;
  }
  .btn.stop-btn:hover {
    background: rgba(239, 68, 68, 0.1);
    border-color: rgba(239, 68, 68, 0.4);
    color: #fca5a5;
    transform: translateY(-0.5px);
  }

  /* Editor */
  .editor-workspace {
    flex: 1;
    display: flex;
    flex-direction: column;
    height: 100%;
    overflow: hidden;
    background: #121212;
  }

  .welcome {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 40px;
  }
  .welcome-card {
    max-width: 500px;
    text-align: center;
    background: #1a1a1a;
    border: 1px solid #2e2e2e;
    border-radius: 20px;
    padding: 40px;
  }
  .pulse {
    width: 64px;
    height: 64px;
    border-radius: 50%;
    background: rgba(165, 180, 252, 0.05);
    border: 1px solid rgba(165, 180, 252, 0.15);
    display: flex;
    align-items: center;
    justify-content: center;
    margin: 0 auto 16px;
    font-size: 28px;
  }
  .welcome-card h1 {
    font-size: 22px;
    font-weight: 800;
    margin: 0 0 8px;
  }
  .welcome-card p {
    font-size: 13px;
    color: #8e8e8e;
    margin: 0 0 24px;
  }
  .steps {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 12px;
    text-align: left;
  }
  .step {
    background: #121212;
    border: 1px solid #2e2e2e;
    border-radius: 10px;
    padding: 14px;
  }
  .step .num {
    width: 20px;
    height: 20px;
    border-radius: 50%;
    background: #818cf8;
    color: white;
    font-weight: 700;
    font-size: 10px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    margin-bottom: 6px;
  }
  .step p {
    font-size: 11px;
    color: #d1d5db;
    margin: 0;
    line-height: 1.4;
  }

  .empty-page {
    text-align: center;
  }
  .big-icon {
    font-size: 48px;
    display: block;
    margin-bottom: 12px;
  }
  .empty-page h2 {
    font-size: 18px;
    margin: 0 0 6px;
  }
  .empty-page p {
    font-size: 13px;
    color: #8e8e8e;
    margin: 0 0 20px;
  }

  .editor-header {
    background: #1a1a1a;
    border-bottom: 1px solid #2e2e2e;
    padding: 10px 24px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    flex-shrink: 0;
  }
  .breadcrumbs {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 12px;
  }
  .back-btn {
    background: transparent;
    border: none;
    color: #8e8e8e;
    cursor: pointer;
    font-size: 11px;
    font-weight: 600;
    padding: 3px 6px;
    border-radius: 4px;
  }
  .back-btn:hover {
    background: #2e2e2e;
    color: white;
  }
  .sep {
    color: #444;
  }
  .bc-root {
    color: #8e8e8e;
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
    gap: 10px;
    padding: 12px 24px 0;
    flex-shrink: 0;
  }
  .emoji-input-btn {
    width: 38px;
    height: 38px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 20px;
    background: #1e1e1e;
    border: 1px solid #2e2e2e;
    border-radius: 8px;
    color: #e2e8f0;
    cursor: pointer;
    flex-shrink: 0;
    transition: all 0.12s;
  }
  .emoji-input-btn:hover {
    border-color: #818cf8;
    background: rgba(129, 140, 248, 0.05);
  }
  .title-input {
    flex: 1;
    background: transparent;
    border: none;
    outline: none;
    font-size: 22px;
    font-weight: 700;
    color: #e2e8f0;
    padding: 6px 0;
    border-bottom: 2px solid transparent;
  }
  .title-input:focus {
    border-bottom-color: #818cf8;
  }
  .title-input::placeholder {
    color: #4a4a4a;
  }

  .card-column {
    flex: 1;
    overflow-y: auto;
    padding: 16px 24px;
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .empty-cards {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 60px 40px;
    text-align: center;
    background: #1a1a1a;
    border: 1px dashed #2e2e2e;
    border-radius: 12px;
    margin: 16px 0;
  }
  .empty-cards .empty-icon {
    font-size: 48px;
    margin-bottom: 12px;
  }
  .empty-cards h3 {
    font-size: 16px;
    font-weight: 700;
    margin: 0 0 6px;
  }
  .empty-cards p {
    font-size: 13px;
    color: #8e8e8e;
    margin: 0 0 20px;
  }
  .empty-actions {
    display: flex;
    gap: 10px;
  }

  .add-card-btn {
    background: transparent;
    border: 1px dashed #3e3e3e;
    border-radius: 8px;
    padding: 10px;
    color: #64748b;
    cursor: pointer;
    font-size: 12px;
    transition: all 0.15s;
  }
  .add-card-btn:hover {
    border-color: #818cf8;
    color: #818cf8;
  }

  .card-slot {
    transition: opacity 0.15s;
  }
  .card-slot[draggable="true"] {
    cursor: grab;
  }
  .card-slot[draggable="true"]:active {
    cursor: grabbing;
  }

  .insert-slot {
    height: 4px;
    margin: 0 24px;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.15s;
  }
  .insert-slot:hover {
    height: 24px;
    background: rgba(129, 140, 248, 0.05);
  }
  .insert-btn {
    opacity: 0;
    background: #2a2a2a;
    border: 1px solid #3e3e3e;
    color: #64748b;
    font-size: 12px;
    width: 20px;
    height: 20px;
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

  /* Notion-style Block Selector Modal Styles */
  .block-selector-modal {
    width: 440px;
    max-width: 90%;
  }
  .block-options-grid {
    display: flex;
    flex-direction: column;
    gap: 10px;
    padding: 16px !important;
  }
  .block-option-row {
    display: flex;
    align-items: center;
    gap: 14px;
    background: #1e1e1e;
    border: 1px solid #2e2e2e;
    border-radius: 8px;
    padding: 12px 16px;
    text-align: left;
    cursor: pointer;
    color: #e2e8f0;
    transition: all 0.15s ease;
    width: 100%;
  }
  .block-option-row:hover {
    background: rgba(129, 140, 248, 0.08);
    border-color: #818cf8;
    transform: translateY(-1px);
  }
  .block-icon {
    font-size: 24px;
    flex-shrink: 0;
  }
  .block-desc {
    display: flex;
    flex-direction: column;
    gap: 3px;
  }
  .block-title {
    font-size: 13px;
    font-weight: 700;
    color: #e2e8f0;
  }
  .block-subtitle {
    font-size: 11px;
    color: #64748b;
    line-height: 1.3;
  }

  /* Emoji Picker Modal Styles */
  .emoji-picker-modal {
    width: 480px;
    max-width: 95%;
    height: 450px;
    max-height: 85vh;
  }
  .emoji-picker-tabs {
    display: flex;
    gap: 4px;
    background: #121212;
    border-bottom: 1px solid #2e2e2e;
    padding: 6px 12px;
    overflow-x: auto;
    scrollbar-width: none;
  }
  .emoji-picker-tabs::-webkit-scrollbar {
    display: none;
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
  .emoji-picker-body {
    padding: 12px !important;
    overflow-y: auto !important;
  }
  .emoji-grid {
    display: grid;
    grid-template-columns: repeat(8, 1fr);
    gap: 6px;
  }
  .emoji-select-btn {
    background: transparent;
    border: none;
    font-size: 24px;
    aspect-ratio: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 6px;
    cursor: pointer;
    transition: background 0.12s;
  }
  .emoji-select-btn:hover {
    background: rgba(255, 255, 255, 0.05);
  }

  /* PIN Modal Specific Styles */
  .pin-entry-modal {
    width: 380px;
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
    gap: 12px;
    background: #121212;
    border: 1px solid #2e2e2e;
    border-radius: 8px;
    padding: 10px 14px;
  }
  .device-icon {
    font-size: 24px;
  }
  .device-details {
    display: flex;
    flex-direction: column;
  }
  .pin-instructions {
    font-size: 12px;
    color: #8e8e8e;
    line-height: 1.5;
    margin: 0;
  }
  .pin-input-container {
    display: flex;
    justify-content: center;
    margin: 8px 0;
  }
  .pin-display-input {
    background: #121212 !important;
    border: 2px solid #2e2e2e !important;
    border-radius: 8px !important;
    color: #818cf8 !important;
    font-size: 32px !important;
    font-weight: 700 !important;
    letter-spacing: 12px !important;
    text-align: center !important;
    width: 180px !important;
    padding: 8px 0 8px 12px !important; /* Offset end letter spacing */
    outline: none !important;
    font-family: monospace;
    transition:
      border-color 0.15s,
      box-shadow 0.15s;
    margin-bottom: 0 !important;
  }
  .pin-display-input:focus {
    border-color: #818cf8 !important;
    box-shadow: 0 0 0 2px rgba(129, 140, 248, 0.2);
  }
  .modal-error-box {
    display: flex;
    align-items: center;
    gap: 8px;
    background: rgba(239, 68, 68, 0.1);
    border: 1px solid rgba(239, 68, 68, 0.2);
    border-radius: 8px;
    padding: 8px 12px;
  }
  .modal-footer {
    display: flex;
    justify-content: flex-end;
    gap: 10px;
    padding: 14px 16px;
    border-top: 1px solid #2e2e2e;
    background: #151515;
  }
  .spinner {
    display: inline-block;
    width: 12px;
    height: 12px;
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
    background: #151515;
    border: none;
    border-right: 1px solid #2e2e2e;
    color: #e2e8f0;
    font-size: 14px;
    font-family: inherit;
    line-height: 1.6;
    padding: 24px;
    resize: none;
    outline: none;
  }
  .fullscreen-textarea.monospace {
    font-family: "Fira Code", monospace;
    font-size: 13px;
  }
  .fullscreen-preview {
    padding: 24px;
    overflow-y: auto;
    background: #121212;
  }

  .server-workspace-pane {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 36px 16px;
    gap: 20px;
  }
  .server-status-card {
    background: rgba(255, 255, 255, 0.02);
    border: 1px solid rgba(255, 255, 255, 0.05);
    border-radius: 12px;
    padding: 24px;
    width: 100%;
    max-width: 480px;
    text-align: center;
    transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  }
  .server-status-card.active {
    background: rgba(16, 185, 129, 0.02);
    border-color: rgba(16, 185, 129, 0.15);
  }
  .server-status-indicator {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 10px;
    margin-bottom: 12px;
  }
  .pulse-indicator {
    width: 10px;
    height: 10px;
    border-radius: 50%;
    background: #ef4444;
    box-shadow: 0 0 8px rgba(239, 68, 68, 0.2);
    transition: all 0.3s ease;
  }
  .pulse-indicator.active {
    background: #10b981;
    box-shadow: 0 0 12px rgba(16, 185, 129, 0.6);
    animation: pulse-active 2s infinite;
  }
  .status-text {
    font-size: 14px;
    font-weight: 600;
    color: #94a3b8;
  }
  .server-status-card.active .status-text {
    color: #cbd5e1;
  }
  .server-address-box {
    display: flex;
    flex-direction: column;
    gap: 6px;
    background: #121212;
    border: 1px solid rgba(255, 255, 255, 0.04);
    padding: 12px;
    border-radius: 8px;
  }
  .address-label {
    font-size: 10px;
    font-weight: 700;
    color: #64748b;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }
  .address-value {
    font-size: 14px;
    color: #818cf8;
    font-family: monospace;
    font-weight: 600;
    word-break: break-all;
  }
  .server-offline-hint {
    font-size: 12px;
    color: #64748b;
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
    gap: 12px;
    width: 100%;
    max-width: 480px;
  }
  .active-server-buttons .btn {
    flex: 1;
  }
  .btn-serve.start {
    background: rgba(16, 185, 129, 0.06);
    border: 1px solid rgba(16, 185, 129, 0.2);
    color: #34d399;
    font-weight: 600;
    font-size: 13px;
    padding: 12px 24px;
    border-radius: 8px;
    cursor: pointer;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
    box-shadow: 0 4px 12px rgba(16, 185, 129, 0.05);
  }
  .btn-serve.start:hover {
    background: rgba(16, 185, 129, 0.12);
    border-color: rgba(16, 185, 129, 0.4);
    color: #6ee7b7;
    box-shadow: 0 6px 16px rgba(16, 185, 129, 0.15);
    transform: translateY(-0.5px);
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
</style>
