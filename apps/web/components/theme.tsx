"use client";
import { ThemeProvider, useTheme } from "next-themes";
import { useEffect, useState, type ReactNode } from "react";
export function Theme({ children }: { children: ReactNode }) {
  return (
    <ThemeProvider attribute="class" defaultTheme="system" enableSystem>
      {children}
    </ThemeProvider>
  );
}
export function ThemeToggle() {
  const { resolvedTheme, setTheme } = useTheme();
  const [ready, setReady] = useState(false);
  useEffect(() => setReady(true), []);
  return (
    <button
      className="icon-button"
      aria-label="تغییر تم روشن و تاریک"
      onClick={() => setTheme(resolvedTheme === "dark" ? "light" : "dark")}
    >
      {ready && resolvedTheme === "dark" ? "☀" : "☾"}
    </button>
  );
}
