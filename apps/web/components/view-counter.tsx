"use client";
import { useEffect } from "react";
export function ViewCounter({ slug }: { slug: string }) {
  useEffect(() => {
    const controller = new AbortController();
    const timer = setTimeout(() => {
      void fetch(`/api/v1/posts/${encodeURIComponent(slug)}`, {
        cache: "no-store",
        signal: controller.signal,
      }).catch(() => {
        /* A failed analytics request must not interrupt reading. */
      });
    }, 800);
    return () => {
      clearTimeout(timer);
      controller.abort();
    };
  }, [slug]);
  return null;
}
