<script>
  import { GetImage } from "../wailsjs/go/main/App.js";

  let { emoji = "", size = 16 } = $props();

  let isImage = $state(false);
  let imageSrc = $state("");

  function checkIconType(text) {
    if (!text) {
      isImage = false;
      return;
    }
    const trimmed = text.trim();
    if (
      trimmed.startsWith("http://") ||
      trimmed.startsWith("https://") ||
      trimmed.startsWith("/") ||
      trimmed.startsWith("file://") ||
      trimmed.startsWith("data:")
    ) {
      isImage = true;
      imageSrc = trimmed;
      return;
    }
    if (trimmed.includes(".") && (
      trimmed.endsWith(".png") ||
      trimmed.endsWith(".jpg") ||
      trimmed.endsWith(".jpeg") ||
      trimmed.endsWith(".gif") ||
      trimmed.endsWith(".webp") ||
      trimmed.endsWith(".svg")
    )) {
      isImage = true;
      if (typeof GetImage === "function") {
        GetImage(trimmed)
          .then((data) => {
            imageSrc = data;
          })
          .catch((err) => {
            console.error("Failed to load page icon image:", err);
          });
      }
      return;
    }
    isImage = false;
  }

  $effect(() => {
    checkIconType(emoji);
  });

  const fallbackSvg = `<path d="M15 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7Z"/><path d="M14 2v4a2 2 0 0 0 2 2h4"/><path d="M10 9H8"/><path d="M16 13H8"/><path d="M16 17H8"/>`;
</script>

{#if isImage}
  <img
    src={imageSrc}
    alt="Icon"
    style="width: {size}px; height: {size}px; min-width: {size}px; min-height: {size}px; object-fit: cover; border-radius: 4px; display: inline-block; vertical-align: middle;"
    onerror={(e) => {
      e.target.style.display = "none";
    }}
  />
{:else}
  <svg
    xmlns="http://www.w3.org/2000/svg"
    width={size}
    height={size}
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    stroke-width="2"
    stroke-linecap="round"
    stroke-linejoin="round"
    class="lucide-icon"
    style="display: inline-block; vertical-align: middle; color: inherit;"
  >
    {@html fallbackSvg}
  </svg>
{/if}
