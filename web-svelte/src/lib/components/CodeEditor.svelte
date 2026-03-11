<script lang="ts">
	import { EditorState, Compartment, type Extension } from "@codemirror/state";
	import { EditorView, lineNumbers } from "@codemirror/view";
	import { tomorrowNightEighties, tomorrowLight } from "$lib/editor-theme";
	import { faArrowTurnDown, faArrowRight, faExpand } from "@fortawesome/free-solid-svg-icons";
	import Card from "./ui/Card.svelte";
	import IconButton from "./ui/IconButton.svelte";
	import { untrack } from "svelte";

	interface Props {
		value?: string;
		extensions?: Extension[];
		readonly?: boolean;
		onfullscreen?: () => void;
		class?: string;
	}

	let {
		value = $bindable(""),
		extensions = [],
		readonly: readonlyProp = false,
		onfullscreen,
		class: className = "",
	}: Props = $props();

	const WRAP_KEY = "editorWordWrap";
	let wrap = $state(
		typeof localStorage !== "undefined" ? localStorage.getItem(WRAP_KEY) !== "false" : true,
	);

	$effect(() => {
		localStorage.setItem(WRAP_KEY, wrap ? "true" : "false");
	});

	let container = $state<HTMLDivElement>();
	let view: EditorView | undefined;
	let internalUpdate = false;

	// Observe .dark class on body — works in both the real app and Storybook
	let isDark = $state(typeof document !== "undefined" && document.body.classList.contains("dark"));

	$effect(() => {
		const observer = new MutationObserver(() => {
			isDark = document.body.classList.contains("dark");
		});
		observer.observe(document.body, { attributes: true, attributeFilter: ["class"] });
		// Sync on mount in case class was set before observer started
		isDark = document.body.classList.contains("dark");
		return () => observer.disconnect();
	});

	const themeComp = new Compartment();
	const extComp = new Compartment();
	const wrapComp = new Compartment();
	const readonlyComp = new Compartment();

	function getThemeExt() {
		return isDark ? tomorrowNightEighties : tomorrowLight;
	}

	function getWrapExt() {
		return wrap ? EditorView.lineWrapping : [];
	}

	function getReadonlyExt(): Extension {
		return readonlyProp
			? [EditorView.editable.of(false), EditorState.readOnly.of(true)]
			: [EditorView.editable.of(true), EditorState.readOnly.of(false)];
	}

	// Mount effect — depends on `container` ($state set by bind:this)
	$effect(() => {
		if (!container) return;

		const initialValue = untrack(() => value);
		const initialExtensions = untrack(() => extensions);
		const initialTheme = untrack(() => getThemeExt());
		const initialWrap = untrack(() => getWrapExt());
		const initialReadonly = untrack(() => getReadonlyExt());

		const state = EditorState.create({
			doc: initialValue,
			extensions: [
				lineNumbers(),
				themeComp.of(initialTheme),
				extComp.of(initialExtensions),
				wrapComp.of(initialWrap),
				readonlyComp.of(initialReadonly),
				EditorView.updateListener.of((update) => {
					if (update.docChanged) {
						internalUpdate = true;
						value = update.state.doc.toString();
					}
				}),
			],
		});

		view = new EditorView({ state, parent: container });

		return () => {
			view?.destroy();
			view = undefined;
		};
	});

	// Theme reconfigure
	$effect(() => {
		const ext = getThemeExt();
		view?.dispatch({ effects: themeComp.reconfigure(ext) });
	});

	// Extensions reconfigure
	$effect(() => {
		// Read `extensions` to track it
		const ext = extensions;
		view?.dispatch({ effects: extComp.reconfigure(ext) });
	});

	// Wrap reconfigure
	$effect(() => {
		const ext = getWrapExt();
		view?.dispatch({ effects: wrapComp.reconfigure(ext) });
	});

	// Readonly reconfigure
	$effect(() => {
		const ext = getReadonlyExt();
		view?.dispatch({ effects: readonlyComp.reconfigure(ext) });
	});

	// External value sync
	$effect(() => {
		// Read `value` to track it
		const v = value;
		if (internalUpdate) {
			internalUpdate = false;
			return;
		}
		if (view && view.state.doc.toString() !== v) {
			view.dispatch({
				changes: { from: 0, to: view.state.doc.length, insert: v },
			});
		}
	});
</script>

<Card
	class="relative overflow-hidden p-[10px] dark:shadow-lg
		[&_.cm-focused]:!outline-none [&_.cm-editor]:font-mono [&_.cm-editor]:text-sm
		[&_.cm-gutters]:!bg-white dark:[&_.cm-gutters]:!bg-(--color-body-dark)
		{className}"
>
	<div bind:this={container}></div>
	<div class="absolute right-[10px] top-[10px] z-10 flex gap-1">
		<IconButton
			icon={wrap ? faArrowTurnDown : faArrowRight}
			aria-label={wrap ? "Disable word wrap" : "Enable word wrap"}
			size="sm"
			onclick={() => (wrap = !wrap)}
		/>
		{#if onfullscreen}
			<IconButton
				icon={faExpand}
				aria-label="Fullscreen"
				size="sm"
				onclick={onfullscreen}
			/>
		{/if}
	</div>
</Card>
