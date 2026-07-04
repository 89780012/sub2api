# Frontend Component Guidelines

Components use Vue 3 Single File Components, `<script setup lang="ts">`, Composition API, Tailwind utility classes, and explicit responsive states.

## Component Shape

- Prefer `<script setup lang="ts">` for new components.
- Define props with `defineProps` and defaults with `withDefaults`, as in `components/common/DataTable.vue`.
- Define emits with typed `defineEmits`, as `DataTable.vue` does for sort events.
- Keep derived display state in `computed` and side effects in `watch`, `onMounted`, and `onUnmounted`.
- Clean up window listeners, observers, timers, and document listeners in `onUnmounted` or `onBeforeUnmount`.

## Shared Component Patterns

- Use `components/common/DataTable.vue` for tabular data when possible. It already supports loading skeletons, empty slots, responsive mobile card rendering, virtual scrolling, sticky columns, client/server sorting, row keys, and persisted sort state.
- Use `components/common/BaseDialog.vue` and `ConfirmDialog.vue` for modal flows rather than building separate dialog shells.
- Use `components/icons/Icon.vue` for the local icon set. Add a named icon path there when the project needs a reusable icon.
- Use slots for cell customization and actions in generic components. `DataTable.vue` uses `cell-{key}`, `header-{key}`, and `empty` slots.

## Styling

- Use Tailwind utility classes and the local theme tokens from `tailwind.config.js`: `primary`, `accent`, and `dark`.
- Support dark mode with `dark:` classes when a component has visible colors.
- Preserve the existing admin/tooling style: dense, functional layouts with clear tables, dialogs, filters, and forms.
- For responsive tables, follow `DataTable.vue`: card-style mobile rendering below 768px and full table behavior on desktop.
- For long-running/loading states, provide stable skeletons or disabled states so layout does not jump.

## Accessibility And Interaction

- Use semantic form controls with labels and button disabled states.
- Dialogs must have a clear close/cancel path and should emit state changes rather than mutating parent state directly.
- Action-heavy table cells should use buttons/links with stable row keys.
- Route pages should not rely only on color to communicate state; include text, icons, or disabled states.

## API And Store Interaction

- Components should call API wrappers or store actions, not inline Axios calls.
- Use `appStore.showSuccess`, `showError`, `showInfo`, or `showWarning` for user feedback.
- Use route query parsing helpers for complex navigation state instead of repeating ad hoc parsing in multiple components.
- Keep payment/navigation side effects explicit. `views/user/PaymentView.vue` is the example for branching payment launch flows and preserving recovery snapshots.

## Common Mistakes

- Do not duplicate a generic table, dialog, toast, or empty-state implementation when a common component already covers the workflow.
- Do not leave event listeners, `ResizeObserver`, timers, or abort controllers active after unmount.
- Do not put backend contract transformations directly in templates; normalize in computed properties, composables, stores, or API wrappers.
- Do not hard-code English-only display text when the surrounding view uses `vue-i18n`.
