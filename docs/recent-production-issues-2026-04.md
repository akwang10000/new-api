# Recent Production Issues and Fixes

## Scope

This note records the recent issues addressed in the current deployment batch so the next rollout and regression checks are easier to follow.

## 1. Chatwoot customer support widget reset during page switches

### Symptoms
- The custom support entry was visible, but navigating between main-site pages could reset the Chatwoot session.
- In some cases the support window could no longer be reopened from the custom launcher.

### Root cause
- The frontend loaded Chatwoot user identity asynchronously from `/api/user/self`.
- During route changes, `UserContext` could briefly be empty before local state was restored.
- The widget effect treated that temporary empty state as a real logout and called `window.$chatwoot.reset()` too early.
- The custom launcher also called `window.$chatwoot.toggle('open')`, which did not reopen the hidden Chatwoot widget reliably in the current integration.

### Fix
- Added an explicit `isUserResolved` state in `web/src/components/common/ChatwootWidget.jsx` so Chatwoot reset only happens after user resolution completes.
- Kept the official Chatwoot bubble hidden and used the custom launcher button.
- Changed the launcher action to `window.$chatwoot.toggle('toggle')`.
- Added a frontend regression test in `web/src/components/common/ChatwootWidget.test.jsx` covering:
  - no premature reset while logged-in user data is still loading
  - custom launcher opens the widget correctly

### Verification
- `bunx vitest run --config "web/vitest.config.js" "web/src/components/common/ChatwootWidget.test.jsx"`

## 2. Registration flow did not respect email domain whitelist correctly

### Symptoms
- When email domain whitelist was enabled without email verification, the frontend registration form could still omit the email field.
- Server-side validation and verification-code flow messaging were inconsistent.

### Root cause
- The frontend only displayed the email field when `email_verification` was enabled.
- Backend whitelist checks were duplicated and not reused consistently between registration and verification-code sending.

### Fix
- Exposed `email_domain_restriction` in `/api/status`.
- Updated `web/src/components/auth/RegisterForm.jsx` to require and show the email field when either email verification or email domain restriction is enabled.
- Centralized whitelist validation in `controller/user.go` with `isEmailDomainWhitelisted`.
- Reused the same whitelist error message for registration and verification email sending.

### Verification
- Covered by backend test suite run below.

## 3. Invitation / affiliate code should be paused during current shutdown window

### Symptoms
- OAuth and registration paths could still read and persist inviter affiliate codes while invitation link sharing was supposed to be paused.

### Root cause
- The OAuth state flow and registration flow still accepted and resolved affiliate codes unconditionally.

### Fix
- Added guards around affiliate-code capture and inviter resolution in:
  - `controller/oauth.go`
  - `controller/user.go`
- Hid `aff_code` from self-response payloads when invitation link sharing is paused.
- Added coverage in `controller/invitation_shutdown_test.go`.

### Verification
- Covered by backend test suite run below.

## 4. Codex OAuth error handling was too opaque

### Symptoms
- Codex OAuth startup / completion failures returned generic frontend errors.
- Session save failures were ignored.
- Upstream token exchange / refresh failures did not surface useful provider messages.

### Root cause
- Session persistence errors were discarded.
- OAuth exchange and refresh handlers only returned generic status-based errors.
- Frontend modal replaced server messages with generic text too early.

### Fix
- Checked and logged session save / cleanup failures in `controller/codex_oauth.go`.
- Surfaced upstream `error` / `error_description` from Codex OAuth responses in `service/codex_oauth.go`.
- Preserved backend error messages in `web/src/components/table/channels/modals/CodexOAuthModal.jsx`.

### Verification
- Covered by backend test suite run below.

## Deployment verification used for this batch

### Backend
```bash
go test ./...
```

### Frontend regression
```bash
bunx vitest run --config "web/vitest.config.js" "web/src/components/common/ChatwootWidget.test.jsx"
```

## Deployment notes

- Production runbook: `docs/deployment-runbook.md`
- Current production environment is documented there as an ARM64 Docker deployment.
- Always build the production image with explicit ARM64 build args on the server.
