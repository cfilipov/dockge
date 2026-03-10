import type { Preview } from "storybook";
import { withThemeByClassName } from "@storybook/addon-themes";
import "../src/app.css";

const preview: Preview = {
	decorators: [
		withThemeByClassName({
			themes: {
				light: "",
				dark: "dark",
			},
			defaultTheme: "dark",
			parentSelector: "body",
		}),
	],
};

export default preview;
