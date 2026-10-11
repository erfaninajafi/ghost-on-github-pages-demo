import type { Post } from "./types";
export type ContentFormat = "plain_text" | "html";
export interface EditorContent {
  content_format: ContentFormat;
  content: string;
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
export function contentPayload(format: ContentFormat, content: string) {
  return {
    content_format: format,
    ...(format === "plain_text"
      ? { content_text: content }
      : { content_html: content }),
  };
}
