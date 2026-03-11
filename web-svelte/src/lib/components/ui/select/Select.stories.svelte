<script module lang="ts">
	import { defineMeta } from "@storybook/addon-svelte-csf";
	import { SelectRoot, SelectTrigger, SelectContent, SelectItem } from ".";

	const { Story } = defineMeta({
		title: "UI/Select",
	});
</script>

<script lang="ts">
	let language = $state("en");
</script>

<Story name="Default">
	<div class="w-64">
		<SelectRoot type="single" bind:value={language}>
			<SelectTrigger>{language === "en" ? "English" : language === "de" ? "Deutsch" : language === "fr" ? "Français" : "Select language"}</SelectTrigger>
			<SelectContent>
				<SelectItem value="en">English</SelectItem>
				<SelectItem value="de">Deutsch</SelectItem>
				<SelectItem value="fr">Français</SelectItem>
				<SelectItem value="ja">日本語</SelectItem>
				<SelectItem value="zh">中文</SelectItem>
			</SelectContent>
		</SelectRoot>
	</div>
</Story>

<Story name="With Placeholder">
	<div class="w-64">
		<SelectRoot type="single">
			<SelectTrigger placeholder="Choose a language..." />
			<SelectContent>
				<SelectItem value="en">English</SelectItem>
				<SelectItem value="de">Deutsch</SelectItem>
				<SelectItem value="fr">Français</SelectItem>
			</SelectContent>
		</SelectRoot>
	</div>
</Story>

<Story name="Disabled">
	<div class="w-64">
		<SelectRoot type="single" value="en" disabled>
			<SelectTrigger disabled>English</SelectTrigger>
			<SelectContent>
				<SelectItem value="en">English</SelectItem>
			</SelectContent>
		</SelectRoot>
	</div>
</Story>
