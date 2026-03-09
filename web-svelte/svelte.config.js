import adapter from "@sveltejs/adapter-static";

/** @type {import('@sveltejs/kit').Config} */
const config = {
	kit: {
		adapter: adapter({
			pages: "../dist-svelte",
			assets: "../dist-svelte",
			fallback: "index.html",
		}),
	},
};

export default config;
