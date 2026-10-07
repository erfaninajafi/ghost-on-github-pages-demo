"use client";
import { useEffect, useState, type FormEvent } from "react";
import { api, date } from "@/lib/api";
import type { Comment } from "@/lib/types";
function CommentForm({
  postID,
  parentID,
  onCancel,
}: {
  postID: string;
  parentID?: string;
  onCancel?: () => void;
}) {
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");
  async function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = e.currentTarget;
    const data = new FormData(form);
    setBusy(true);
    setError("");
    setMessage("");
    try {
      await api<Comment>(`/posts/${postID}/comments`, {
        method: "POST",
        body: JSON.stringify({
          author_name: data.get("name"),
          author_email: data.get("email"),
          content: data.get("content"),
          parent_id: parentID || null,
        }),
      });
      form.reset();
      setMessage("نظر شما ثبت شد و پس از تأیید نمایش داده می‌شود.");
    } catch (e) {
      setError(e instanceof Error ? e.message : "ثبت نظر انجام نشد.");
    } finally {
      setBusy(false);
    }
  }
  return (
    <form className="comment-form" onSubmit={submit}>
      <h3>{parentID ? "پاسخ شما" : "به گفت‌وگو بپیوندید"}</h3>
      <div className="form-grid">
        <label>
          نام شما
          <input
            name="name"
            required
            minLength={2}
            maxLength={100}
            autoComplete="name"
          />
        </label>
        <label>
          ایمیل <span className="muted">(نمایش داده نمی‌شود)</span>
          <input
            name="email"
            required
            type="email"
            maxLength={254}
            autoComplete="email"
            dir="ltr"
          />
        </label>
      </div>
      <label>
        متن {parentID ? "پاسخ" : "نظر"}
        <textarea
          name="content"
          required
          minLength={2}
          maxLength={5000}
          rows={4}
        />
      </label>
      <p className="muted text-sm">
        با ثبت نظر، اعلان پاسخ به ایمیل شما ارسال می‌شود.
      </p>
      <div className="actions">
        <button className="button" disabled={busy}>
          {busy ? "در حال ارسال…" : "ارسال برای بررسی"}
        </button>
        {onCancel && (
          <button type="button" className="button secondary" onClick={onCancel}>
            انصراف
          </button>
        )}
      </div>
      {message && (
        <p role="status" className="success">
          {message}
        </p>
      )}
      {error && (
        <p role="alert" className="error">
          {error}
        </p>
      )}
    </form>
  );
}
function CommentNode({
  comment,
  postID,
  depth = 1,
}: {
  comment: Comment;
  postID: string;
  depth?: number;
}) {
  const [reply, setReply] = useState(false);
  return (
    <li className="comment-node">
      <article>
        <div className="comment-heading">
          <span className="avatar">{comment.author_name.slice(0, 1)}</span>
          <strong>{comment.author_name}</strong>
          <time dateTime={comment.created_at}>{date(comment.created_at)}</time>
        </div>
        <p className="whitespace-pre-wrap break-words">{comment.content}</p>
        {depth < 8 && (
          <button
            className="text-button"
            onClick={() => setReply(!reply)}
            aria-expanded={reply}
          >
            پاسخ دادن ↶
          </button>
        )}
        {reply && (
          <CommentForm
            postID={postID}
            parentID={comment.id}
            onCancel={() => setReply(false)}
          />
        )}
      </article>
      {comment.replies.length > 0 && (
        <ul className="replies">
          {comment.replies.map((c) => (
            <CommentNode
              key={c.id}
              comment={c}
              postID={postID}
              depth={depth + 1}
            />
          ))}
        </ul>
      )}
    </li>
  );
}
export function Comments({ postID }: { postID: string }) {
  const [comments, setComments] = useState<Comment[]>([]);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(true);
  const [attempt, setAttempt] = useState(0);
  useEffect(() => {
    const c = new AbortController();
    setBusy(true);
    setError("");
    api<Comment[]>(`/posts/${postID}/comments`, { signal: c.signal })
      .then(setComments)
      .catch((e: unknown) => {
        if (!c.signal.aborted)
          setError(e instanceof Error ? e.message : "خطا در دریافت نظرات");
      })
      .finally(() => {
        if (!c.signal.aborted) setBusy(false);
      });
    return () => c.abort();
  }, [postID, attempt]);
  return (
    <section id="comments" className="comments">
      <div className="section-title">
        <h2>گفت‌وگو و دیدگاه‌ها</h2>
      </div>
      {busy ? (
        <p role="status">در حال دریافت نظرات…</p>
      ) : error ? (
        <p role="alert">
          {error}{" "}
          <button onClick={() => setAttempt(attempt + 1)}>تلاش مجدد</button>
        </p>
      ) : comments.length ? (
        <ul>
          {comments.map((c) => (
            <CommentNode key={c.id} comment={c} postID={postID} />
          ))}
        </ul>
      ) : (
        <p className="muted">اولین دیدگاه را شما بنویسید.</p>
      )}
      <CommentForm postID={postID} />
    </section>
  );
}
