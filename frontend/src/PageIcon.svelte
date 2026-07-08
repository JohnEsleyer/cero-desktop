<script>
  import { onMount } from "svelte";
  import { GetImage } from "../wailsjs/go/main/App.js";

  export let emoji = "📝";
  export let size = 16;

  let isEmoji = true;
  let imageSrc = "";

  function checkIconType(text) {
    if (!text) {
      isEmoji = true;
      return;
    }
    const trimmed = text.trim();
    if (trimmed.startsWith("http://") || trimmed.startsWith("https://")) {
      isEmoji = false;
      imageSrc = trimmed;
      return;
    }
    if (trimmed.startsWith("/") || trimmed.startsWith("file://") || trimmed.startsWith("data:")) {
      isEmoji = false;
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
      isEmoji = false;
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
    isEmoji = true;
  }

  $: checkIconType(emoji);
</script>

{#if isEmoji}
  <span style="font-size: {size}px; line-height: 1; display: inline-flex; align-items: center; justify-content: center;">
    {emoji || "📝"}
  </span>
{:else}
  <img
    src={imageSrc}
    alt="Icon"
    style="width: {size}px; height: {size}px; min-width: {size}px; min-height: {size}px; object-fit: cover; border-radius: inherit; display: inline-block; vertical-align: middle;"
    on:error={(e) => {
      e.target.style.display = "none";
    }}
  />
{/if}
