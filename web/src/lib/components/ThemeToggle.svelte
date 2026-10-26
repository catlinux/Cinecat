<script lang="ts">
	import { onMount } from 'svelte';
	import {
		applyTheme,
		getStoredTheme,
		setThemePreference,
		type ThemePreference
	} from '$lib/theme';

	const options: Array<{ value: ThemePreference; label: string; title: string }> = [
		{ value: 'system', label: 'Auto', title: 'Tema del sistema' },
		{ value: 'light', label: 'Claro', title: 'Tema claro' },
		{ value: 'dark', label: 'Oscuro', title: 'Tema oscuro' }
	];

	let { defaultPreference = 'system' }: { defaultPreference?: ThemePreference } = $props();
	let preference = $state<ThemePreference | null>(null);
	const selected = $derived(preference ?? defaultPreference);

	onMount(() => {
		preference = getStoredTheme(defaultPreference);
		applyTheme(preference);

		const media = window.matchMedia('(prefers-color-scheme: dark)');
		const onChange = () => {
			if (preference === 'system') {
				applyTheme('system');
			}
		};
		media.addEventListener('change', onChange);
		return () => media.removeEventListener('change', onChange);
	});

	function choose(value: ThemePreference) {
		preference = value;
		setThemePreference(value);
	}
</script>

<div class="theme-toggle" role="group" aria-label="Tema de la interfaz">
	{#each options as option (option.value)}
		<button
			type="button"
			aria-pressed={selected === option.value}
			title={option.title}
			onclick={() => choose(option.value)}
		>
			{option.label}
		</button>
	{/each}
</div>
