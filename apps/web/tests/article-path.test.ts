import assert from "node:assert/strict";
import { articlePath, articleSlug } from "../lib/article-path";
const slug = "اولین-تست-نویسنده";
assert.equal(articleSlug(encodeURIComponent(slug)), slug);
assert.equal(articleSlug(slug), slug);
assert.equal(articleSlug("hello-world-123"), "hello-world-123");
assert.equal(
  articlePath(articleSlug(encodeURIComponent(slug))!),
  `/blog/${encodeURIComponent(slug)}`,
);
for (const invalid of [
  "%E0%A4%A",
  "../admin",
  "with space",
  "%2Fadmin",
  encodeURIComponent(encodeURIComponent(slug)),
])
  assert.equal(articleSlug(invalid), null);
