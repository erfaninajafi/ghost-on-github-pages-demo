import { test } from "node:test";
import assert from "node:assert/strict";
import { signupPasswordError } from "../lib/password";
test("signup requires matching passwords and enforces bcrypt's byte limit for Persian", () => {
  assert.equal(
    signupPasswordError("correct-password-123", "correct-password-123"),
    null,
  );
  assert.ok(signupPasswordError("correct-password-123", "different-password"));
  assert.ok(signupPasswordError("", ""));
  assert.ok(signupPasswordError("ی".repeat(37), "ی".repeat(37)));
  assert.equal(signupPasswordError("ی".repeat(36), "ی".repeat(36)), null);
});
