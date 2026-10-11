export function signupPasswordError(
  password: unknown,
  confirmation: unknown,
): string | null {
  if (
    typeof password !== "string" ||
    typeof confirmation !== "string" ||
    password !== confirmation
  )
    return "رمز عبور و تکرار آن یکسان نیستند.";
  const bytes = new TextEncoder().encode(password).length;
  if (password.length < 12 || bytes > 72)
    return "رمز عبور باید حداقل ۱۲ کاراکتر و حداکثر ۷۲ بایت باشد.";
  return null;
}
