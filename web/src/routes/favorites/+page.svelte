<script lang="ts">
	import UserListPage from '$lib/components/UserListPage.svelte';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();
</script>

<UserListPage list="favorite" {data} />
