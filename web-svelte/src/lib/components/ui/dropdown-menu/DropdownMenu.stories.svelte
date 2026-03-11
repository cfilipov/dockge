<script module lang="ts">
	import { defineMeta } from "@storybook/addon-svelte-csf";
	import {
		DropdownMenuRoot,
		DropdownMenuTrigger,
		DropdownMenuContent,
		DropdownMenuItem,
		DropdownMenuSeparator,
	} from ".";
	import Icon from "../../Icon.svelte";
	import {
		faPlay,
		faStop,
		faPen,
		faTrash,
		faArrowsRotate,
		faEllipsisVertical,
	} from "@fortawesome/free-solid-svg-icons";

	const { Story } = defineMeta({
		title: "UI/DropdownMenu",
		argTypes: {
			open: { control: "boolean" },
		},
		args: {
			open: true,
		},
	});
</script>

<Story name="Default">
	{#snippet template(args)}
		<DropdownMenuRoot open={args.open}>
			<DropdownMenuTrigger>
				{#snippet child({ props })}
					<button
						{...props}
						class="rounded-lg border border-gray-300 bg-white px-4 py-2 text-sm dark:border-(--color-border-dark) dark:bg-(--color-body-dark) dark:text-(--color-font-dark)"
					>
						Open Menu
					</button>
				{/snippet}
			</DropdownMenuTrigger>
			<DropdownMenuContent>
				<DropdownMenuItem onclick={() => console.log("edit")}>
					<Icon icon={faPen} />
					Edit
				</DropdownMenuItem>
				<DropdownMenuItem onclick={() => console.log("restart")}>
					<Icon icon={faArrowsRotate} />
					Restart
				</DropdownMenuItem>
				<DropdownMenuItem onclick={() => console.log("stop")}>
					<Icon icon={faStop} />
					Stop
				</DropdownMenuItem>
				<DropdownMenuSeparator />
				<DropdownMenuItem variant="danger" onclick={() => console.log("delete")}>
					<Icon icon={faTrash} />
					Delete
				</DropdownMenuItem>
			</DropdownMenuContent>
		</DropdownMenuRoot>
	{/snippet}
</Story>

<Story name="With Icon Trigger">
	{#snippet template(args)}
		<DropdownMenuRoot open={args.open}>
			<DropdownMenuTrigger>
				{#snippet child({ props })}
					<button
						{...props}
						aria-label="More actions"
						class="inline-flex h-8 w-8 items-center justify-center rounded-full text-gray-500 hover:bg-gray-100 dark:text-(--color-font-dark) dark:hover:bg-white/10"
					>
						<Icon icon={faEllipsisVertical} />
					</button>
				{/snippet}
			</DropdownMenuTrigger>
			<DropdownMenuContent>
				<DropdownMenuItem>
					<Icon icon={faPlay} />
					Start
				</DropdownMenuItem>
				<DropdownMenuItem>
					<Icon icon={faStop} />
					Stop
				</DropdownMenuItem>
				<DropdownMenuItem>
					<Icon icon={faArrowsRotate} />
					Restart
				</DropdownMenuItem>
			</DropdownMenuContent>
		</DropdownMenuRoot>
	{/snippet}
</Story>

<Story name="Danger Items">
	{#snippet template(args)}
		<DropdownMenuRoot open={args.open}>
			<DropdownMenuTrigger>
				{#snippet child({ props })}
					<button
						{...props}
						class="rounded-lg border border-gray-300 bg-white px-4 py-2 text-sm dark:border-(--color-border-dark) dark:bg-(--color-body-dark) dark:text-(--color-font-dark)"
					>
						Destructive Actions
					</button>
				{/snippet}
			</DropdownMenuTrigger>
			<DropdownMenuContent>
				<DropdownMenuItem onclick={() => console.log("stop")}>
					<Icon icon={faStop} />
					Stop
				</DropdownMenuItem>
				<DropdownMenuSeparator />
				<DropdownMenuItem variant="danger" onclick={() => console.log("delete")}>
					<Icon icon={faTrash} />
					Delete
				</DropdownMenuItem>
				<DropdownMenuItem variant="danger" onclick={() => console.log("force-delete")}>
					<Icon icon={faTrash} />
					Force Delete
				</DropdownMenuItem>
			</DropdownMenuContent>
		</DropdownMenuRoot>
	{/snippet}
</Story>

<Story name="Text-Only Items">
	{#snippet template(args)}
		<DropdownMenuRoot open={args.open}>
			<DropdownMenuTrigger>
				{#snippet child({ props })}
					<button
						{...props}
						class="rounded-lg border border-gray-300 bg-white px-4 py-2 text-sm dark:border-(--color-border-dark) dark:bg-(--color-body-dark) dark:text-(--color-font-dark)"
					>
						Options
					</button>
				{/snippet}
			</DropdownMenuTrigger>
			<DropdownMenuContent>
				<DropdownMenuItem>Profile</DropdownMenuItem>
				<DropdownMenuItem>Settings</DropdownMenuItem>
				<DropdownMenuItem>Notifications</DropdownMenuItem>
				<DropdownMenuSeparator />
				<DropdownMenuItem variant="danger">Log Out</DropdownMenuItem>
			</DropdownMenuContent>
		</DropdownMenuRoot>
	{/snippet}
</Story>
