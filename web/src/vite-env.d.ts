/* eslint-disable */
/// <reference types="vite/client" />

declare const FRONTEND_VERSION: string;

// Vite define replacements for dev container / codespace detection
declare const DEVCONTAINER: string | undefined;
declare const CODESPACE_NAME: string;
declare const GITHUB_CODESPACES_PORT_FORWARDING_DOMAIN: string;

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
