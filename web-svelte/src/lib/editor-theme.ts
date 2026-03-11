/**
 * CodeMirror theme matching the PrismJS "Tomorrow Night Eighties" color scheme
 * that was used in the original Prism-based editor.
 *
 * Color reference: https://github.com/chriskempson/tomorrow-theme
 */
import { tags as t } from "@lezer/highlight";
import { EditorView } from "@codemirror/view";
import { HighlightStyle, syntaxHighlighting } from "@codemirror/language";

const theme = EditorView.theme({
    "&": {
        backgroundColor: "transparent",
        color: "#cccccc",
    },
    ".cm-content": {
        caretColor: "#cccccc",
    },
    ".cm-cursor, .cm-dropCursor": {
        borderLeftColor: "#cccccc",
    },
    "&.cm-focused .cm-selectionBackground, .cm-selectionBackground, .cm-content ::selection": {
        backgroundColor: "#74c2ff3d",
    },
    ".cm-activeLine": {
        backgroundColor: "transparent",
    },
    ".cm-gutters": {
        backgroundColor: "transparent",
        color: "#999999",
    },
    ".cm-activeLineGutter": {
        backgroundColor: "transparent",
    },
}, { dark: true });

const highlightStyle = HighlightStyle.define([
    {
        tag: t.comment,
        color: "#999999",
    },
    {
        tag: t.punctuation,
        color: "#cccccc",
    },
    {
        tag: [t.propertyName, t.definition(t.propertyName), t.keyword],
        color: "#cc99cd",
    },
    {
        tag: [t.string, t.special(t.brace)],
        color: "#7ec699",
    },
    {
        tag: [t.number, t.bool, t.null],
        color: "#f08d49",
    },
    {
        tag: t.operator,
        color: "#67cdcc",
    },
    {
        tag: [t.className, t.typeName, t.definition(t.typeName)],
        color: "#f8c555",
    },
    {
        tag: t.function(t.variableName),
        color: "#6196cc",
    },
    {
        tag: [t.tagName, t.attributeName],
        color: "#e2777a",
    },
    {
        tag: t.variableName,
        color: "#cc99cd",
    },
]);

export const tomorrowNightEighties = [theme, syntaxHighlighting(highlightStyle)];

// --- Light theme (Tomorrow / VS Code inspired) ---

const lightTheme = EditorView.theme({
    "&": {
        backgroundColor: "transparent",
        color: "#1e1e1e",
    },
    ".cm-content": {
        caretColor: "#1e1e1e",
    },
    ".cm-cursor, .cm-dropCursor": {
        borderLeftColor: "#1e1e1e",
    },
    "&.cm-focused .cm-selectionBackground, .cm-selectionBackground, .cm-content ::selection": {
        backgroundColor: "#add6ff",
    },
    ".cm-activeLine": {
        backgroundColor: "transparent",
    },
    ".cm-gutters": {
        backgroundColor: "transparent",
        color: "#999999",
    },
    ".cm-activeLineGutter": {
        backgroundColor: "transparent",
    },
}, { dark: false });

const lightHighlightStyle = HighlightStyle.define([
    {
        tag: t.comment,
        color: "#6a737d",
    },
    {
        tag: t.punctuation,
        color: "#444444",
    },
    {
        tag: [t.propertyName, t.definition(t.propertyName), t.keyword],
        color: "#7c3aed",
    },
    {
        tag: [t.string, t.special(t.brace)],
        color: "#0a7e32",
    },
    {
        tag: [t.number, t.bool, t.null],
        color: "#c45100",
    },
    {
        tag: t.operator,
        color: "#0598bc",
    },
    {
        tag: [t.className, t.typeName, t.definition(t.typeName)],
        color: "#b35e00",
    },
    {
        tag: t.function(t.variableName),
        color: "#0451a5",
    },
    {
        tag: [t.tagName, t.attributeName],
        color: "#c4291c",
    },
    {
        tag: t.variableName,
        color: "#7c3aed",
    },
]);

export const tomorrowLight = [lightTheme, syntaxHighlighting(lightHighlightStyle)];
