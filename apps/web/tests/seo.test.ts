import { test } from "node:test";
import assert from "node:assert/strict";
import { jsonLD, safeHTML, xml } from "../lib/seo";
test("article rendering removes executable markup", () => {
  const output = safeHTML(
    '<p onclick="alert(1)">سلام</p><script>alert(1)</script><a href="javascript:alert(1)">x</a><iframe src="https://evil.example"></iframe>',
  );
  assert.ok(output.includes("سلام"));
  assert.ok(!/onclick|<script|javascript:|iframe/.test(output));
});
test("JSON-LD cannot escape its script element", () => {
  const output = jsonLD({ headline: "</script><script>alert(1)</script>" });
  assert.ok(!output.includes("<"));
  assert.deepEqual(JSON.parse(output), {
    headline: "</script><script>alert(1)</script>",
  });
});
test("RSS escapes XML metacharacters while preserving Persian", () => {
  assert.equal(xml('سلام < & > "'), "سلام &lt; &amp; &gt; &quot;");
});
