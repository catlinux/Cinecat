<script lang="ts">
	import { enhance } from '$app/forms';
	import { page } from '$app/state';
	import type { SubmitFunction } from '@sveltejs/kit';
	import type { UserListKind, UserListName } from '$lib/api';
	import { listToggleLabel, listToggleResult } from '$lib/lists';

	let {
		kind,
		id,
		list,
		active,
		size = 'normal',
		label = ''
	}: {
		kind: UserListKind;
		id: number;
		list: UserListName;
		active: boolean;
		size?: 'normal' | 'small';
		label?: string;
	} = $props();

	// svelte-ignore state_referenced_locally
	let currentActive = $state(active);
	let pending = $state(false);
	let error = $state<string | null>(null);
	// svelte-ignore state_referenced_locally
	let lastActive = active;
	const redirectTo = $derived(`${page.url.pathname}${page.url.search}${page.url.hash}`);
	const ariaLabel = $derived(listToggleLabel(list, currentActive));

	$effect(() => {
		const next = active;
		if (next === lastActive || pending) {
			return;
		}
		lastActive = next;
		currentActive = next;
	});

	const enhanceToggle: SubmitFunction = () => {
		const previous = currentActive;
		currentActive = !currentActive;
		pending = true;
		error = null;
		return async ({ result }) => {
			pending = false;
			if (result.type === 'failure' || result.type === 'error') {
				const next = listToggleResult(previous, result.type);
				currentActive = next.active;
				error = next.error;
			} else if (result.type === 'redirect' && result.location) {
				window.location.assign(result.location);
			}
		};
	};
</script>

<form
	method="POST"
	action="/lists"
	class="list-toggle-form"
	class:list-toggle-form--small={size === 'small'}
	use:enhance={enhanceToggle}
>
	<input type="hidden" name="kind" value={kind} />
	<input type="hidden" name="id" value={id} />
	<input type="hidden" name="list" value={list} />
	<input type="hidden" name="on" value={currentActive ? '0' : '1'} />
	<input type="hidden" name="redirectTo" value={redirectTo} />
	<button
		type="submit"
		class="list-toggle"
		class:list-toggle--active={currentActive}
		class:list-toggle--small={size === 'small'}
		aria-pressed={currentActive}
		aria-label={ariaLabel}
		title={ariaLabel}
		disabled={pending}
	>
		{#if list === 'favorite'}
			<svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">
				<path
					d="M12 20.4 4.6 13a4.8 4.8 0 0 1 6.8-6.8l.6.6.6-.6A4.8 4.8 0 1 1 19.4 13z"
					fill={currentActive ? 'currentColor' : 'none'}
					stroke="currentColor"
					stroke-width="1.8"
					stroke-linejoin="round"
				/>
			</svg>
		{:else}
			<svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">
				<path
					d="M6 4.5A1.5 1.5 0 0 1 7.5 3h9A1.5 1.5 0 0 1 18 4.5V21l-6-3.8L6 21z"
					fill={currentActive ? 'currentColor' : 'none'}
					stroke="currentColor"
					stroke-width="1.8"
					stroke-linejoin="round"
				/>
			</svg>
		{/if}
		{#if label}<span>{label}</span>{/if}
	</button>
	{#if error}<span class="list-toggle-error" role="status">{error}</span>{/if}
</form>

<style>
	.list-toggle-form {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
		margin: 0;
	}
	.list-toggle {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 0.4rem;
		min-height: 40px;
		padding: 0.45rem 0.65rem;
		border: 1px solid var(--borde-suave);
		border-radius: 3px;
		background: var(--superficie);
		color: var(--texto-tenue);
		font: inherit;
		font-size: 0.7rem;
		font-weight: 600;
		cursor: pointer;
	}
	.list-toggle svg {
		width: 1.15rem;
		height: 1.15rem;
		flex: 0 0 auto;
	}
	.list-toggle:hover:not(:disabled),
	.list-toggle:focus-visible {
		border-color: var(--dorado);
		color: var(--granate);
	}
	.list-toggle--active {
		border-color: var(--dorado);
		background: var(--tinte);
		color: var(--naranja);
	}
	.list-toggle:disabled {
		cursor: wait;
		opacity: 0.65;
	}
	.list-toggle--small {
		width: 40px;
		height: 40px;
		min-height: 40px;
		padding: 0.4rem;
		border-radius: 50%;
		background: rgb(28 13 15 / 78%);
		color: var(--crema-clara);
		box-shadow: 0 0.4rem 1rem -0.5rem var(--sombra);
	}
	.list-toggle--small.list-toggle--active {
		background: var(--dorado);
		color: var(--granate-oscuro);
	}
	.list-toggle-error {
		color: var(--naranja);
		font-size: 0.66rem;
	}
</style>
