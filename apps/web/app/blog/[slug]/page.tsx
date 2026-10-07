import { cache } from "react";
import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import {
  APIError,
  getPost,
  getPopular,
  date,
  number,
  siteURL,
} from "@/lib/api";
import { jsonLD, safeHTML } from "@/lib/seo";
import { Trending } from "@/components/post-card";
import { Comments } from "@/components/comments";
import { ViewCounter } from "@/components/view-counter";
export const revalidate = 60;
export async function generateStaticParams() {
  return [];
}
const load = cache(async (slug: string) => {
  try {
    return await getPost(slug);
  } catch (e) {
    if (e instanceof APIError && e.status === 404) notFound();
    throw e;
  }
});
export async function generateMetadata({
  params,
}: {
  params: Promise<{ slug: string }>;
}): Promise<Metadata> {
  const { slug } = await params;
  const p = await load(slug);
  const url = `${siteURL}/blog/${encodeURIComponent(p.slug)}`;
  return {
    title: p.title,
    description: p.excerpt,
    alternates: { canonical: url },
    openGraph: {
      title: p.title,
      description: p.excerpt,
      type: "article",
      url,
      publishedTime: p.published_at || undefined,
      modifiedTime: p.updated_at,
      images: [{ url: "/og.png", width: 1200, height: 630 }],
    },
    twitter: {
      card: "summary_large_image",
      title: p.title,
      description: p.excerpt,
      images: ["/og.png"],
    },
  };
}
export default async function Article({
  params,
}: {
  params: Promise<{ slug: string }>;
}) {
  const { slug } = await params;
  const [post, popular] = await Promise.all([load(slug), getPopular()]);
  const structured = {
    "@context": "https://schema.org",
    "@type": "Article",
    headline: post.title,
    description: post.excerpt,
    datePublished: post.published_at,
    dateModified: post.updated_at,
    inLanguage: "fa-IR",
    mainEntityOfPage: `${siteURL}/blog/${encodeURIComponent(post.slug)}`,
    author: { "@type": "Organization", name: "ExpertsAcademy", url: siteURL },
    publisher: {
      "@type": "Organization",
      name: "ExpertsAcademy",
      url: siteURL,
    },
    image: `${siteURL}/og.png`,
  };
  return (
    <div className="container article-layout">
      <article>
        <nav className="breadcrumbs">
          <Link href="/">مجله</Link>
          <span>/</span>
          <span>مقاله</span>
        </nav>
        <header className="article-header">
          <p className="eyebrow">یادداشت‌های ExpertsAcademy</p>
          <h1>{post.title}</h1>
          <p className="article-excerpt">{post.excerpt}</p>
          <div className="article-meta">
            <span className="avatar">E</span>
            <span>تحریریه ExpertsAcademy</span>
            <time dateTime={post.published_at || undefined}>
              {date(post.published_at)}
            </time>
            <span>{number(post.reading_time_minutes)} دقیقه مطالعه</span>
          </div>
        </header>
        <div
          className="prose-content"
          dangerouslySetInnerHTML={{ __html: safeHTML(post.content_html) }}
        />
        <div className="tag-list">
          {post.tags.map((t) => (
            <Link key={t.id} href={`/?tag=${encodeURIComponent(t.slug)}`}>
              #{t.name}
            </Link>
          ))}
        </div>
        <Comments postID={post.id} />
      </article>
      <aside>
        <Trending posts={popular} />
        <div className="aside-note">
          <h3>دانش، با اشتراک‌گذاری رشد می‌کند.</h3>
          <p>دیدگاه و تجربهٔ خود را در بخش نظرات به این مقاله اضافه کنید.</p>
          <a href="#comments">پیوستن به گفت‌وگو ←</a>
        </div>
      </aside>
      <ViewCounter slug={slug} />
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: jsonLD(structured) }}
      />
    </div>
  );
}
