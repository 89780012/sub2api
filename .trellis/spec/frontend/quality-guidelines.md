# Frontend Quality Guidelines

Frontend quality is enforced with Vue type checking, ESLint, Vitest, and build verification.

## Verification Commands

- Type check: `cd frontend && pnpm run typecheck`
- Lint without modifying files: `cd frontend && pnpm run lint:check`
- Lint with fixes when appropriate: `cd frontend && pnpm run lint`
- Unit/integration tests: `cd frontend && pnpm run test:run`
- Build: `cd frontend && pnpm run build`

## Test Patterns

- Tests use Vitest and Vue Test Utils. Global browser shims live in `frontend/src/__tests__/setup.ts`.
- Put tests near the code under `__tests__` directories, such as `api/__tests__`, `stores/__tests__`, `composables/__tests__`, `router/__tests__`, `views/**/__tests__`, and `utils/__tests__`.
- Mock network/API modules with `vi.mock` before importing modules under test when the module has import-time state.
- Use `setActivePinia(createPinia())` for store tests.
- Use fake timers for auth refresh, debounce, and polling tests.
- Router guard tests can extract and simulate guard logic when direct router setup would be too heavy, as in `router/__tests__/guards.spec.ts`.

## Required Checks Before Frontend Changes Land

- `pnpm run typecheck` passes for TypeScript and `.vue` files.
- API contract changes update API wrappers, shared types, stores, route logic, and tests together.
- Pages with auth/admin/simple/backend-mode behavior have route guard or store tests when behavior changes.
- Components with async loading handle loading, empty, error, and cancellation states where relevant.
- UI changes preserve dark-mode classes when editing shared components or admin/user views.
- Public text uses `vue-i18n` when the surrounding feature is localized.

## Lint And Type Rules

- ESLint extends `plugin:vue/vue3-essential` and `@typescript-eslint/recommended`.
- `@typescript-eslint/no-explicit-any` is disabled for legacy compatibility, but prefer `unknown`, generics, and shared types for new code.
- `noUnusedLocals` and `noUnusedParameters` are enforced by TypeScript; prefix intentionally unused variables with `_`.
- `vue/multi-word-component-names` is disabled, so route and common component names can follow existing concise names.

## Accessibility And UX Review

- Forms need labels, disabled/loading states, and clear errors.
- Tables need stable row keys and empty/loading states.
- Dialogs need clear cancel/close behavior.
- Navigation changes must preserve route title resolution and auth redirect behavior.
- Long-running actions should use store toasts or local inline feedback.

## Common Mistakes

- Do not rely on `pnpm run lint` as the only lint check in automation because it mutates files with `--fix`; use `lint:check` for verification.
- Do not add direct `axios` calls in views or stores when an API wrapper belongs in `src/api`.
- Do not introduce route-level flags without updating route meta typing and guard tests.
- Do not ignore aborted requests as failures in loaders; cancellation should be silent and stale responses should not overwrite current state.
