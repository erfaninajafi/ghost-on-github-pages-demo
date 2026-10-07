import Link from "next/link";
export default function NotFound() {
  return (
    <section className="container empty-state">
      <p className="eyebrow">۴۰۴</p>
      <h1>این صفحه پیدا نشد.</h1>
      <p>ممکن است مقاله هنوز منتشر نشده باشد یا نشانی آن تغییر کرده باشد.</p>
      <Link href="/" className="button">
        بازگشت به مجله
      </Link>
    </section>
  );
}
