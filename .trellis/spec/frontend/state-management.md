# Frontend State Management

The frontend uses Pinia setup stores for shared app state, auth, settings, subscriptions, payments, announcements, and admin workflows.

## Store Shape

- Define stores with `defineStore('name', () => { ... })`.
- Use `ref` for mutable state and `computed` for derived state.
- Return state, computed values, and actions explicitly.
- Expose `readonly(...)` for state that consumers should observe but not mutate directly, as in `auth.ts` for `runMode` and pending auth session.
- Keep module-level constants for storage keys, intervals, and timing buffers.

## Auth Store

- `stores/auth.ts` is the source of truth for logged-in user, access token, refresh token, expiry timestamp, run mode, pending auth sessions, and auth lifecycle actions.
- Persist auth state in localStorage keys `auth_token`, `auth_user`, `refresh_token`, `token_expires_at`, and `pending_auth_session`.
- `checkAuth()` restores localStorage state, refreshes user data asynchronously, starts user auto-refresh, and schedules proactive token refresh when possible.
- Clear token refresh and auto-refresh timers when auth state is cleared.
- Preserve pending auth sessions intentionally during failed registration or auth-flow retries.

## App Store

- `stores/app.ts` owns global UI state: sidebar, mobile sidebar, global loading count, toasts, public settings cache, version cache, and injected config.
- Use `setLoading`'s counter semantics for overlapping operations rather than directly toggling `loading`.
- Use `showSuccess`, `showError`, `showInfo`, and `showWarning` for consistent toast durations.
- Public settings should be initialized from `window.__APP_CONFIG__` before mount when available, then refreshed through the API as needed.

## Server State

- Use stores for server state that is shared across views, persisted, polled, or reused after navigation, such as auth, subscriptions, announcements, admin compliance, and public settings.
- Use component-local state or `useTableLoader` for page-specific lists and filters.
- Use API wrapper modules as the network boundary. Stores should not duplicate Axios client behavior.

## Router And Store Interaction

- Route guards in `router/index.ts` use auth/app/admin settings stores for authentication, admin access, setup state, backend mode, simple mode, route prefetch, and loading progress.
- Route meta fields are typed in `router/meta.d.ts`; add meta there when introducing a new cross-route property.
- `App.vue` reacts to auth state to start/stop subscription polling, announcement fetches, and admin compliance checks.

## Common Mistakes

- Do not store short-lived form fields globally unless several routes need them.
- Do not mutate another store's state directly from a component; call store actions.
- Do not create duplicate public settings fetches; use `appStore.fetchPublicSettings()` cache semantics.
- Do not leave polling intervals active after logout or unmount.
