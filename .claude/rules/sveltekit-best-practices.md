---
description: SvelteKit Best Practices - enforces current best practices when working with SvelteKit / Svelte 5 code
alwaysApply: true
---

Use Svelte 5 runes ($state, $derived, $effect) - never use Svelte 4 stores (writable, readable) or reactive declarations ($:).
Use $props() for component props - never use export let.
Use load functions in +page.server.ts for data fetching - never fetch data in onMount.
Use form actions for mutations - never create API routes just for form submissions.
Use $effect for side effects - never use $: reactive statements.
Include `<script lang="ts">` for clarity even if TypeScript is configured - it improves IDE support and makes intent explicit.

## Styling (Tailwind-first)

All styling in `web-svelte/` MUST use Tailwind utility classes inline on elements. Do NOT add `<style>` blocks to Svelte components. Specifically:
- Use Tailwind's `dark:` variant for dark mode - never use `:global(.dark)` selectors.
- Use standard Tailwind breakpoints (`sm:`, `md:`, `lg:`, `xl:`, `2xl:`) - never add custom `@media` queries.
- For styles reused across multiple components, define an `@utility` rule in `app.css` - never duplicate CSS.
- If a style truly cannot be expressed as Tailwind utilities (e.g., complex animations, pseudo-element content), stop and explain why before adding a `<style>` block.