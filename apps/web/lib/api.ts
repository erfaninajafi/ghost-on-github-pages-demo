import type { Post, PostPage, Taxonomy } from "./types";
export const siteURL =
  process.env.NEXT_PUBLIC_SITE_URL || "https://blog.expertsacademy.ir";
export class APIError extends Error {
  constructor(
    public readonly status: number,
    message: string,
  ) {
    super(message);
  }
}
// Server requests use the private container network. Browser requests stay same-origin.
export async function api<T>(
  path: string,
  init: RequestInit & { next?: { revalidate?: number; tags?: string[] } } = {},
): Promise<T> {
  const base =
    typeof window === "undefined"
      ? process.env.API_INTERNAL_URL || "http://localhost:8080"
      : "";
  const res = await fetch(`${base}/api/v1${path}`, {
    ...init,
    headers: {
      ...(init.body ? { "Content-Type": "application/json" } : {}),
      ...init.headers,
    },
    signal: init.signal || AbortSignal.timeout(10000),
  });
  if (!res.ok) {
    let message = "ارتباط با سرور برقرار نشد.";
    try {
      const body: unknown = await res.json();
      if (typeof body === "object" && body !== null && "error" in body) {
        const error = body.error;
        if (
          typeof error === "object" &&
          error !== null &&
          "message" in error &&
          typeof error.message === "string"
        )
          message = error.message;
      }
    } catch {
      /* Non-JSON upstream failures use the safe default message. */
    }
    throw new APIError(res.status, message);
  }
  if (res.status === 204) return undefined as T;
  return res.json() as Promise<T>;
}
export const getPosts = (params: URLSearchParams = new URLSearchParams()) =>
  api<PostPage>(`/posts?${params}`, { next: { revalidate: 60 } });
export const getPopular = () =>
  api<Post[]>("/posts/popular", { next: { revalidate: 30 } });
export const getTaxonomy = () =>
  api<Taxonomy>("/taxonomy", { next: { revalidate: 60 } });
export const getPost = (slug: string) =>
  api<Post>(`/posts/${encodeURIComponent(slug)}?count_view=false`, {
    next: { revalidate: 60 },
  });
export function date(value: string | null) {
  return value
    ? new Intl.DateTimeFormat("fa-IR", { dateStyle: "medium" }).format(
        new Date(value),
      )
    : "پیش‌نویس";
}
export const number = (value: number) =>
  new Intl.NumberFormat("fa-IR").format(value);
