import { ref, computed, watch } from "vue";

// Module-level reactive state (shared by all importers)
const system = ref(window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light");
const userTheme = ref<string>(localStorage.theme || "dark");
const isMobile = ref(window.innerWidth < 768);
const userTimezone = ref<string>(localStorage.timezone || "auto");

const theme = computed(() => {
    if (userTheme.value === "auto") {
        return system.value;
    }
    return userTheme.value;
});

const isDark = computed(() => theme.value === "dark");

// Persist userTheme to localStorage
watch(userTheme, (to) => {
    localStorage.theme = to;
});

// Update body class and meta tag when theme changes
watch(theme, (to, from) => {
    if (from) {
        document.body.classList.remove(from);
    }
    document.body.classList.add(to);
    updateThemeColorMeta();
});

// Track window resize for isMobile
window.addEventListener("resize", () => {
    isMobile.value = window.innerWidth < 768;
});

// Track system theme preference changes
window.matchMedia("(prefers-color-scheme: dark)").addEventListener("change", (e) => {
    system.value = e.matches ? "dark" : "light";
});

function updateThemeColorMeta() {
    const el = document.querySelector("#theme-color");
    if (el) {
        if (theme.value === "dark") {
            el.setAttribute("content", "#161B22");
        } else {
            el.setAttribute("content", "#5cdd8b");
        }
    }
}

// Initialize on first import
if (!localStorage.theme) {
    userTheme.value = "dark";
}
document.body.classList.add(theme.value);
updateThemeColorMeta();

export function useTheme() {
    return {
        userTheme,
        theme,
        isDark,
        isMobile,
        userTimezone,
    };
}
