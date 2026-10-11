import { timingSafeEqual } from "node:crypto";
import { revalidatePath, revalidateTag } from "next/cache";
import { articlePath, articleSlug } from "@/lib/article-path";
export const runtime = "nodejs";
export async function POST(request: Request) {
  const expected = process.env.REVALIDATION_SECRET;
  if (!expected || expected.length < 32)
    return Response.json({ error: "Unavailable" }, { status: 503 });
  const provided =
    request.headers.get("Authorization")?.replace(/^Bearer /, "") || "";
  const a = Buffer.from(provided),
    b = Buffer.from(expected);
  if (a.length !== b.length || !timingSafeEqual(a, b))
    return Response.json({ error: "Unauthorized" }, { status: 401 });
  // The internal webhook accepts only bounded, canonical slugs.
  if (Number(request.headers.get("Content-Length") || 0) > 4096)
    return Response.json({ error: "Too large" }, { status: 413 });
  const raw = await request.text();
  if (Buffer.byteLength(raw) > 4096)
    return Response.json({ error: "Too large" }, { status: 413 });
  let body: unknown;
  try {
    body = JSON.parse(raw);
  } catch {
    return Response.json({ error: "Invalid JSON" }, { status: 400 });
  }
  if (
    typeof body !== "object" ||
    body === null ||
    !("slugs" in body) ||
    !Array.isArray(body.slugs) ||
    body.slugs.length > 2 ||
    !body.slugs.every((s) => typeof s === "string" && articleSlug(s) === s)
  )
    return Response.json({ error: "Invalid slugs" }, { status: 400 });
  revalidateTag("posts");
  for (const slug of body.slugs as string[]) {
    revalidateTag(`post:${slug}`);
    revalidatePath(articlePath(slug));
    if (articlePath(slug) !== `/blog/${slug}`) revalidatePath(`/blog/${slug}`);
  }
  revalidatePath("/");
  revalidatePath("/sitemap.xml");
  revalidatePath("/rss.xml");
  return Response.json({ revalidated: true });
}
