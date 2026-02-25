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
        backgroundColor: "#2d2d2d",
        color: "#cccccc",
    },
    ".cm-content": {
        caretColor: "#cccccc",
    },
    ".cm-cursor, .cm-dropCursor": {
        borderLeftColor: "#cccccc",
    },
    "&.cm-focused .cm-selectionBackground, .cm-selectionBackground, .cm-content ::selection": {
        backgroundColor: "#515151",
    },
    ".cm-activeLine": {
        backgroundColor: "#393939",
    },
    ".cm-gutters": {
        backgroundColor: "#2d2d2d",
        color: "#999999",
    },
    ".cm-activeLineGutter": {
        backgroundColor: "#393939",
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
