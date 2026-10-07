"use client";
export default function ErrorPage({ reset }: { reset: () => void }) {
  return (
    <section className="container empty-state">
      <h1>دریافت اطلاعات انجام نشد.</h1>
      <p>لطفاً چند لحظه بعد دوباره تلاش کنید.</p>
      <button className="button" onClick={reset}>
        تلاش مجدد
      </button>
    </section>
  );
}
