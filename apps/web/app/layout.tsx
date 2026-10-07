import type { Metadata } from "next";
import Link from "next/link";
import "@fontsource-variable/vazirmatn";
import "./globals.css";
import { Theme, ThemeToggle } from "@/components/theme";
import { SearchBar } from "@/components/search-bar";
import { siteURL } from "@/lib/api";
export const metadata: Metadata = {
  metadataBase: new URL(siteURL),
  title: {
    default: "وبلاگ ExpertsAcademy | از دانش تا تخصص",
    template: "%s | ExpertsAcademy",
  },
  description:
    "یادداشت‌ها، تجربه‌ها و آموزش‌های متخصصان اکسپرتس آکادمی؛ برای یادگیری عمیق و رشد حرفه‌ای.",
  alternates: { types: { "application/rss+xml": "/rss.xml" } },
  openGraph: {
    type: "website",
    locale: "fa_IR",
    siteName: "ExpertsAcademy",
    images: [
      { url: "/og.png", width: 1200, height: 630, alt: "ExpertsAcademy" },
    ],
  },
  robots: { index: true, follow: true },
};
export default function Layout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="fa" dir="rtl" suppressHydrationWarning>
      <body>
        <Theme>
          <a href="#main" className="skip-link">
            رفتن به محتوای اصلی
          </a>
          <header className="navbar">
            <div className="container nav-inner">
              <Link href="/" className="brand" aria-label="خانه ExpertsAcademy">
                <span className="brand-symbol">
                  E<span>✦</span>
                </span>
                <span dir="ltr">
                  Experts<span>Academy</span>
                  <small>مجلهٔ دانش و تجربه</small>
                </span>
              </Link>
              <nav aria-label="ناوبری اصلی">
                <Link href="/">مجله</Link>
                <Link href="/#articles">تازه‌ترین‌ها</Link>
                <Link href="/studio">پنل نویسندگان</Link>
              </nav>
              <ThemeToggle />
            </div>
          </header>
          <div className="container search-row">
            <span className="search-caption">یادگیری، یک مسیر همیشگی است.</span>
            <SearchBar />
          </div>
          <main id="main">{children}</main>
          <footer>
            <div className="container footer-inner">
              <div className="brand-text" dir="ltr">
                Experts<span>Academy</span>
              </div>
              <p>دانش را به تجربه، و تجربه را به تخصص تبدیل کنیم.</p>
              <Link href="/rss.xml">خوراک RSS ↗</Link>
            </div>
          </footer>
        </Theme>
      </body>
    </html>
  );
}
