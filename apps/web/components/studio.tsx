"use client";
import Link from "next/link";
import { useEffect, useRef, useState, type FormEvent } from "react";
import { api, date } from "@/lib/api";
import type { Comment, Post, Taxonomy, User } from "@/lib/types";
const empty = {
  id: "",
  title: "",
  slug: "",
  excerpt: "",
  content_html: "",
  category_id: "",
  tag_ids: [] as string[],
  status: "draft" as "draft" | "published",
};
export function Studio() {
  const [user, setUser] = useState<User | null>(null);
  const [ready, setReady] = useState(false);
  const [signup, setSignup] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [posts, setPosts] = useState<Post[]>([]);
  const [comments, setComments] = useState<Comment[]>([]);
  const [commentStatus, setCommentStatus] = useState("pending");
  const [taxonomy, setTaxonomy] = useState<Taxonomy>({
    categories: [],
    tags: [],
  });
  const [draft, setDraft] = useState(empty);
  const editor = useRef<HTMLTextAreaElement>(null);
  useEffect(() => {
    api<User>("/auth/me")
      .then(setUser)
      .catch(() => setUser(null))
      .finally(() => setReady(true));
  }, []);
  async function refresh() {
    const [p, c, t] = await Promise.all([
      api<Post[]>("/studio/posts"),
      api<Comment[]>(`/studio/comments?status=${commentStatus}`),
      api<Taxonomy>("/taxonomy"),
    ]);
    setPosts(p);
    setComments(c);
    setTaxonomy(t);
  }
  useEffect(() => {
    if (user)
      void refresh().catch((e: unknown) =>
        setError(e instanceof Error ? e.message : "دریافت اطلاعات انجام نشد."),
      );
  }, [user, commentStatus]);
  async function authenticate(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const data = new FormData(e.currentTarget);
    setBusy(true);
    setError("");
    try {
      setUser(
        await api<User>(`/auth/${signup ? "signup" : "login"}`, {
          method: "POST",
          body: JSON.stringify({
            email: data.get("email"),
            password: data.get("password"),
            ...(signup
              ? { name: data.get("name"), invite_code: data.get("invite") }
              : {}),
          }),
        }),
      );
    } catch (e) {
      setError(e instanceof Error ? e.message : "ورود انجام نشد.");
    } finally {
      setBusy(false);
    }
  }
  async function logout() {
    try {
      await api<void>("/auth/logout", { method: "POST" });
      setUser(null);
      setPosts([]);
      setComments([]);
      setDraft(empty);
      setNotice("");
    } catch (e) {
      setError(e instanceof Error ? e.message : "خروج انجام نشد.");
    }
  }
  async function save(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setBusy(true);
    setError("");
    setNotice("");
    try {
      const p = await api<Post>("/studio/posts", {
        method: "POST",
        body: JSON.stringify({
          ...draft,
          category_id: draft.category_id || null,
          content_json: {
            version: 1,
            format: "html",
            content: draft.content_html,
          },
        }),
      });
      setDraft({ ...draft, id: p.id, content_html: p.content_html });
      setNotice(
        p.status === "published"
          ? "مقاله منتشر شد. صفحات عمومی حداکثر تا ۶۰ ثانیه به‌روز می‌شوند."
          : "پیش‌نویس ذخیره شد.",
      );
      await refresh();
    } catch (e) {
      setError(e instanceof Error ? e.message : "ذخیره انجام نشد.");
    } finally {
      setBusy(false);
    }
  }
  async function moderate(id: string, status: string) {
    setBusy(true);
    setError("");
    try {
      await api<Comment>(`/studio/comments/${id}`, {
        method: "PATCH",
        body: JSON.stringify({ status }),
      });
      await refresh();
      setNotice(
        status === "approved"
          ? "نظر تأیید شد."
          : status === "spam"
            ? "نظر به هرزنامه منتقل شد."
            : "نظر به صف بررسی بازگشت.",
      );
    } catch (e) {
      setError(e instanceof Error ? e.message : "عملیات انجام نشد.");
    } finally {
      setBusy(false);
    }
  }
  function edit(p: Post) {
    setDraft({
      id: p.id,
      title: p.title,
      slug: p.slug,
      excerpt: p.excerpt,
      content_html: p.content_html,
      category_id: p.category_id || "",
      tag_ids: p.tags.map((t) => t.id),
      status: p.status,
    });
    setNotice("");
    setError("");
    document.getElementById("editor")?.scrollIntoView({ behavior: "smooth" });
  }
  function insert(tag: string) {
    const el = editor.current;
    if (!el) return;
    const a = el.selectionStart,
      b = el.selectionEnd;
    const selected = draft.content_html.slice(a, b) || "متن";
    const fragment = `<${tag}>${selected}</${tag}>`;
    setDraft({
      ...draft,
      content_html:
        draft.content_html.slice(0, a) + fragment + draft.content_html.slice(b),
    });
    el.focus();
  }
  async function addTaxon(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = e.currentTarget;
    const d = new FormData(form);
    setBusy(true);
    setError("");
    try {
      await api(`/studio/taxonomy/${d.get("kind")}`, {
        method: "POST",
        body: JSON.stringify({ name: d.get("name"), slug: d.get("slug") }),
      });
      form.reset();
      await refresh();
      setNotice("موضوع جدید ثبت شد.");
    } catch (e) {
      setError(e instanceof Error ? e.message : "ثبت انجام نشد.");
    } finally {
      setBusy(false);
    }
  }
  if (!ready)
    return (
      <p className="container section-space" role="status">
        در حال بررسی نشست…
      </p>
    );
  if (!user)
    return (
      <section className="auth-panel">
        <p className="eyebrow">فضای نویسندگان</p>
        <h1>{signup ? "به تحریریه بپیوندید" : "خوش آمدید."}</h1>
        <p className="muted">تجربهٔ بعدی را شما روایت کنید.</p>
        <form onSubmit={authenticate}>
          {signup && (
            <>
              <label>
                نام کامل
                <input
                  name="name"
                  required
                  minLength={2}
                  maxLength={100}
                  autoComplete="name"
                />
              </label>
              <label>
                کد دعوت نویسنده
                <input
                  name="invite"
                  required
                  type="password"
                  autoComplete="off"
                />
              </label>
            </>
          )}
          <label>
            ایمیل
            <input
              name="email"
              type="email"
              required
              autoComplete="email"
              dir="ltr"
            />
          </label>
          <label>
            رمز عبور
            <input
              name="password"
              type="password"
              required
              minLength={signup ? 12 : 1}
              maxLength={72}
              autoComplete={signup ? "new-password" : "current-password"}
              dir="ltr"
            />
          </label>
          {signup && (
            <p className="muted text-sm">
              رمز عبور باید حداقل ۱۲ کاراکتر و حداکثر ۷۲ بایت باشد. کد دعوت را
              از مدیر دریافت کنید.
            </p>
          )}
          <button className="button" disabled={busy}>
            {busy ? "در حال بررسی…" : signup ? "ساخت حساب نویسنده" : "ورود"}
          </button>
        </form>
        <button
          className="text-button"
          onClick={() => {
            setSignup(!signup);
            setError("");
          }}
        >
          {signup ? "حساب دارید؟ وارد شوید" : "کد دعوت دارید؟ ثبت‌نام کنید"}
        </button>
        {error && (
          <p className="error" role="alert">
            {error}
          </p>
        )}
      </section>
    );
  return (
    <section className="container section-space">
      <div className="section-title">
        <div>
          <p className="eyebrow">
            پنل {user.role === "admin" ? "مدیریت" : "نویسنده"}
          </p>
          <h1>سلام، {user.name}</h1>
        </div>
        <button className="button secondary" onClick={logout}>
          خروج
        </button>
      </div>
      {error && (
        <p className="error" role="alert">
          {error}
        </p>
      )}
      {notice && (
        <p className="success" role="status">
          {notice}
        </p>
      )}
      <div className="studio-layout">
        <section className="panel" id="editor">
          <div className="section-title">
            <h2>{draft.id ? "ویرایش مقاله" : "مقالهٔ تازه"}</h2>
            <button
              className="text-button"
              onClick={() => {
                setDraft(empty);
                setNotice("");
              }}
            >
              + نوشتهٔ جدید
            </button>
          </div>
          <form onSubmit={save}>
            <label>
              عنوان
              <input
                required
                maxLength={240}
                value={draft.title}
                onChange={(e) => setDraft({ ...draft, title: e.target.value })}
              />
            </label>
            <label>
              نشانی مقاله (حروف، عدد و خط تیره)
              <input
                required
                maxLength={200}
                value={draft.slug}
                onChange={(e) => setDraft({ ...draft, slug: e.target.value })}
              />
            </label>
            <label>
              خلاصه
              <textarea
                maxLength={500}
                rows={3}
                value={draft.excerpt}
                onChange={(e) =>
                  setDraft({ ...draft, excerpt: e.target.value })
                }
              />
            </label>
            <div className="editor-toolbar" aria-label="قالب‌بندی متن">
              {[
                ["p", "پاراگراف"],
                ["h2", "تیتر"],
                ["strong", "پررنگ"],
                ["blockquote", "نقل‌قول"],
                ["code", "کد"],
              ].map(([tag, label]) => (
                <button key={tag} type="button" onClick={() => insert(tag)}>
                  {label}
                </button>
              ))}
            </div>
            <label>
              متن مقاله (HTML)
              <textarea
                ref={editor}
                required
                rows={16}
                value={draft.content_html}
                onChange={(e) =>
                  setDraft({ ...draft, content_html: e.target.value })
                }
              />
            </label>
            <p className="muted text-sm">
              متن را انتخاب کنید و قالب‌بندی را بزنید. HTML در سرور پاک‌سازی
              می‌شود؛ اسکریپت و iframe مجاز نیست.
            </p>
            <div className="form-grid">
              <label>
                دسته‌بندی
                <select
                  value={draft.category_id}
                  onChange={(e) =>
                    setDraft({ ...draft, category_id: e.target.value })
                  }
                >
                  <option value="">بدون دسته‌بندی</option>
                  {taxonomy.categories.map((c) => (
                    <option key={c.id} value={c.id}>
                      {c.name}
                    </option>
                  ))}
                </select>
              </label>
              <label>
                وضعیت
                <select
                  value={draft.status}
                  onChange={(e) =>
                    setDraft({
                      ...draft,
                      status: e.target.value as "draft" | "published",
                    })
                  }
                >
                  <option value="draft">پیش‌نویس</option>
                  <option value="published">انتشار عمومی</option>
                </select>
              </label>
            </div>
            <fieldset>
              <legend>برچسب‌ها</legend>
              <div className="checkboxes">
                {taxonomy.tags.map((t) => (
                  <label key={t.id}>
                    <input
                      type="checkbox"
                      checked={draft.tag_ids.includes(t.id)}
                      onChange={(e) =>
                        setDraft({
                          ...draft,
                          tag_ids: e.target.checked
                            ? [...draft.tag_ids, t.id]
                            : draft.tag_ids.filter((id) => id !== t.id),
                        })
                      }
                    />
                    {t.name}
                  </label>
                ))}
              </div>
            </fieldset>
            <div className="actions">
              <button className="button" disabled={busy}>
                {busy
                  ? "در حال ذخیره…"
                  : draft.status === "published"
                    ? "ذخیره و انتشار"
                    : "ذخیرهٔ پیش‌نویس"}
              </button>
              {draft.id && draft.status === "published" && (
                <Link className="text-button" href={`/blog/${draft.slug}`}>
                  مشاهدهٔ مقاله ←
                </Link>
              )}
            </div>
          </form>
        </section>
        <aside>
          <section className="panel">
            <h2>{user.role === "admin" ? "همهٔ نوشته‌ها" : "نوشته‌های شما"}</h2>
            {posts.length ? (
              posts.map((p) => (
                <button
                  className="studio-post"
                  key={p.id}
                  onClick={() => edit(p)}
                >
                  <strong>{p.title}</strong>
                  <small>
                    {p.status === "published" ? "منتشرشده" : "پیش‌نویس"} ·{" "}
                    {date(p.updated_at)}
                  </small>
                </button>
              ))
            ) : (
              <p className="muted">اولین مقالهٔ خود را بنویسید.</p>
            )}
          </section>
          {user.role === "admin" && (
            <section className="panel">
              <h2>دسته‌بندی و برچسب</h2>
              <form onSubmit={addTaxon}>
                <label>
                  نوع
                  <select name="kind">
                    <option value="categories">دسته‌بندی</option>
                    <option value="tags">برچسب</option>
                  </select>
                </label>
                <label>
                  نام
                  <input name="name" required maxLength={100} />
                </label>
                <label>
                  نامک
                  <input name="slug" required maxLength={200} />
                </label>
                <button className="button secondary" disabled={busy}>
                  ایجاد
                </button>
              </form>
            </section>
          )}
        </aside>
      </div>
      <section className="panel moderation">
        <div className="section-title">
          <h2>مدیریت نظرات</h2>
          <button
            className="text-button"
            onClick={() =>
              void refresh().catch((e: unknown) =>
                setError(e instanceof Error ? e.message : "خطا"),
              )
            }
          >
            به‌روزرسانی
          </button>
        </div>
        <nav className="filters" aria-label="وضعیت نظرات">
          {[
            ["pending", "در انتظار بررسی"],
            ["approved", "تأییدشده"],
            ["spam", "هرزنامه"],
          ].map(([value, label]) => (
            <button
              key={value}
              className={`button ${commentStatus === value ? "" : "secondary"}`}
              onClick={() => setCommentStatus(value)}
            >
              {label}
            </button>
          ))}
        </nav>
        {comments.length ? (
          comments.map((c) => (
            <article className="moderation-item" key={c.id}>
              <strong>{c.author_name}</strong>
              <time>{date(c.created_at)}</time>
              {c.parent_id && <span className="badge">پاسخ به یک نظر</span>}
              <p className="whitespace-pre-wrap">{c.content}</p>
              <div className="actions">
                {c.status !== "approved" && (
                  <button
                    className="button"
                    disabled={busy}
                    onClick={() => moderate(c.id, "approved")}
                  >
                    تأیید و انتشار
                  </button>
                )}
                {c.status !== "spam" && (
                  <button
                    className="button secondary"
                    disabled={busy}
                    onClick={() => moderate(c.id, "spam")}
                  >
                    هرزنامه
                  </button>
                )}
                {c.status !== "pending" && (
                  <button
                    className="button secondary"
                    disabled={busy}
                    onClick={() => moderate(c.id, "pending")}
                  >
                    بازگردانی به بررسی
                  </button>
                )}
                {posts.find((p) => p.id === c.post_id) && (
                  <Link
                    className="text-button"
                    href={`/blog/${posts.find((p) => p.id === c.post_id)!.slug}#comments`}
                  >
                    مقاله و پاسخ دادن ←
                  </Link>
                )}
              </div>
            </article>
          ))
        ) : (
          <p className="muted">نظری در این وضعیت نیست.</p>
        )}
      </section>
    </section>
  );
}
