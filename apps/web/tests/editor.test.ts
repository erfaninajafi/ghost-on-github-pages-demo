import { test } from "node:test";
import assert from "node:assert/strict";
import { readEditorContent, contentPayload } from "../lib/editor";
test("plain text posts reopen from source without escaped HTML", () => {
  const source = "سلام <b>متن واقعی</b>\n\nخط بعد";
  assert.deepEqual(
    readEditorContent({
      content_html: "<p>rendered</p>",
      content_json: { version: 1, format: "plain_text", content: source },
    }),
    { content_format: "plain_text", content: source },
  );
  assert.deepEqual(contentPayload("plain_text", source), {
    content_format: "plain_text",
    content_text: source,
  });
});
test("legacy and HTML posts use the sanitized HTML", () => {
  for (const content_json of [
    null,
    {},
    "invalid",
    { format: "html", content: "unsafe source" },
    { format: "plain_text", content: 123 },
  ]) {
    assert.deepEqual(
      readEditorContent({ content_html: "<p>safe</p>", content_json }),
      { content_format: "html", content: "<p>safe</p>" },
    );
  }
  assert.deepEqual(contentPayload("html", "<p>سلام</p>"), {
    content_format: "html",
    content_html: "<p>سلام</p>",
  });
});
