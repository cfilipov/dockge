import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import Components from "unplugin-vue-components/vite";
import { BootstrapVueNextResolver } from "unplugin-vue-components/resolvers";
import "vue";

// https://vitejs.dev/config/
export default defineConfig({
    server: {
        port: 5000,
        strictPort: true,
        allowedHosts: true,
        proxy: {
            "/ws": {
                target: "http://localhost:5001",
                ws: true,
            },
        },
    },
    define: {
        "FRONTEND_VERSION": JSON.stringify(process.env.npm_package_version),
    },
    root: ".",
    build: {
        outDir: "../dist",
        emptyOutDir: true,
        chunkSizeWarningLimit: 800,
        rollupOptions: {
            output: {
                manualChunks(id) {
                    if (id.includes("node_modules")) {
                        if (id.includes("codemirror") || id.includes("thememirror") || id.includes("@lezer")) {
                            return "codemirror";
                        }
                        if (id.includes("@xterm") || id.includes("xterm-addon")) {
                            return "xterm";
                        }
                    }
                },
            },
        },
    },
    plugins: [
        vue(),
        Components({
            resolvers: [ BootstrapVueNextResolver() ],
        }),
    ],
});
