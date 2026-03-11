<script module lang="ts">
	import { defineMeta } from "@storybook/addon-svelte-csf";
	import { SelectRoot, SelectTrigger, SelectContent, SelectItem, SelectPill } from ".";

	const { Story } = defineMeta({
		title: "UI/Select",
	});
</script>

<script lang="ts">
	const languages: Record<string, string> = { en: "English", de: "Deutsch", fr: "Français", ja: "日本語", zh: "中文" };
	let language = $state("en");
</script>

<Story name="Default">
	<div class="w-64">
		<SelectRoot type="single" bind:value={language}>
			<SelectPill>
				<SelectTrigger>{languages[language] ?? "Select language"}</SelectTrigger>
				<SelectContent>
					<SelectItem value="en">English</SelectItem>
					<SelectItem value="de">Deutsch</SelectItem>
					<SelectItem value="fr">Français</SelectItem>
					<SelectItem value="ja">日本語</SelectItem>
					<SelectItem value="zh">中文</SelectItem>
				</SelectContent>
			</SelectPill>
		</SelectRoot>
	</div>
</Story>

<Story name="With Placeholder">
	<div class="w-64">
		<SelectRoot type="single">
			<SelectPill>
				<SelectTrigger placeholder="Choose a language..." />
				<SelectContent>
					<SelectItem value="en">English</SelectItem>
					<SelectItem value="de">Deutsch</SelectItem>
					<SelectItem value="fr">Français</SelectItem>
				</SelectContent>
			</SelectPill>
		</SelectRoot>
	</div>
</Story>

<Story name="Disabled">
	<div class="w-64">
		<SelectRoot type="single" value="en" disabled>
			<SelectPill>
				<SelectTrigger disabled>English</SelectTrigger>
				<SelectContent>
					<SelectItem value="en">English</SelectItem>
				</SelectContent>
			</SelectPill>
		</SelectRoot>
	</div>
</Story>
