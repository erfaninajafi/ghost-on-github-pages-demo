import Link from "next/link";
import { getPosts, getPopular, getTaxonomy, number } from "@/lib/api";
import { PostCard, Trending } from "@/components/post-card";
export const revalidate = 60;
export default async function Home({
  searchParams,
}: {
  searchParams: Promise<{ page?: string; category?: string; tag?: string }>;
}) {
  const query = await searchParams;
  const page = Math.min(
    10000,
    Math.max(1, Number.parseInt(query.page || "1", 10) || 1),
  );
  const params = new URLSearchParams({ page: String(page), limit: "9" });
  if (query.category) params.set("category", query.category);
  if (query.tag) params.set("tag", query.tag);
  const [data, popular, taxonomy] = await Promise.all([
    getPosts(params),
    getPopular(),
    getTaxonomy(),
  ]);
  const pageLink = (n: number) => {
    const p = new URLSearchParams(params);
    p.set("page", String(n));
    p.delete("limit");
    return `/?${p}#articles`;
  };
  return (
    <>
      <section className="container hero">
        <div className="hero-content">
          <p className="eyebrow light">
            <span className="dot" /> مجلهٔ اکسپرتس آکادمی
          </p>
          <h1>
            از دانستن،
            <br />
            تا <em>متخصص شدن.</em>
          </h1>
          <p>
            اینجا تجربه‌های واقعی، ایده‌های تازه و دانش کاربردی را به اشتراک
            می‌گذاریم. برای قدم بعدی مسیر حرفه‌ای شما.
          </p>
          <Link className="button" href="#articles">
            کاوش در مقاله‌ها <span>↙</span>
          </Link>
          <div className="hero-footnote">
            <span>یاد بگیر.</span>
            <span>تجربه کن.</span>
            <span>رشد کن.</span>
          </div>
        </div>
        <div className="hero-art" aria-hidden="true">
          <div className="orbit orbit-one" />
          <div className="orbit orbit-two" />
          <span className="hero-star">✳</span>
          <div className="floating-note">
            <span>KNOWLEDGE</span>
            <strong>
              آینده را
              <br />
              یاد بگیر.
            </strong>
            <i>↗</i>
          </div>
          <span className="vertical-type" dir="ltr">
            A SPACE FOR EXPERT MINDS
          </span>
        </div>
      </section>
      <section className="container editorial-layout" id="articles">
        <div>
          <div className="section-title">
            <div>
              <p className="eyebrow">از نگاه متخصصان</p>
              <h2>تازه‌ترین نوشته‌ها</h2>
            </div>
            <span className="muted">{number(data.total)} مقاله</span>
          </div>
          <nav className="filters" aria-label="فیلتر دسته‌بندی">
            <Link className={!query.category ? "active" : ""} href="/#articles">
              همهٔ موضوعات
            </Link>
            {taxonomy.categories.map((c) => (
              <Link
                className={query.category === c.slug ? "active" : ""}
                key={c.id}
                href={`/?category=${encodeURIComponent(c.slug)}#articles`}
              >
                {c.name}
              </Link>
            ))}
          </nav>
          {query.tag && (
            <p className="filter-notice">
              برچسب: {query.tag} <Link href="/#articles">حذف فیلتر ×</Link>
            </p>
          )}
          {data.items.length ? (
            <div className="post-grid">
              {data.items.map((p, i) => (
                <PostCard key={p.id} post={p} index={i} />
              ))}
            </div>
          ) : (
            <div className="empty-state">
              <span>✦</span>
              <h3>داستان بعدی، از اینجا شروع می‌شود.</h3>
              <p>هنوز مقاله‌ای در این بخش منتشر نشده است.</p>
              <Link href="/studio" className="text-button">
                ورود به پنل نویسندگان ←
              </Link>
            </div>
          )}
          <nav className="pagination" aria-label="صفحه‌بندی">
            {page > 1 && (
              <Link className="button secondary" href={pageLink(page - 1)}>
                صفحهٔ قبل
              </Link>
            )}
            <span>صفحهٔ {number(page)}</span>
            {page * data.limit < data.total && (
              <Link className="button secondary" href={pageLink(page + 1)}>
                صفحهٔ بعد
              </Link>
            )}
          </nav>
        </div>
        <aside>
          <Trending posts={popular} />
          <div className="aside-note">
            <span>✦</span>
            <h3>تجربهٔ شما ارزشمند است.</h3>
            <p>
              با نوشتن، دانسته‌هایتان را عمیق‌تر کنید و در رشد دیگران سهم داشته
              باشید.
            </p>
            <Link href="/studio">به جمع نویسندگان بپیوندید ←</Link>
          </div>
          {taxonomy.tags.length > 0 && (
            <div className="tag-list">
              <h3>برچسب‌ها</h3>
              {taxonomy.tags.map((t) => (
                <Link
                  key={t.id}
                  href={`/?tag=${encodeURIComponent(t.slug)}#articles`}
                >
                  #{t.name}
                </Link>
              ))}
            </div>
          )}
        </aside>
      </section>
    </>
  );
}
