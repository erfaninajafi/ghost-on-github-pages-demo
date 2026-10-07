export default function Loading() {
  return (
    <div className="container section-space" role="status" aria-live="polite">
      <div className="skeleton h-72" />
      <p>در حال بارگذاری مجله…</p>
    </div>
  );
}
