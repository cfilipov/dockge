import { browser } from "$app/environment";

type ThemeSetting = "dark" | "light" | "auto";

const STORAGE_KEY = "dockge-theme";
const MOBILE_BREAKPOINT = 768;

function getStoredTheme(): ThemeSetting {
	if (!browser) return "auto";
	return (localStorage.getItem(STORAGE_KEY) as ThemeSetting) ?? "dark";
}

function getSystemPrefersDark(): boolean {
	if (!browser) return true;
	return window.matchMedia("(prefers-color-scheme: dark)").matches;
}

let userTheme = $state<ThemeSetting>(getStoredTheme());
let systemDark = $state(getSystemPrefersDark());
let windowWidth = $state(browser ? window.innerWidth : 1024);

const actualTheme = $derived<"dark" | "light">(
	userTheme === "auto" ? (systemDark ? "dark" : "light") : userTheme,
);

const isDark = $derived(actualTheme === "dark");
const isMobile = $derived(windowWidth < MOBILE_BREAKPOINT);

if (browser) {
	const mq = window.matchMedia("(prefers-color-scheme: dark)");
	mq.addEventListener("change", (e) => {
		systemDark = e.matches;
	});

	window.addEventListener("resize", () => {
		windowWidth = window.innerWidth;
	});
}

function setTheme(t: ThemeSetting) {
	userTheme = t;
	if (browser) {
		localStorage.setItem(STORAGE_KEY, t);
	}
}

export const theme = {
	get actual() {
		return actualTheme;
	},
	get isDark() {
		return isDark;
	},
	get isMobile() {
		return isMobile;
	},
	get setting() {
		return userTheme;
	},
	set: setTheme,
};
