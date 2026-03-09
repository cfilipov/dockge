import { sveltekit } from "@sveltejs/kit/vite";
import tailwindcss from "@tailwindcss/vite";
import { defineConfig } from "vite";

export default defineConfig({
	plugins: [tailwindcss(), sveltekit()],
	server: {
		port: 6100,
		strictPort: true,
		allowedHosts: true,
		proxy: {
			"/ws": {
				target: "http://localhost:6001",
				ws: true,
			},
			"/api": {
				target: "http://localhost:6001",
			},
		},
	},
});
