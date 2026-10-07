import type { MetadataRoute } from "next";
import { getPosts, siteURL } from "@/lib/api";
export const dynamic = "force-dynamic";
export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  const entries: MetadataRoute.Sitemap = [
    { url: siteURL, changeFrequency: "daily", priority: 1 },
  ];
  let page = 1;
  for (;;) {
    const data = await getPosts(
      new URLSearchParams({ page: String(page), limit: "50" }),
    );
    entries.push(
      ...data.items.map((p) => ({
        url: `${siteURL}/blog/${encodeURIComponent(p.slug)}`,
        lastModified: p.updated_at,
        changeFrequency: "weekly" as const,
        priority: 0.7,
      })),
    );
    if (page * 50 >= data.total) break;
    page++;
  }
  return entries;
}
