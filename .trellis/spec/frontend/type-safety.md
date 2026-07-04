# Frontend Type Safety

The frontend runs TypeScript in strict mode with `noUnusedLocals`, `noUnusedParameters`, and Vue type checking.

## Shared Types

- Put cross-feature domain types in `frontend/src/types/index.ts`; use focused files such as `types/payment.ts` when a domain grows large.
- Keep API response interfaces aligned with backend JSON field names, which mostly use snake_case.
- Reuse shared types in API wrappers, stores, and components instead of redefining local copies.
- Export narrow literal unions for known values, such as user roles, group platforms, announcement statuses, payment types, and route/store state modes.

## API Wrappers

- Use `apiClient` from `frontend/src/api/client.ts` for all standard backend requests.
- API wrapper functions should declare Promise return types and unwrap `response.data`, because the Axios interceptor already unwraps `{ code, message, data }`.
- Include `AbortSignal` support in list/search wrappers that are called by cancellable loaders. `api/admin/accounts.ts` is the reference with `options?: { signal?: AbortSignal }`.
- For special HTTP behavior such as ETag `304`, use typed result objects and local `validateStatus`, as in `listWithEtag`.
- Return typed structured errors from utilities or let the shared interceptor normalize errors with `status`, `code`, `reason`, `message`, and `metadata`.

## Components

- Use typed props and emits. `DataTable.vue` defines a typed `sort` emit and a `Props` interface.
- Prefer `unknown` over `any` for new untrusted values, then narrow with runtime checks. The project allows `any` in ESLint for compatibility, but new code should avoid it when practical.
- Use `Record<string, unknown>` for arbitrary JSON payloads and provider-specific metadata.
- Keep template data normalized in computed properties so templates do not need unsafe casts.

## Runtime Validation And Normalization

- This project does not standardize on a runtime schema library such as Zod. Use local type guards and normalization helpers.
- Normalize localStorage and injected config values before use. See `utils/tablePreferences.ts`.
- Validate route query and payment resume data through helper functions rather than trusting raw `route.query`.
- Validate API error shapes before reading nested fields. `api/client.ts` checks that response data is an object before accessing normalized error fields.

## Route Meta

- Add or update route meta typing in `frontend/src/router/meta.d.ts` when adding route-level flags.
- Use `titleKey` and `descriptionKey` when the page participates in localization.
- Keep route components lazy-loaded with `component: () => import(...)`.

## Common Mistakes

- Do not treat `axios` responses as raw backend envelopes after they pass through `apiClient`; success `data` has already been unwrapped.
- Do not cast backend payloads in templates to get around missing types; update shared types or API wrapper return types.
- Do not assume localStorage JSON parses successfully.
- Do not add unused parameters without a leading underscore when strict TypeScript will reject them.
