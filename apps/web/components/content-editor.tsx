"use client";
import { useEffect, useRef, useState, type ChangeEvent } from "react";
import { api } from "@/lib/api";
import { articlePath } from "@/lib/article-path";
import {
  insertContentBlock,
  removeContentBlock,
  renderContentBlocks,
  textBlock,
  validArticleLink,
  type ContentBlock,
  type ContentFormat,
} from "@/lib/editor";
import { safeHTML } from "@/lib/seo";
import type { Post } from "@/lib/types";

type Selection = { index: number; start: number; end: number };
function ImageSizeControl({
  width,
  onChange,
}: {
  width: number;
  onChange: (width: number) => void;
}) {
  const [value, setValue] = useState(String(width));
  useEffect(() => setValue(String(width)), [width]);
  return (
    <div className="image-size-control">
      <label>
        عرض تصویر (پیکسل)
        <input
          type="number"
          min={80}
          max={2000}
          value={value}
          onChange={(e) => {
            setValue(e.target.value);
            const next = Number(e.target.value);
            if (Number.isInteger(next) && next >= 80 && next <= 2000)
              onChange(next);
          }}
          onBlur={() => setValue(String(width))}
        />
      </label>
      <label>
        کوچک / بزرگ کردن
        <input
          type="range"
          min={80}
          max={2000}
          step={1}
          value={width}
          onChange={(e) => onChange(Number(e.target.value))}
        />
      </label>
    </div>
  );
}
export function ContentEditor({
  format,
  blocks,
  onChange,
  disabled,
  onBusyChange,
  onPendingChange,
}: {
  format: ContentFormat;
  blocks: ContentBlock[];
  onChange: (blocks: ContentBlock[]) => void;
  disabled: boolean;
  onBusyChange: (busy: boolean) => void;
  onPendingChange: (pending: boolean) => void;
}) {
  const inputs = useRef<(HTMLTextAreaElement | null)[]>([]);
  const selection = useRef<Selection | null>(null);
  const fileInput = useRef<HTMLInputElement>(null);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState("");
  const [linkOpen, setLinkOpen] = useState(false);
  const [linkIndex, setLinkIndex] = useState<number | null>(null);
  const [linkType, setLinkType] = useState<"internal" | "external">("internal");
  const [linkText, setLinkText] = useState("");
  const [linkURL, setLinkURL] = useState("");
  const [query, setQuery] = useState("");
  const [suggestions, setSuggestions] = useState<Post[]>([]);
  const [searching, setSearching] = useState(false);
  const [linkError, setLinkError] = useState("");
  const linkSelection = useRef<Selection | null>(null);
  const locked = disabled || uploading;
  const full = blocks.length > 197;

  useEffect(() => {
    onPendingChange(linkOpen);
    return () => onPendingChange(false);
  }, [linkOpen, onPendingChange]);

  useEffect(() => {
    if (!linkOpen || linkType !== "internal" || query.trim().length < 2) {
      setSuggestions([]);
      setSearching(false);
      return;
    }
    const controller = new AbortController();
    setSearching(true);
    const timer = setTimeout(() => {
      void api<Post[]>(
        `/posts/search-suggest?q=${encodeURIComponent(query.trim())}`,
        { signal: controller.signal },
      )
        .then((posts) => {
          if (!controller.signal.aborted) setSuggestions(posts);
        })
        .catch((e: unknown) => {
          if (!controller.signal.aborted)
            setLinkError(e instanceof Error ? e.message : "جستجو انجام نشد.");
        })
        .finally(() => {
          if (!controller.signal.aborted) setSearching(false);
        });
    }, 300);
    return () => {
      clearTimeout(timer);
      controller.abort();
    };
  }, [linkOpen, linkType, query]);

  function caret(): Selection {
    const current = selection.current;
    if (current && blocks[current.index]?.type === "text") return current;
    const index = blocks.findLastIndex((block) => block.type === "text");
    const block = blocks[index];
    const end = block?.type === "text" ? block.text.length : 0;
    return { index, start: end, end };
  }
  function remember(index: number, el: HTMLTextAreaElement) {
    selection.current = {
      index,
      start: el.selectionStart,
      end: el.selectionEnd,
    };
  }
  function update(index: number, block: ContentBlock) {
    onChange(blocks.map((current, i) => (i === index ? block : current)));
  }
  function focusText(index: number, position = 0) {
    requestAnimationFrame(() => {
      const el = inputs.current[index];
      el?.focus();
      el?.setSelectionRange(position, position);
      selection.current = { index, start: position, end: position };
    });
  }
  function insert(block: ContentBlock, at: Selection, replace = false) {
    onChange(
      insertContentBlock(
        blocks,
        at.index,
        at.start,
        replace ? at.end : at.start,
        block,
      ),
    );
    focusText(at.index >= 0 ? at.index + 2 : blocks.length + 1);
  }
  async function upload(e: ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    setError("");
    if (file.size > 5 * 1024 * 1024) {
      setError("حجم تصویر باید حداکثر ۵ مگابایت باشد.");
      e.target.value = "";
      return;
    }
    const at = caret();
    setUploading(true);
    onBusyChange(true);
    try {
      const body = new FormData();
      body.append("image", file);
      const image = await api<{ url: string; width: number }>("/studio/media", {
        method: "POST",
        body,
      });
      insert(
        {
          type: "image",
          src: image.url,
          alt: "",
          width: Math.max(80, Math.min(640, image.width)),
        },
        at,
      );
    } catch (e) {
      setError(e instanceof Error ? e.message : "آپلود تصویر انجام نشد.");
    } finally {
      setUploading(false);
      onBusyChange(false);
      if (fileInput.current) fileInput.current.value = "";
    }
  }
  function openLink(index: number | null = null) {
    const block = index === null ? null : blocks[index];
    const at = caret();
    const selected = blocks[at.index];
    setLinkIndex(index);
    setLinkText(
      block?.type === "link"
        ? block.text
        : selected?.type === "text"
          ? selected.text.slice(at.start, at.end)
          : "",
    );
    setLinkURL(block?.type === "link" ? block.href : "");
    setLinkType(
      block?.type === "link" && !block.href.startsWith("/")
        ? "external"
        : "internal",
    );
    setQuery("");
    setLinkError("");
    linkSelection.current = at;
    setLinkOpen(true);
  }
  function applyLink() {
    const href = linkURL.trim();
    if (
      !linkText.trim() ||
      !validArticleLink(href) ||
      (linkType === "internal" && !href.startsWith("/blog/"))
    ) {
      setLinkError("متن لینک و یک مقاله یا آدرس معتبر http/https انتخاب کنید.");
      return;
    }
    const block: ContentBlock = { type: "link", href, text: linkText };
    if (linkIndex !== null) update(linkIndex, block);
    else insert(block, linkSelection.current || caret(), true);
    setLinkOpen(false);
  }
  function formatSelection(tag: string) {
    const at = caret();
    const block = blocks[at.index];
    if (block?.type !== "text") return;
    const selected = block.text.slice(at.start, at.end) || "متن";
    const fragment = `<${tag}>${selected}</${tag}>`;
    update(at.index, {
      ...block,
      text: block.text.slice(0, at.start) + fragment + block.text.slice(at.end),
    });
    focusText(at.index, at.start + fragment.length);
  }
  function move(index: number, direction: -1 | 1) {
    const target = index + direction;
    if (target < 0 || target >= blocks.length) return;
    const next = [...blocks];
    [next[index], next[target]] = [next[target], next[index]];
    selection.current = null;
    onChange(next);
  }
  return (
    <section className="content-editor" aria-label="ویرایش محتوای مقاله">
      <div className="editor-toolbar">
        <button
          type="button"
          disabled={locked || full || linkOpen}
          onClick={() => fileInput.current?.click()}
        >
          + تصویر داخل مقاله
        </button>
        <button
          type="button"
          disabled={locked || full || linkOpen}
          onClick={() => openLink()}
        >
          + لینک
        </button>
        <button
          type="button"
          disabled={locked || full || linkOpen}
          onClick={() => {
            onChange([...blocks, textBlock("\n\n")]);
            focusText(blocks.length, 2);
          }}
        >
          + بخش متن
        </button>
        {format === "html" &&
          [
            ["p", "پاراگراف"],
            ["h2", "تیتر"],
            ["strong", "پررنگ"],
            ["blockquote", "نقل‌قول"],
            ["code", "کد"],
          ].map(([tag, label]) => (
            <button
              key={tag}
              type="button"
              disabled={locked || linkOpen}
              onClick={() => formatSelection(tag)}
            >
              {label}
            </button>
          ))}
      </div>
      <input
        className="sr-only"
        ref={fileInput}
        type="file"
        accept="image/png,image/jpeg"
        tabIndex={-1}
        aria-label="فایل تصویر داخل مقاله"
        disabled={locked || full}
        onChange={(e) => void upload(e)}
      />
      <p className="muted text-sm">
        در متن کلیک کنید تا تصویر یا لینک همان‌جا اضافه شود. اندازهٔ هر تصویر
        جدا تنظیم می‌شود و نسبت ابعاد آن حفظ می‌شود.
      </p>
      {uploading && <p role="status">در حال آپلود تصویر…</p>}
      {error && (
        <p className="error" role="alert">
          {error}
        </p>
      )}
      {linkOpen && (
        <fieldset
          className="link-editor"
          disabled={locked}
          onKeyDown={(e) => {
            if (e.key === "Enter" && e.target instanceof HTMLInputElement) {
              e.preventDefault();
              applyLink();
            }
          }}
        >
          <legend>{linkIndex === null ? "افزودن لینک" : "ویرایش لینک"}</legend>
          <div className="actions">
            {(["internal", "external"] as const).map((type) => (
              <button
                key={type}
                type="button"
                className={`button ${linkType === type ? "" : "secondary"}`}
                aria-pressed={linkType === type}
                onClick={() => {
                  setLinkType(type);
                  setLinkURL("");
                  setLinkError("");
                }}
              >
                {type === "internal" ? "مقالهٔ این سایت" : "آدرس دیگر"}
              </button>
            ))}
          </div>
          <label>
            متن لینک
            <input
              value={linkText}
              maxLength={500}
              onChange={(e) => setLinkText(e.target.value)}
            />
          </label>
          {linkType === "internal" ? (
            <>
              <label>
                جستجوی مقاله
                <input
                  value={query}
                  placeholder="حداقل دو حرف از عنوان یا متن"
                  onChange={(e) => {
                    setQuery(e.target.value);
                    setLinkError("");
                  }}
                />
              </label>
              {searching && <p role="status">در حال جستجو…</p>}
              {!searching &&
                query.trim().length >= 2 &&
                !suggestions.length && (
                  <p className="muted">مقالهٔ منتشرشده‌ای پیدا نشد.</p>
                )}
              <div className="link-suggestions">
                {suggestions.map((post) => (
                  <button
                    key={post.id}
                    type="button"
                    aria-pressed={linkURL === articlePath(post.slug)}
                    onClick={() => {
                      setLinkURL(articlePath(post.slug));
                      if (!linkText.trim()) setLinkText(post.title);
                    }}
                  >
                    {post.title}
                  </button>
                ))}
              </div>
              {linkURL && (
                <p className="muted" dir="ltr">
                  {linkURL}
                </p>
              )}
            </>
          ) : (
            <label>
              آدرس لینک
              <input
                type="url"
                dir="ltr"
                placeholder="https://example.com"
                value={linkURL}
                maxLength={2048}
                onChange={(e) => setLinkURL(e.target.value)}
              />
            </label>
          )}
          {linkError && (
            <p className="error" role="alert">
              {linkError}
            </p>
          )}
          <div className="actions">
            <button className="button" type="button" onClick={applyLink}>
              {linkIndex === null ? "درج لینک" : "ذخیرهٔ لینک"}
            </button>
            <button
              className="button secondary"
              type="button"
              onClick={() => setLinkOpen(false)}
            >
              انصراف
            </button>
          </div>
        </fieldset>
      )}
      <fieldset className="content-blocks" disabled={locked || linkOpen}>
        <legend>متن و تصاویر مقاله</legend>
        {blocks.map((block, index) => (
          <div
            key={index}
            className={`content-block content-block-${block.type}`}
          >
            {block.type === "text" ? (
              <label>
                متن {format === "html" ? "(HTML)" : ""}
                <textarea
                  ref={(el) => {
                    inputs.current[index] = el;
                  }}
                  rows={Math.max(
                    3,
                    Math.min(12, block.text.split("\n").length + 2),
                  )}
                  value={block.text}
                  maxLength={500000}
                  onSelect={(e) => remember(index, e.currentTarget)}
                  onBlur={(e) => remember(index, e.currentTarget)}
                  onChange={(e) => {
                    remember(index, e.currentTarget);
                    update(index, { ...block, text: e.target.value });
                  }}
                />
              </label>
            ) : block.type === "image" ? (
              <>
                {/* Uploaded media is already bounded and re-encoded by the API; width controls its intrinsic display size. */}
                {/* eslint-disable-next-line @next/next/no-img-element */}
                <img
                  className="inline-image-preview"
                  src={block.src}
                  alt={block.alt || "پیش‌نمایش تصویر داخل مقاله"}
                  width={block.width}
                />
                <label>
                  توضیح تصویر
                  <input
                    value={block.alt}
                    maxLength={300}
                    onChange={(e) =>
                      update(index, { ...block, alt: e.target.value })
                    }
                  />
                </label>
                <ImageSizeControl
                  width={block.width}
                  onChange={(width) => update(index, { ...block, width })}
                />
              </>
            ) : (
              <>
                <p className="editor-link-label">{block.text}</p>
                <p className="muted editor-link-url" dir="ltr">
                  {block.href}
                </p>
                <button
                  type="button"
                  className="text-button"
                  onClick={() => openLink(index)}
                >
                  ویرایش لینک
                </button>
              </>
            )}
            <div className="block-actions">
              <button
                type="button"
                disabled={index === 0}
                onClick={() => move(index, -1)}
                aria-label={`بخش ${index + 1}: انتقال به بالا`}
              >
                ↑ بالا
              </button>
              <button
                type="button"
                disabled={index === blocks.length - 1}
                onClick={() => move(index, 1)}
                aria-label={`بخش ${index + 1}: انتقال به پایین`}
              >
                ↓ پایین
              </button>
              <button
                type="button"
                disabled={blocks.length === 1 && block.type === "text"}
                onClick={() => {
                  onChange(removeContentBlock(blocks, index));
                  selection.current = null;
                }}
              >
                حذف{" "}
                {block.type === "image"
                  ? "تصویر"
                  : block.type === "link"
                    ? "لینک"
                    : "بخش متن"}
              </button>
            </div>
          </div>
        ))}
      </fieldset>
      <details className="article-preview">
        <summary>پیش‌نمایش متن و تصاویر</summary>
        <div
          className={`prose-content${format === "plain_text" ? " plain-text" : ""}`}
          dangerouslySetInnerHTML={{
            __html: safeHTML(renderContentBlocks(format, blocks)),
          }}
        />
      </details>
      <p className="muted text-sm">
        {format === "plain_text"
          ? "متن ساده، فاصله‌ها و خط‌ها را حفظ می‌کند؛ تصویر و لینک با دکمه‌های بالا اضافه می‌شوند."
          : "HTML هنگام نمایش پاک‌سازی می‌شود. تصویر و لینک را با دکمه‌های بالا اضافه کنید یا تگ‌هایشان را بنویسید."}{" "}
        PNG یا JPEG تا ۵ مگابایت. تغییر قالب، تصاویر و لینک‌ها را حفظ می‌کند.
      </p>
    </section>
  );
}
