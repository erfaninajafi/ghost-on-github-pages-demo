import type { MetadataRoute } from "next";
import { siteURL } from "@/lib/api";
export default function robots(): MetadataRoute.Robots {
  return {
    rules: { userAgent: "*", allow: "/", disallow: ["/studio", "/api/"] },
    sitemap: `${siteURL}/sitemap.xml`,
    host: siteURL,
  };
}
