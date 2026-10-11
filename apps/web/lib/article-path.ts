// Next's route params can contain escaped Unicode. Decode once before constructing
// an API URL; otherwise Persian slugs become double-encoded and return 404.
export function articleSlug(value: string): string | null {
  try {
    const slug = decodeURIComponent(value);
    return /^[\p{L}\p{N}-]{1,200}$/u.test(slug) ? slug : null;
  } catch {
    return null;
  }
}
export const articlePath = (slug: string) =>
  `/blog/${encodeURIComponent(slug)}`;
