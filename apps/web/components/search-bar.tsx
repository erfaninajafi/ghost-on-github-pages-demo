"use client";
import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import { api } from "@/lib/api";
import type { Post } from "@/lib/types";
export function SearchBar() {
  const [q, setQ] = useState("");
  const [items, setItems] = useState<Post[]>([]);
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const root = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const close = (e: MouseEvent) => {
      if (!root.current?.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", close);
    return () => document.removeEventListener("mousedown", close);
  }, []);
  useEffect(() => {
    const controller = new AbortController();
    setItems([]);
    setError("");
    if (q.trim().length < 2) {
      setBusy(false);
      return () => controller.abort();
    }
    setBusy(true);
    const timer = setTimeout(() => {
      api<Post[]>(`/posts/search-suggest?q=${encodeURIComponent(q.trim())}`, {
        signal: controller.signal,
      })
        .then(setItems)
        .catch((e: unknown) => {
          if (!controller.signal.aborted)
            setError(e instanceof Error ? e.message : "خطای جستجو");
        })
        .finally(() => {
          if (!controller.signal.aborted) setBusy(false);
        });
    }, 250);
    return () => {
      clearTimeout(timer);
      controller.abort();
    };
  }, [q]);
  return (
    <div
      className="search"
      ref={root}
      onKeyDown={(e) => {
        if (e.key === "Escape") setOpen(false);
      }}
    >
      <span aria-hidden="true">⌕</span>
      <input
        aria-label="جستجوی مقاله"
        aria-controls="search-results"
        aria-expanded={open && q.length >= 2}
        type="search"
        placeholder="چه چیزی می‌خواهید یاد بگیرید؟"
        value={q}
        maxLength={100}
        onFocus={() => setOpen(true)}
        onChange={(e) => {
          setQ(e.target.value);
          setOpen(true);
        }}
      />
      {open && q.trim().length >= 2 && (
        <div id="search-results" className="search-results">
          <p role="status">
            {busy
              ? "در حال جستجو…"
              : error ||
                (!items.length
                  ? "مقاله‌ای پیدا نشد."
                  : `${items.length} مقاله پیدا شد`)}
          </p>
          {items.map((p) => (
            <Link
              key={p.id}
              href={`/blog/${p.slug}`}
              onClick={() => setOpen(false)}
            >
              {p.title}
              <small>{p.reading_time_minutes} دقیقه مطالعه</small>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}
