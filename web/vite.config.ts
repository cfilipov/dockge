import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import Components from "unplugin-vue-components/vite";
import { BootstrapVueNextResolver } from "unplugin-vue-components/resolvers";
import "vue";

// https://vitejs.dev/config/
const backendPort = process.env.VITE_BACKEND_PORT ?? "5001";

export default defineConfig({
    server: {
        port: parseInt(process.env.VITE_PORT ?? "5000"),
        strictPort: true,
        allowedHosts: true,
        proxy: {
            "/ws": {
                target: `http://localhost:${backendPort}`,
                ws: true,
            },
            "/api": {
                target: `http://localhost:${backendPort}`,
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
                        if (id.includes("codemirror") || id.includes("@lezer")) {
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
