import { yaml } from "@codemirror/lang-yaml";
import { python } from "@codemirror/lang-python";
import { json } from "@codemirror/lang-json";
import { indentUnit, indentService } from "@codemirror/language";

const yamlIndent = indentService.of((cx, pos) => {
    const line = cx.lineAt(pos);
    if (line.from === 0) {
        return 0;
    }
    const prev = cx.lineAt(line.from - 1);
    const prevText = prev.text;
    const prevIndent = prevText.match(/^\s*/)![0].length;
    const trimmed = prevText.trimEnd();
    if (
        trimmed.endsWith(":") ||
        trimmed.endsWith("|-") ||
        trimmed.endsWith("|") ||
        trimmed.endsWith(">") ||
        trimmed.endsWith(">-")
    ) {
        return prevIndent + 2;
    }
    return prevIndent;
});

export const yamlLang = [yaml(), indentUnit.of("  "), yamlIndent];
export const envLang = [python()];
export const jsonLang = [json()];
