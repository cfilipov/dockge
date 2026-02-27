import { ref, computed, watch } from "vue";
import { yaml } from "@codemirror/lang-yaml";
import { python } from "@codemirror/lang-python";
import { tomorrowNightEighties, tomorrowLight } from "../editor-theme";
import { lineNumbers, EditorView } from "@codemirror/view";
import { indentUnit, indentService } from "@codemirror/language";
import { useTheme } from "./useTheme";

const yamlIndent = indentService.of((cx: any, pos: number) => {
    const line = cx.lineAt(pos);
    if (line.number === 1) {
        return 0;
    }
    const prev = cx.lineAt(line.from - 1);
    const prevText = prev.text;
    const prevIndent = prevText.match(/^\s*/)[0].length;
    const trimmed = prevText.trimEnd();
    if (trimmed.endsWith(":") || trimmed.endsWith("|-") || trimmed.endsWith("|") || trimmed.endsWith(">") || trimmed.endsWith(">-")) {
        return prevIndent + 2;
    }
    return prevIndent;
});

export function useCodeMirrorEditor() {
    const { isDark } = useTheme();

    const editorFocus = ref(false);
    const wordWrap = ref(localStorage.getItem("editorWordWrap") !== "false");

    const focusEffectHandler = (_state: any, focusing: boolean) => {
        editorFocus.value = focusing;
        return null;
    };

    const yamlExtensions = computed(() => [
        isDark.value ? tomorrowNightEighties : tomorrowLight,
        yaml(),
        indentUnit.of("  "),
        yamlIndent,
        lineNumbers(),
        EditorView.focusChangeEffect.of(focusEffectHandler),
    ]);

    const envExtensions = computed(() => [
        isDark.value ? tomorrowNightEighties : tomorrowLight,
        python(),
        lineNumbers(),
        EditorView.focusChangeEffect.of(focusEffectHandler),
    ]);

    watch(wordWrap, (v) => {
        localStorage.setItem("editorWordWrap", v ? "true" : "false");
    });

    return {
        isDark,
        editorFocus,
        wordWrap,
        yamlExtensions,
        envExtensions,
    };
}
