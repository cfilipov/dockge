// Dayjs init inside this, so it has to be the first import
import "./common/util-common";

import { createApp } from "vue";
import App from "./App.vue";
import { router } from "./router";
import { FontAwesomeIcon } from "./icon.js";
import { i18n } from "./i18n";

// Dependencies
import "bootstrap";
import Toast, { POSITION } from "vue-toastification";
import "@xterm/xterm/lib/xterm.js";

// CSS
import "@fontsource/jetbrains-mono";
import "vue-toastification/dist/index.css";
import "@xterm/xterm/css/xterm.css";
import "./styles/main.scss";

// Composables
import { useSocket, initWebSocket } from "./composables/useSocket";
import { useTheme } from "./composables/useTheme";
import { useLang, initLang } from "./composables/useLang";
import { useAppToast } from "./composables/useAppToast";

// Set Title
document.title = document.title + " - " + location.host;

const app = createApp(App);

app.use(Toast, {
    position: POSITION.BOTTOM_RIGHT,
    showCloseButtonOnHover: true,
    closeOnClick: false,
});
app.use(router);
app.use(i18n);
app.component("FontAwesomeIcon", FontAwesomeIcon);

// Initialize composables (module-level singletons)
useSocket();
useTheme();
useLang();
useAppToast();
initLang(i18n);
initWebSocket();

app.mount("#app");
