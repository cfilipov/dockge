<script lang="ts">
	import Button from "./ui/Button.svelte";
	import * as m from "$lib/paraglide/messages";
	import { faPlus } from "@fortawesome/free-solid-svg-icons";
</script>

<Button variant="brand" icon={faPlus} text={m.compose()} />
