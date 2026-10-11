"use client";
import { useEffect, useState, type FormEvent } from "react";
import { api, number } from "@/lib/api";
import type { AuthorPage, User } from "@/lib/types";

const blank = { id: "", name: "", email: "", disabled: false };
export function AuthorManagement() {
  const [page, setPage] = useState(1);
  const [authors, setAuthors] = useState<AuthorPage>({
    items: [],
    total: 0,
    page: 1,
    limit: 20,
  });
  const [draft, setDraft] = useState(blank);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  async function refresh() {
    setAuthors(await api<AuthorPage>(`/studio/authors?page=${page}&limit=20`));
  }
  useEffect(() => {
    let active = true;
    api<AuthorPage>(`/studio/authors?page=${page}&limit=20`)
      .then((v) => {
        if (active) setAuthors(v);
      })
      .catch((e: unknown) => {
        if (active)
          setError(
            e instanceof Error ? e.message : "دریافت نویسندگان انجام نشد.",
          );
      });
    return () => {
      active = false;
    };
  }, [page]);
  async function save(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = e.currentTarget;
    const data = new FormData(form);
    setBusy(true);
    setError("");
    setNotice("");
    try {
      await api<User>(
        draft.id ? `/studio/authors/${draft.id}` : "/studio/authors",
        {
          method: draft.id ? "PATCH" : "POST",
          body: JSON.stringify(
            draft.id
              ? {
                  name: draft.name,
                  email: draft.email,
                  disabled: draft.disabled,
                }
              : {
                  name: draft.name,
                  email: draft.email,
                  password: data.get("password"),
                },
          ),
        },
      );
      setNotice(
        draft.id
          ? "اطلاعات نویسنده ذخیره شد."
          : "حساب نویسنده ایجاد شد؛ نویسنده با ایمیل و رمز تعیین‌شده وارد می‌شود.",
      );
      form.reset();
      setDraft(blank);
      await refresh();
    } catch (e) {
      setError(e instanceof Error ? e.message : "ثبت نویسنده انجام نشد.");
    } finally {
      setBusy(false);
    }
  }
  return (
    <section className="panel" aria-labelledby="authors-title">
      <div className="section-title">
        <h2 id="authors-title">مدیریت نویسندگان</h2>
        <button
          className="text-button"
          disabled={busy}
          onClick={() => {
            setDraft(blank);
            setNotice("");
          }}
        >
          + نویسندهٔ جدید
        </button>
      </div>
      <p className="muted">
        حساب غیرفعال امکان ورود یا استفاده از نشست قبلی را ندارد؛ مقاله‌هایش حفظ
        می‌شوند.
      </p>
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
      <form
        onSubmit={save}
        aria-label={draft.id ? "ویرایش نویسنده" : "ایجاد نویسنده"}
      >
        <div className="form-grid">
          <label>
            نام نویسنده
            <input
              required
              minLength={2}
              maxLength={100}
              value={draft.name}
              onChange={(e) => setDraft({ ...draft, name: e.target.value })}
            />
          </label>
          <label>
            ایمیل نویسنده
            <input
              type="email"
              required
              autoComplete="off"
              dir="ltr"
              value={draft.email}
              onChange={(e) => setDraft({ ...draft, email: e.target.value })}
            />
          </label>
        </div>
        {draft.id ? (
          <label>
            وضعیت حساب
            <select
              value={draft.disabled ? "disabled" : "active"}
              onChange={(e) =>
                setDraft({ ...draft, disabled: e.target.value === "disabled" })
              }
            >
              <option value="active">فعال</option>
              <option value="disabled">غیرفعال</option>
            </select>
          </label>
        ) : (
          <label>
            رمز عبور اولیه
            <input
              name="password"
              type="password"
              required
              minLength={12}
              maxLength={72}
              autoComplete="new-password"
              dir="ltr"
            />
            <small className="muted">حداقل ۱۲ کاراکتر و حداکثر ۷۲ بایت</small>
          </label>
        )}
        <div className="actions">
          <button className="button" disabled={busy}>
            {busy
              ? "در حال ذخیره…"
              : draft.id
                ? "ذخیرهٔ نویسنده"
                : "ایجاد نویسنده"}
          </button>
          {draft.id && (
            <button
              type="button"
              className="text-button"
              disabled={busy}
              onClick={() => setDraft(blank)}
            >
              انصراف
            </button>
          )}
        </div>
      </form>
      <div className="author-list">
        {authors.items.length ? (
          authors.items.map((a) => (
            <article className="author-item" key={a.id}>
              <div>
                <strong>{a.name}</strong>
                <p dir="ltr">{a.email}</p>
                <span className="badge">
                  {a.disabled_at ? "غیرفعال" : "فعال"}
                </span>
              </div>
              <button
                className="button secondary"
                disabled={busy}
                onClick={() => {
                  setDraft({
                    id: a.id,
                    name: a.name,
                    email: a.email,
                    disabled: !!a.disabled_at,
                  });
                  setNotice("");
                  setError("");
                }}
              >
                ویرایش {a.name}
              </button>
            </article>
          ))
        ) : (
          <p className="muted">نویسنده‌ای ثبت نشده است.</p>
        )}
      </div>
      <div className="actions">
        <button
          className="button secondary"
          disabled={busy || page === 1}
          onClick={() => setPage(page - 1)}
        >
          صفحهٔ قبل
        </button>
        <span>
          صفحهٔ {number(page)} · {number(authors.total)} نویسنده
        </span>
        <button
          className="button secondary"
          disabled={busy || page * 20 >= authors.total}
          onClick={() => setPage(page + 1)}
        >
          صفحهٔ بعد
        </button>
      </div>
    </section>
  );
}
