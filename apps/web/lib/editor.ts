import type { Post } from "./types";
export type ContentFormat = "plain_text" | "html";
export interface EditorContent {
  content_format: ContentFormat;
  content: string;
}
export type ContentBlock =
  | { type: "text"; text: string }
  | { type: "image"; src: string; alt: string; width: number }
  | { type: "link"; href: string; text: string };
export const textBlock = (text = ""): ContentBlock => ({ type: "text", text });

export function validArticleLink(value: string): boolean {
  if (
    !value ||
    value.length > 2048 ||
    /[\\\r\n\t]/.test(value) ||
    value.trim() !== value
  )
    return false;
  if (value.startsWith("/")) return !value.startsWith("//");
  try {
    const url = new URL(value);
    return (
      (url.protocol === "http:" || url.protocol === "https:") &&
      !!url.hostname &&
      !url.username &&
      !url.password
    );
  } catch {
    return false;
  }
}
function isBlock(value: unknown): value is ContentBlock {
  if (typeof value !== "object" || value === null || !("type" in value))
    return false;
  if (value.type === "text")
    return "text" in value && typeof value.text === "string";
  if (value.type === "link")
    return (
      "text" in value &&
      typeof value.text === "string" &&
      "href" in value &&
      typeof value.href === "string" &&
      validArticleLink(value.href)
    );
  return (
    value.type === "image" &&
    "src" in value &&
    typeof value.src === "string" &&
    value.src.startsWith("/api/v1/media/") &&
    "alt" in value &&
    typeof value.alt === "string" &&
    "width" in value &&
    typeof value.width === "number" &&
    Number.isInteger(value.width) &&
    value.width >= 80 &&
    value.width <= 2000
  );
}
export function readEditorDocument(
  post: Pick<Post, "content_html" | "content_json">,
): { content_format: ContentFormat; blocks: ContentBlock[] } {
  const document = post.content_json;
  if (
    typeof document === "object" &&
    document !== null &&
    "version" in document &&
    document.version === 2 &&
    "format" in document &&
    (document.format === "plain_text" || document.format === "html") &&
    "blocks" in document &&
    Array.isArray(document.blocks) &&
    document.blocks.length > 0 &&
    document.blocks.length <= 200 &&
    document.blocks.every(isBlock)
  ) {
    return { content_format: document.format, blocks: document.blocks };
  }
  const legacy = readEditorContent(post);
  return {
    content_format: legacy.content_format,
    blocks: [textBlock(legacy.content)],
  };
}
export function insertContentBlock(
  blocks: ContentBlock[],
  index: number,
  start: number,
  end: number,
  block: ContentBlock,
): ContentBlock[] {
  const current = blocks[index];
  if (current?.type !== "text") return [...blocks, block, textBlock()];
  return [
    ...blocks.slice(0, index),
    textBlock(current.text.slice(0, start)),
    block,
    textBlock(current.text.slice(end)),
    ...blocks.slice(index + 1),
  ];
}
export function removeContentBlock(
  blocks: ContentBlock[],
  index: number,
): ContentBlock[] {
  const result: ContentBlock[] = [];
  for (const block of blocks.filter((_, i) => i !== index)) {
    const previous = result[result.length - 1];
    if (previous?.type === "text" && block.type === "text")
      previous.text += block.text;
    else result.push({ ...block });
  }
  return result.length ? result : [textBlock()];
}
export function escapeHTML(value: string): string {
  return value.replace(
    /[&<>"']/g,
    (char) =>
      ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[
        char
      ]!,
  );
}
// Preview callers sanitize the rendered HTML before placing it in the DOM.
export function renderContentBlocks(
  format: ContentFormat,
  blocks: ContentBlock[],
): string {
  const source = blocks
    .map((block) => {
      if (block.type === "image")
        return `<img src="${escapeHTML(block.src)}" alt="${escapeHTML(block.alt)}" width="${block.width}">`;
      if (block.type === "link")
        return `<a href="${escapeHTML(block.href)}">${escapeHTML(block.text)}</a>`;
      const text = block.text.replace(/\r\n?/g, "\n");
      return format === "html"
        ? text
        : escapeHTML(text).replace(/\n\n/g, "</p><p>").replace(/\n/g, "<br>");
    })
    .join("");
  return format === "plain_text" ? `<p>${source}</p>` : source;
}
// Legacy posts do not have format metadata and continue to open as HTML.
export function readEditorContent(
  post: Pick<Post, "content_html" | "content_json">,
): EditorContent {
  const document = post.content_json;
  if (
    typeof document === "object" &&
    document !== null &&
    "format" in document &&
    document.format === "plain_text" &&
    "content" in document &&
    typeof document.content === "string"
  ) {
    return { content_format: "plain_text", content: document.content };
  }
  return { content_format: "html", content: post.content_html };
}
export function contentPayload(
  format: ContentFormat,
  content: string | ContentBlock[],
) {
  if (Array.isArray(content))
    return { content_format: format, content_blocks: content };
  return {
    content_format: format,
    ...(format === "plain_text"
      ? { content_text: content }
      : { content_html: content }),
  };
}
