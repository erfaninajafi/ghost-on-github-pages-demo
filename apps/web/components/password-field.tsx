"use client";
import { useState } from "react";

export function PasswordField({
  name,
  label,
  signup = false,
  disabled = false,
}: {
  name: string;
  label: string;
  signup?: boolean;
  disabled?: boolean;
}) {
  const [visible, setVisible] = useState(false);
  return (
    <div className="password-field">
      <label htmlFor={name}>{label}</label>
      <div className="password-input">
        <input
          id={name}
          name={name}
          type={visible ? "text" : "password"}
          required
          minLength={signup ? 12 : 1}
          maxLength={72}
          autoComplete={signup ? "new-password" : "current-password"}
          dir="ltr"
          disabled={disabled}
        />
        <button
          type="button"
          className="password-toggle"
          aria-label={`${visible ? "پنهان کردن" : "نمایش"} ${label}`}
          aria-pressed={visible}
          aria-controls={name}
          disabled={disabled}
          onClick={() => setVisible(!visible)}
        >
          <svg
            width="22"
            height="22"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="1.7"
            aria-hidden="true"
          >
            <path d="M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7S2 12 2 12Z" />
            <circle cx="12" cy="12" r="3" />
            {visible && <path d="m3 3 18 18" />}
          </svg>
        </button>
      </div>
    </div>
  );
}
