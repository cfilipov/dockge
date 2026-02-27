# Step 7 Issues — Non-Test-Code Changes Needed

These issues were discovered during Step 7 test writing but require changes
outside of test code, so they were deferred per user instructions.

## 1. Pause/Resume Stack — No UI Buttons

**Priority 12 test: SKIPPED**

The backend fully supports pause/resume (`handlePauseStack`, `handleResumeStack`
in `internal/handlers/stack.go`), and `MockState` supports `"paused"` status.
However, the frontend's `StackList.vue` has no visible buttons to enter selection
mode or trigger `pauseDialog()` / `resumeSelected()`. The functions exist in
`<script setup>` but are never exposed in the `<template>`.

**To make this testable:**
- Add a pause/resume button to the individual stack page (Compose.vue), OR
- Add selection mode toggle and toolbar buttons to `StackList.vue`

## 2. Force Delete Stack — Cannot Trigger in Mock Mode (from Step 6)

**Priority 8 test: SKIPPED**

The "Force Delete" menu item in Compose.vue only appears when `errorDelete === true`,
which is set when a regular delete fails. In mock mode, `handleDeleteStack` always
returns `ok: true`, so `errorDelete` is never set and Force Delete is never shown.

**To make this testable:**
- Add a way for mock mode to simulate a delete failure, OR
- Expose Force Delete as a separate menu item that's always available
