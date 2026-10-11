"use client";
import Link from "next/link";
import { useEffect, useRef, useState, type FormEvent } from "react";
import { api, date } from "@/lib/api";
import {
  readEditorDocument,
  textBlock,
  contentPayload,
  type ContentFormat,
} from "@/lib/editor";
import type { Comment, Post, Taxonomy, User } from "@/lib/types";
import { AuthorManagement } from "./author-management";
import { CoverUpload } from "./cover-upload";
import { ContentEditor } from "./content-editor";
import { PasswordField } from "./password-field";
import { signupPasswordError } from "@/lib/password";
import { articlePath } from "@/lib/article-path";
const statusLabels = {
  draft: "پیش‌نویس",
  published: "منتشرشده",
  archived: "آرشیوشده",
  deleted: "حذف‌شده",
};
const empty = {
  id: "",
  title: "",
  slug: "",
  excerpt: "",
  blocks: [textBlock()],
  cover_image_url: null as string | null,
  content_format: "plain_text" as ContentFormat,
  category_id: "",
  tag_ids: [] as string[],
  status: "draft" as "draft" | "published",
};
export function Studio() {
  const [user, setUser] = useState<User | null>(null);
  const [ready, setReady] = useState(false);
  const [signup, setSignup] = useState(false);
  const [tab, setTab] = useState<"articles" | "comments" | "authors">(
    "articles",
  );
  const [busy, setBusy] = useState(false);
  const [uploadBusy, setUploadBusy] = useState(false);
  const [editorPending, setEditorPending] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [posts, setPosts] = useState<Post[]>([]);
  const [postFilter, setPostFilter] = useState("all");
  const [postAction, setPostAction] = useState<{
    post: Post;
    status: "archived" | "deleted" | "draft";
  } | null>(null);
  const [comments, setComments] = useState<Comment[]>([]);
  const [commentStatus, setCommentStatus] = useState("pending");
  const [taxonomy, setTaxonomy] = useState<Taxonomy>({
    categories: [],
    tags: [],
  });
  const [draft, setDraft] = useState(empty);
  const [editorSession, setEditorSession] = useState(0);
  const actionPanel = useRef<HTMLElement>(null);
  useEffect(() => {
    if (postAction) {
      actionPanel.current?.scrollIntoView({
        behavior: "smooth",
        block: "center",
      });
      actionPanel.current?.focus();
    }
  }, [postAction]);
  useEffect(() => {
    api<User>("/auth/me")
      .then(setUser)
      .catch(() => setUser(null))
      .finally(() => setReady(true));
  }, []);
  async function refresh() {
    const [p, c, t] = await Promise.all([
      api<Post[]>(
        postFilter === "deleted"
          ? "/studio/posts?status=deleted"
          : "/studio/posts",
      ),
      user?.role === "admin" && tab === "comments"
        ? api<Comment[]>(`/studio/comments?status=${commentStatus}`)
        : Promise.resolve([] as Comment[]),
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
  }, [user, commentStatus, postFilter, tab]);
  async function authenticate(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const data = new FormData(e.currentTarget);
    setError("");
    if (signup) {
      const message = signupPasswordError(
        data.get("password"),
        data.get("password_confirm"),
      );
      if (message) {
        setError(message);
        return;
      }
    }
    setBusy(true);
    try {
      setUser(
        await api<User>(`/auth/${signup ? "signup" : "login"}`, {
          method: "POST",
          body: JSON.stringify({
            email: data.get("email"),
            password: data.get("password"),
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
      setTab("articles");
      setPosts([]);
      setComments([]);
      setPostFilter("all");
      setPostAction(null);
      setDraft(empty);
      setNotice("");
    } catch (e) {
      setError(e instanceof Error ? e.message : "خروج انجام نشد.");
    }
  }
  async function applyPostAction() {
    if (!postAction) return;
    const { post, status } = postAction;
    setBusy(true);
    setError("");
    setNotice("");
    try {
      await api(
        status === "deleted"
          ? `/studio/posts/${post.id}`
          : `/studio/posts/${post.id}/status`,
        {
          method: status === "deleted" ? "DELETE" : "PATCH",
          ...(status !== "deleted" ? { body: JSON.stringify({ status }) } : {}),
        },
      );
      if (draft.id === post.id) setDraft(empty);
      setPostAction(null);
      await refresh();
      setNotice(
        status === "deleted"
          ? "مقاله به سطل حذف منتقل شد و قابل بازگردانی است."
          : status === "archived"
            ? "مقاله آرشیو شد و از نمایش عمومی خارج شد."
            : "مقاله به پیش‌نویس بازگردانده شد؛ برای انتشار دوباره آن را ویرایش و منتشر کنید.",
      );
    } catch (e) {
      setError(e instanceof Error ? e.message : "عملیات انجام نشد.");
    } finally {
      setBusy(false);
    }
  }
  async function save(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (uploadBusy || editorPending) return;
    setBusy(true);
    setError("");
    setNotice("");
    try {
      const { blocks, content_format, ...fields } = draft;
      const p = await api<Post>("/studio/posts", {
        method: "POST",
        body: JSON.stringify({
          ...fields,
          category_id: draft.category_id || null,
          ...contentPayload(content_format, blocks),
        }),
      });
      setDraft({
        ...draft,
        id: p.id,
        slug: p.slug,
        title: p.title,
        cover_image_url: p.cover_image_url,
        ...readEditorDocument(p),
      });
      setNotice(
        p.status === "published"
          ? "مقاله منتشر شد. از «مشاهدهٔ مقاله» آن را باز کنید."
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
    if (p.status === "archived" || p.status === "deleted") return;
    setEditorSession((previous) => previous + 1);
    setDraft({
      id: p.id,
      title: p.title,
      slug: p.slug,
      excerpt: p.excerpt,
      cover_image_url: p.cover_image_url,
      ...readEditorDocument(p),
      category_id: p.category_id || "",
      tag_ids: p.tags.map((t) => t.id),
      status: p.status,
    });
    setNotice("");
    setError("");
    document.getElementById("editor")?.scrollIntoView({ behavior: "smooth" });
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
          <PasswordField
            key={signup ? "signup-password" : "login-password"}
            name="password"
            label="رمز عبور"
            signup={signup}
            disabled={busy}
          />
          {signup && (
            <PasswordField
              name="password_confirm"
              label="تکرار رمز عبور"
              signup
              disabled={busy}
            />
          )}
          {signup && (
            <p className="muted text-sm">
              فقط با ایمیل و رمز عبور حساب نویسنده بسازید. رمز عبور باید حداقل
              ۱۲ کاراکتر و حداکثر ۷۲ بایت باشد.
            </p>
          )}
          <button className="button" disabled={busy || uploadBusy}>
            {busy ? "در حال بررسی…" : signup ? "ساخت حساب نویسنده" : "ورود"}
          </button>
        </form>
        <button
          className="text-button"
          disabled={busy}
          onClick={() => {
            setSignup(!signup);
            setError("");
          }}
        >
          {signup ? "حساب دارید؟ وارد شوید" : "حساب ندارید؟ ثبت‌نام کنید"}
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
        <button
          className="button secondary"
          disabled={busy || uploadBusy}
          onClick={logout}
        >
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
      <p className="muted">
        {user.role === "admin"
          ? "شما می‌توانید مقاله‌های خود و همهٔ نویسندگان را مدیریت کنید. مقالهٔ جدید به نام حساب شما ثبت می‌شود."
          : "شما می‌توانید مقاله‌های خود را بنویسید، ویرایش و منتشر کنید. مدیریت نویسندگان، نظرات، حذف و آرشیو با مدیر است."}
      </p>
      {user.role === "admin" && (
        <nav className="studio-tabs" role="tablist" aria-label="بخش‌های مدیریت">
          {(
            [
              ["articles", "مقاله‌ها"],
              ["comments", "مدیریت نظرات"],
              ["authors", "مدیریت نویسندگان"],
            ] as const
          ).map(([value, label], index) => (
            <button
              key={value}
              id={`tab-${value}`}
              type="button"
              role="tab"
              aria-selected={tab === value}
              aria-controls={`panel-${value}`}
              tabIndex={tab === value ? 0 : -1}
              disabled={busy || uploadBusy}
              onClick={() => {
                setTab(value);
                setError("");
                setNotice("");
              }}
              onKeyDown={(e) => {
                const tabs = ["articles", "comments", "authors"] as const;
                let target = index;
                if (e.key === "ArrowLeft") target = (index + 1) % tabs.length;
                else if (e.key === "ArrowRight")
                  target = (index + tabs.length - 1) % tabs.length;
                else if (e.key === "Home") target = 0;
                else if (e.key === "End") target = tabs.length - 1;
                else return;
                e.preventDefault();
                setTab(tabs[target]);
                document.getElementById(`tab-${tabs[target]}`)?.focus();
              }}
            >
              {label}
            </button>
          ))}
        </nav>
      )}
      <div
        id="panel-articles"
        role={user.role === "admin" ? "tabpanel" : undefined}
        aria-labelledby={user.role === "admin" ? "tab-articles" : undefined}
        hidden={user.role === "admin" && tab !== "articles"}
      >
        {postAction && (
          <section
            className="panel action-confirmation"
            ref={actionPanel}
            tabIndex={-1}
            role="alertdialog"
            aria-modal="false"
            aria-labelledby="post-action-title"
          >
            <h2 id="post-action-title">
              {postAction.status === "deleted"
                ? "حذف مقاله"
                : postAction.status === "archived"
                  ? "آرشیو مقاله"
                  : "بازگردانی مقاله"}
            </h2>
            <p>
              «{postAction.post.title}»{" "}
              {postAction.status === "deleted"
                ? "از سایت خارج و به سطل حذف منتقل می‌شود. محتوا و نظرات برای بازگردانی حفظ می‌شوند."
                : postAction.status === "archived"
                  ? "از نمایش عمومی خارج می‌شود و در آرشیو مدیر باقی می‌ماند."
                  : "به پیش‌نویس خصوصی بازمی‌گردد و خودکار منتشر نمی‌شود."}
            </p>
            <div className="actions">
              <button
                className="button"
                disabled={busy || uploadBusy}
                onClick={() => void applyPostAction()}
              >
                {busy ? "در حال انجام…" : "تأیید عملیات"}
              </button>
              <button
                className="button secondary"
                disabled={busy || uploadBusy}
                onClick={() => setPostAction(null)}
              >
                انصراف
              </button>
            </div>
          </section>
        )}
        <div className="studio-layout">
          <section className="panel" id="editor">
            <div className="section-title">
              <h2>{draft.id ? "ویرایش مقاله" : "مقالهٔ تازه"}</h2>
              <button
                className="text-button"
                onClick={() => {
                  setDraft(empty);
                  setEditorSession((previous) => previous + 1);
                  setNotice("");
                }}
                disabled={busy || uploadBusy || editorPending}
              >
                + نوشتهٔ جدید
              </button>
            </div>
            <form onSubmit={save}>
              <fieldset
                disabled={busy || uploadBusy}
                className="article-fields"
              >
                <CoverUpload
                  value={draft.cover_image_url}
                  onChange={(url) =>
                    setDraft((previous) => ({
                      ...previous,
                      cover_image_url: url,
                    }))
                  }
                  disabled={busy || uploadBusy}
                  onBusyChange={setUploadBusy}
                />
                <label>
                  عنوان
                  <input
                    required
                    maxLength={240}
                    value={draft.title}
                    onChange={(e) =>
                      setDraft({ ...draft, title: e.target.value })
                    }
                  />
                </label>
                <label>
                  نشانی مقاله (حروف، عدد و خط تیره)
                  <input
                    required
                    maxLength={200}
                    value={draft.slug}
                    onChange={(e) =>
                      setDraft({ ...draft, slug: e.target.value })
                    }
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
                <label>
                  قالب متن
                  <select
                    value={draft.content_format}
                    onChange={(e) =>
                      setDraft({
                        ...draft,
                        content_format: e.target.value as ContentFormat,
                      })
                    }
                  >
                    <option value="plain_text">متن ساده (Plain text)</option>
                    <option value="html">HTML</option>
                  </select>
                </label>
                <ContentEditor
                  key={`${draft.id || "new"}:${editorSession}`}
                  format={draft.content_format}
                  blocks={draft.blocks}
                  onChange={(blocks) =>
                    setDraft((previous) => ({ ...previous, blocks }))
                  }
                  disabled={busy || uploadBusy}
                  onBusyChange={setUploadBusy}
                  onPendingChange={setEditorPending}
                />
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
                  <button
                    className="button"
                    disabled={busy || uploadBusy || editorPending}
                  >
                    {busy
                      ? "در حال ذخیره…"
                      : draft.status === "published"
                        ? "ذخیره و انتشار"
                        : "ذخیرهٔ پیش‌نویس"}
                  </button>
                  {draft.id && draft.status === "published" && (
                    <Link
                      className="text-button"
                      href={articlePath(draft.slug)}
                      prefetch={false}
                    >
                      مشاهدهٔ مقاله ←
                    </Link>
                  )}
                </div>
              </fieldset>
            </form>
          </section>
          <aside>
            <section className="panel">
              <h2>
                {user.role === "admin" ? "همهٔ نوشته‌ها" : "نوشته‌های شما"}
              </h2>
              {user.role === "admin" && (
                <label>
                  فیلتر نوشته‌ها
                  <select
                    value={postFilter}
                    disabled={busy || uploadBusy}
                    onChange={(e) => {
                      setPostFilter(e.target.value);
                      setPostAction(null);
                    }}
                  >
                    <option value="all">همهٔ نوشته‌ها</option>
                    <option value="mine">نوشته‌های من</option>
                    <option value="archived">آرشیو</option>
                    <option value="deleted">سطل حذف</option>
                  </select>
                </label>
              )}
              {posts.filter((p) =>
                postFilter === "mine"
                  ? p.author_id === user.id
                  : postFilter === "archived"
                    ? p.status === "archived"
                    : true,
              ).length ? (
                posts
                  .filter((p) =>
                    postFilter === "mine"
                      ? p.author_id === user.id
                      : postFilter === "archived"
                        ? p.status === "archived"
                        : true,
                  )
                  .map((p) => (
                    <article className="studio-post" key={p.id}>
                      {p.status === "archived" || p.status === "deleted" ? (
                        <strong>{p.title}</strong>
                      ) : (
                        <button
                          className="studio-post-title"
                          disabled={busy || uploadBusy}
                          onClick={() => edit(p)}
                        >
                          <strong>{p.title}</strong>
                        </button>
                      )}
                      <small>
                        {statusLabels[p.status]} · {date(p.updated_at)}
                      </small>
                      {user.role === "admin" && (
                        <>
                          <small>
                            {p.author_id === user.id
                              ? "نویسنده: شما"
                              : `نویسنده: ${p.author_name || "حساب دیگر"}`}
                          </small>
                          <div className="actions">
                            {p.status === "archived" ||
                            p.status === "deleted" ? (
                              <button
                                className="text-button"
                                disabled={busy || uploadBusy}
                                onClick={() =>
                                  setPostAction({ post: p, status: "draft" })
                                }
                              >
                                بازگردانی به پیش‌نویس
                              </button>
                            ) : (
                              <button
                                className="text-button"
                                disabled={busy || uploadBusy}
                                onClick={() =>
                                  setPostAction({ post: p, status: "archived" })
                                }
                              >
                                آرشیو
                              </button>
                            )}
                            {p.status !== "deleted" && (
                              <button
                                className="text-button danger-button"
                                disabled={busy || uploadBusy}
                                onClick={() =>
                                  setPostAction({ post: p, status: "deleted" })
                                }
                              >
                                حذف
                              </button>
                            )}
                          </div>
                        </>
                      )}
                    </article>
                  ))
              ) : (
                <p className="muted">نوشته‌ای در این فهرست نیست.</p>
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
                  <button
                    className="button secondary"
                    disabled={busy || uploadBusy}
                  >
                    ایجاد
                  </button>
                </form>
              </section>
            )}
          </aside>
        </div>
      </div>
      {user.role === "admin" && (
        <>
          <div
            id="panel-authors"
            role="tabpanel"
            aria-labelledby="tab-authors"
            hidden={tab !== "authors"}
          >
            <AuthorManagement />
          </div>
          <div
            id="panel-comments"
            role="tabpanel"
            aria-labelledby="tab-comments"
            hidden={tab !== "comments"}
          >
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
                    {c.parent_id && (
                      <span className="badge">پاسخ به یک نظر</span>
                    )}
                    <p className="whitespace-pre-wrap">{c.content}</p>
                    <div className="actions">
                      {c.status !== "approved" && (
                        <button
                          className="button"
                          disabled={busy || uploadBusy}
                          onClick={() => moderate(c.id, "approved")}
                        >
                          تأیید و انتشار
                        </button>
                      )}
                      {c.status !== "spam" && (
                        <button
                          className="button secondary"
                          disabled={busy || uploadBusy}
                          onClick={() => moderate(c.id, "spam")}
                        >
                          هرزنامه
                        </button>
                      )}
                      {c.status !== "pending" && (
                        <button
                          className="button secondary"
                          disabled={busy || uploadBusy}
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
          </div>
        </>
      )}
    </section>
  );
}
