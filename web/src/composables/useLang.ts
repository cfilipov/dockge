import { ref, watch } from "vue";
import { currentLocale } from "../i18n";
import { setPageLocale } from "../util-frontend";

const langModules = import.meta.glob("../lang/*.json");

const language = ref(currentLocale());

// Module-level i18n instance reference (set during init)
let i18nInstance: any = null;

export function initLang(i18n: any) {
    i18nInstance = i18n;

    // Load non-English language on init
    if (language.value !== "en") {
        changeLang(language.value);
    }

    // Watch for language changes
    watch(language, async (lang) => {
        await changeLang(lang);
    });
}

async function changeLang(lang: string) {
    if (!i18nInstance) return;
    const module = await langModules["../lang/" + lang + ".json"]();
    const message = (module as any).default;
    i18nInstance.global.setLocaleMessage(lang, message);
    i18nInstance.global.locale = lang;
    localStorage.locale = lang;
    setPageLocale();
}

export function useLang() {
    return {
        language,
        changeLang,
    };
}
