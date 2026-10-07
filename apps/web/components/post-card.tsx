import Link from "next/link";
import { date, number } from "@/lib/api";
import type { Post } from "@/lib/types";
export function PostCard({ post, index = 0 }: { post: Post; index?: number }) {
  return (
    <article className="post-card">
      <Link
        href={`/blog/${post.slug}`}
        className={`card-art art-${index % 3}`}
        aria-hidden="true"
        tabIndex={-1}
      >
        <span className="art-grid" />
        <span className="art-word">
          {["دانش", "تجربه", "آینده"][index % 3]}
        </span>
        <span className="art-label" dir="ltr">
          EXPERTS / INSIGHTS
        </span>
      </Link>
      <div className="card-body">
        <p className="eyebrow">
          دیدگاه متخصصان <span>•</span> {number(post.reading_time_minutes)}{" "}
          دقیقه
        </p>
        <h2>
          <Link href={`/blog/${post.slug}`}>{post.title}</Link>
        </h2>
        <p className="muted line-clamp-3">{post.excerpt}</p>
        <div className="card-footer">
          <span>تحریریه ExpertsAcademy</span>
          <time dateTime={post.published_at || undefined}>
            {date(post.published_at)}
          </time>
        </div>
      </div>
    </article>
  );
}
export function Trending({ posts }: { posts: Post[] }) {
  return (
    <section className="trending">
      <div className="section-title">
        <h2>بیشتر خوانده‌شده‌ها</h2>
        <span className="orange">↗</span>
      </div>
      {posts.length ? (
        posts.map((post, i) => (
          <Link
            className="trending-item"
            key={post.id}
            href={`/blog/${post.slug}`}
          >
            <span className="rank">{number(i + 1).padStart(2, "۰")}</span>
            <div>
              <h3>{post.title}</h3>
              <small>
                {number(post.views_count)} بازدید ·{" "}
                {number(post.reading_time_minutes)} دقیقه مطالعه
              </small>
            </div>
          </Link>
        ))
      ) : (
        <p className="muted">مقالات منتشرشده اینجا نمایش داده می‌شوند.</p>
      )}
    </section>
  );
}
