<script lang="ts">
	import type { Snippet } from 'svelte';
	import AdminNav from '$lib/components/AdminNav.svelte';
	import type { LayoutData } from './$types';

	let { data, children }: { data: LayoutData; children: Snippet } = $props();
</script>

<div class="admin-shell">
	<AdminNav pathname={data.pathname} />
	<div class="admin-main">
		{@render children()}
	</div>
</div>
