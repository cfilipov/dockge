/* eslint-disable */
/// <reference types="vite/client" />

declare const FRONTEND_VERSION: string;

declare module "composerize" {
    export default function composerize(
        command: string,
        existingCompose?: string,
        composeVersion?: string,
        indent?: number,
    ): string;
}

declare module "*.vue" {
    import type { DefineComponent } from "vue";
    const component: DefineComponent<{}, {}, any>;
    export default component;
}
