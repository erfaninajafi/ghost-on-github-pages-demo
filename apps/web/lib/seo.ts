import sanitizeHtml from "sanitize-html";
export function safeHTML(html: string) {
  return sanitizeHtml(html, {
    allowedTags: [...sanitizeHtml.defaults.allowedTags, "img", "h1", "h2"],
    allowedAttributes: {
      a: ["href", "title", "rel"],
      img: ["src", "alt", "width", "height", "loading"],
      code: ["class"],
    },
    allowedSchemes: ["http", "https", "mailto"],
    transformTags: {
      a: sanitizeHtml.simpleTransform("a", {
        rel: "nofollow noopener noreferrer",
      }),
      img: sanitizeHtml.simpleTransform("img", { loading: "lazy" }),
    },
  });
}
export function jsonLD(value: unknown) {
  return JSON.stringify(value).replace(/</g, "\\u003c");
}
export function xml(value: string) {
  return value.replace(
    /[<>&"']/g,
    (c) =>
      ({
        "<": "&lt;",
        ">": "&gt;",
        "&": "&amp;",
        '"': "&quot;",
        "'": "&apos;",
      })[c]!,
  );
}
