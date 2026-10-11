"use client";
import Image from "next/image";
import { useRef, useState, type ChangeEvent } from "react";
import { api } from "@/lib/api";

export function CoverUpload({
  value,
  onChange,
  disabled,
  onBusyChange,
}: {
  value: string | null;
  onChange: (url: string | null) => void;
  disabled: boolean;
  onBusyChange: (busy: boolean) => void;
}) {
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState("");
  const input = useRef<HTMLInputElement>(null);
  async function upload(e: ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    if (file.size > 5 * 1024 * 1024) {
      setError("حجم تصویر باید حداکثر ۵ مگابایت باشد.");
      e.target.value = "";
      return;
    }
    setUploading(true);
    onBusyChange(true);
    setError("");
    try {
      const body = new FormData();
      body.append("image", file);
      const image = await api<{ url: string }>("/studio/media", {
        method: "POST",
        body,
      });
      onChange(image.url);
    } catch (e) {
      setError(e instanceof Error ? e.message : "آپلود انجام نشد.");
    } finally {
      setUploading(false);
      onBusyChange(false);
      if (input.current) input.current.value = "";
    }
  }
  return (
    <fieldset className="cover-upload">
      <legend>تصویر جلد مقاله (اختیاری)</legend>
      <p className="muted text-sm">
        PNG یا JPEG، حداکثر ۵ مگابایت و ۱۲ میلیون پیکسل. تصویر پس از ذخیرهٔ
        مقاله روی جلد و صفحهٔ آن نمایش داده می‌شود.
      </p>
      {value && (
        <Image
          className="cover-preview"
          src={value}
          alt="پیش‌نمایش تصویر جلد"
          width={800}
          height={450}
          sizes="(max-width: 900px) 100vw, 700px"
        />
      )}
      <label>
        انتخاب تصویر
        <input
          ref={input}
          type="file"
          accept="image/png,image/jpeg"
          disabled={disabled || uploading}
          onChange={(e) => void upload(e)}
        />
      </label>
      {uploading && <p role="status">در حال آپلود تصویر…</p>}
      {value && (
        <button
          className="text-button"
          type="button"
          disabled={disabled || uploading}
          onClick={() => {
            onChange(null);
            setError("");
          }}
        >
          برداشتن تصویر جلد
        </button>
      )}
      {error && (
        <p className="error" role="alert">
          {error}
        </p>
      )}
    </fieldset>
  );
}
