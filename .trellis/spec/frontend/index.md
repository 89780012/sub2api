# Frontend Development Guidelines

Source-backed conventions for the Vue frontend in `frontend/`.

## Guidelines Index

| Guide | Purpose | Status |
|-------|---------|--------|
| [Directory Structure](./directory-structure.md) | Where frontend code belongs and how modules are grouped | Complete |
| [Component Guidelines](./component-guidelines.md) | Vue component, slot, styling, and responsive patterns | Complete |
| [Hook Guidelines](./hook-guidelines.md) | Composable naming, cancellation, debounce, and cleanup patterns | Complete |
| [State Management](./state-management.md) | Pinia stores, persistence, global UI state, and server state | Complete |
| [Type Safety](./type-safety.md) | TypeScript contracts, API wrappers, and typed props/events | Complete |
| [Quality Guidelines](./quality-guidelines.md) | Build, lint, tests, accessibility, and review checks | Complete |

## Pre-Development Checklist

Read these before frontend changes:

- Always read [Directory Structure](./directory-structure.md).
- Read [Component Guidelines](./component-guidelines.md) for `.vue` components, layouts, tables, dialogs, forms, or visual states.
- Read [Hook Guidelines](./hook-guidelines.md) before adding or changing composables.
- Read [State Management](./state-management.md) for Pinia stores, persistence, app config, auth, settings, polling, or shared server data.
- Read [Type Safety](./type-safety.md) for API wrappers, shared types, route meta, and request/response shapes.
- Read [Quality Guidelines](./quality-guidelines.md) before testing or verification.

## Main Verification Commands

- `cd frontend && pnpm run typecheck`
- `cd frontend && pnpm run lint:check`
- `cd frontend && pnpm run test:run`
- `cd frontend && pnpm run build`
