# Frontend Directory Structure

The frontend is a Vue 3, Vite, TypeScript, Pinia, Vue Router, vue-i18n, TailwindCSS application in `frontend/`.

## Source Layout

- `frontend/src/main.ts` bootstraps theme class, Pinia, injected public config, i18n, router, and mounts after `router.isReady()`.
- `frontend/src/App.vue` owns global shell components, setup-status checks, public settings refresh, document title/favicon updates, auth-driven polling, announcements, and admin-compliance dialog wiring.
- `frontend/src/router/` owns route definitions, guards, route titles, setup redirects, route meta typing, and router tests.
- `frontend/src/api/` contains typed API wrapper modules. `api/client.ts` owns the shared Axios client, auth headers, locale/timezone injection, response unwrapping, token refresh, and structured error normalization.
- `frontend/src/stores/` contains Pinia setup stores. `auth.ts`, `app.ts`, `payment.ts`, `subscriptions.ts`, and admin stores manage shared state and persistence.
- `frontend/src/views/` contains route-level page components, grouped by area: `auth`, `user`, `admin`, `setup`, and `public`.
- `frontend/src/components/` contains reusable UI and feature components. Shared components live under `components/common`; layout and auth-specific pieces have their own folders.
- `frontend/src/composables/` contains reusable Composition API logic such as table loading, debounced search, route prefetch, OAuth flows, clipboard, forms, and persisted page size.
- `frontend/src/types/` contains shared TypeScript interfaces and domain types used across stores, API wrappers, and components.
- `frontend/src/utils/` contains pure utilities and small domain helpers, with colocated tests in `utils/__tests__`.
- `frontend/src/i18n/` contains localization setup and locale dictionaries.
- `frontend/src/assets/` and `frontend/src/styles/` hold static assets and shared CSS.

## Module Rules

- Put new pages in `views/<area>/` and register them lazily in `router/index.ts`.
- Put reusable, cross-page components in `components/common` only when they are truly generic. Keep feature-specific components close to their feature folder.
- Put network calls in `api/` wrappers rather than directly in components. Components and stores should call wrapper functions.
- Put cross-component state in Pinia stores only when it is shared, persisted, or long-lived. Keep local UI state inside components otherwise.
- Put reusable stateful logic in `composables/use*.ts`; pure functions belong in `utils/`.
- Put shared request/response/domain types in `types/index.ts` or a focused file such as `types/payment.ts`; avoid redefining backend contract shapes in individual components.

## Naming Patterns

- Vue components use PascalCase filenames: `DataTable.vue`, `ConfirmDialog.vue`, `PaymentView.vue`.
- Composables use `useX.ts`: `useTableLoader.ts`, `useKeyedDebouncedSearch.ts`, `usePersistedPageSize.ts`.
- Stores use lower camel or feature names: `auth.ts`, `adminSettings.ts`, `subscriptions.ts`.
- Tests use `.spec.ts` and live in `__tests__` directories near the code they cover.
- API modules are grouped by backend area, including nested admin modules under `api/admin/`.

## Build And Runtime Notes

- Vite builds into `../backend/internal/web/dist`, which is embedded by the backend for production.
- The dev server proxies `/api`, `/v1`, and `/setup` to `VITE_DEV_PROXY_TARGET` or `http://localhost:8080`.
- Public settings can be injected into `window.__APP_CONFIG__` by the backend and by the Vite dev plugin to prevent UI flash.
- The app uses `@` as an alias for `frontend/src`.
