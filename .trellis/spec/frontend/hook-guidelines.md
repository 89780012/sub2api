# Frontend Composable Guidelines

Reusable Composition API logic lives in `frontend/src/composables` and uses `use*` naming.

## Naming And Scope

- Name composables `useFeature.ts`, such as `useTableLoader`, `useForm`, `useKeyedDebouncedSearch`, and `usePersistedPageSize`.
- Keep composables focused on reusable behavior. If logic is specific to one view and not reused, keep it in the view or extract a local helper file.
- Return refs, reactive state, and action functions with stable names; avoid hidden global state unless persistence is the purpose.

## Async Loading And Cancellation

- For paginated tables, prefer `useTableLoader`. It owns `items`, `loading`, `params`, `pagination`, `load`, `reload`, debounced reload, page changes, page-size persistence, and request cancellation.
- When a composable starts fetches that may overlap, use `AbortController` and ignore cancellation errors. `useTableLoader` treats `AbortError`, `ERR_CANCELED`, and `CanceledError` as silent.
- Abort in-flight requests on unmount.
- Protect against stale responses when several keyed searches can run at once. `useKeyedDebouncedSearch` uses per-key timers, abort controllers, and version counters.

## Persistence

- Keep storage access guarded by `typeof window !== 'undefined'`.
- Normalize persisted values before use. `usePersistedPageSize` delegates to `utils/tablePreferences.ts` so invalid localStorage and injected config values cannot break pagination.
- Catch storage failures and log a warning rather than failing rendering.

## Form Submission

- Use `useForm` for simple submit flows that need loading state, success toast, error toast, and rethrowing so the component can show local validation state.
- For complex forms with multi-step recovery, explicit route redirects, or provider-specific branching, keep the workflow in a view-specific function and use store/API helpers directly.

## Cleanup Rules

- If a composable registers timers, controllers, listeners, observers, or external subscriptions, it must expose a clear method and also clean up automatically when called inside a component instance.
- `useKeyedDebouncedSearch` checks `getCurrentInstance()` before installing `onUnmounted`, which allows it to be used safely in tests or non-component contexts.

## Common Mistakes

- Do not put API endpoint strings in composables when an `api/` wrapper already exists.
- Do not swallow non-cancellation errors in data loaders; rethrow so pages and tests can react.
- Do not persist raw user input without normalization.
- Do not share one debounce timer across independent keyed controls when each row/filter needs isolated behavior.
