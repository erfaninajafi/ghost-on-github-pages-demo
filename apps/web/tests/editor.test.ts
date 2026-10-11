import { test } from "node:test";
import assert from "node:assert/strict";
import {
  readEditorContent,
  readEditorDocument,
  contentPayload,
  insertContentBlock,
  removeContentBlock,
  renderContentBlocks,
  validArticleLink,
  type ContentBlock,
} from "../lib/editor";
import { safeHTML } from "../lib/seo";
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
test("mixed article blocks survive reopening, insertion, removal and format changes", () => {
  const image: ContentBlock = {
    type: "image",
    src: "/api/v1/media/owner/file.png",
    alt: 'عکس "اول"',
    width: 320,
  };
  const blocks = insertContentBlock(
    [{ type: "text", text: "قبل بعد" }],
    0,
    4,
    4,
    image,
  );
  const linked = insertContentBlock(blocks, 2, 0, 3, {
    type: "link",
    href: "/blog/article",
    text: "بعد",
  });
  for (const format of ["plain_text", "html"] as const) {
    const doc = readEditorDocument({
      content_html: "rendered",
      content_json: { version: 2, format, blocks: linked },
    });
    assert.deepEqual(doc, { content_format: format, blocks: linked });
    assert.deepEqual(contentPayload(format, doc.blocks), {
      content_format: format,
      content_blocks: linked,
    });
    const rendered = safeHTML(renderContentBlocks(format, doc.blocks));
    assert.match(rendered, /width="320"/);
    assert.match(rendered, /href="\/blog\/article"/);
    assert.ok(rendered.indexOf("قبل") < rendered.indexOf("<img"));
    assert.ok(rendered.indexOf("<img") < rendered.indexOf("<a"));
  }
  assert.deepEqual(removeContentBlock(blocks, 1), [
    { type: "text", text: "قبل بعد" },
  ]);
  assert.deepEqual(
    readEditorDocument({ content_html: "<p>old</p>", content_json: null }),
    { content_format: "html", blocks: [{ type: "text", text: "<p>old</p>" }] },
  );
});
test("plain blocks preserve literal HTML and link labels cannot execute markup", () => {
  const blocks: ContentBlock[] = [
    { type: "text", text: "<script>evil()</script>\n\nنص" },
    {
      type: "link",
      href: "https://example.com?q=1&b=2",
      text: "<img src=x onerror=evil()>",
    },
  ];
  const rendered = safeHTML(renderContentBlocks("plain_text", blocks));
  assert.ok(rendered.includes("&lt;script&gt;"));
  assert.ok(!rendered.includes("<script>"));
  assert.ok(!rendered.includes("<img"));
  assert.ok(rendered.includes("https://example.com?q=1&amp;b=2"));
  for (const bad of [
    "javascript:alert(1)",
    "data:text/html,x",
    "//example.com",
    "/\\example.com",
    "https://user:pass@example.com",
    "https://example.com\n",
  ])
    assert.equal(validArticleLink(bad), false);
  for (const good of [
    "/blog/آموزش",
    "https://example.com",
    "http://example.com/path",
  ])
    assert.equal(validArticleLink(good), true);
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
