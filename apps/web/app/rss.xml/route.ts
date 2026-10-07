import { getPosts, siteURL } from "@/lib/api";
import { xml } from "@/lib/seo";
export const dynamic = "force-dynamic";
export async function GET() {
  const { items } = await getPosts(new URLSearchParams({ limit: "50" }));
  const content = `<?xml version="1.0" encoding="UTF-8"?><rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom"><channel><title>ExpertsAcademy | مجله دانش و تجربه</title><link>${xml(siteURL)}</link><description>یادداشت‌های متخصصان اکسپرتس آکادمی</description><language>fa-ir</language><atom:link href="${xml(siteURL)}/rss.xml" rel="self" type="application/rss+xml"/>${items
    .map((p) => {
      const url = `${siteURL}/blog/${encodeURIComponent(p.slug)}`;
      return `<item><title>${xml(p.title)}</title><link>${xml(url)}</link><guid isPermaLink="true">${xml(url)}</guid><description>${xml(p.excerpt)}</description><pubDate>${new Date(p.published_at!).toUTCString()}</pubDate></item>`;
    })
    .join("")}</channel></rss>`;
  return new Response(content, {
    headers: {
      "Content-Type": "application/rss+xml; charset=utf-8",
      "Cache-Control": "public, s-maxage=300, stale-while-revalidate=60",
    },
  });
}
