<script lang="ts">
	import { tick } from 'svelte';

	export interface MenuItem {
		value: string;
		label: string;
		checked: boolean;
		hint?: string;
	}

	let {
		id,
		label,
		triggerLabel,
		items,
		open = false,
		disabled = false,
		onToggle,
		onSelect,
		onClose
	}: {
		id: string;
		label: string;
		triggerLabel: string;
		items: MenuItem[];
		open?: boolean;
		disabled?: boolean;
		onToggle: () => void;
		onSelect: (value: string) => void;
		onClose: () => void;
	} = $props();

	let panel = $state<HTMLElement | null>(null);

	$effect(() => {
		if (!open) {
			return;
		}
		void tick().then(() => {
			panel?.querySelector<HTMLButtonElement>('[role="menuitemradio"]')?.focus();
		});
	});

	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Escape') {
			event.preventDefault();
			onClose();
			return;
		}
		if (event.key !== 'ArrowDown' && event.key !== 'ArrowUp') {
			return;
		}
		event.preventDefault();
		const menu = event.currentTarget as HTMLElement;
		const options = Array.from(menu.querySelectorAll<HTMLButtonElement>('[role="menuitemradio"]'));
		if (options.length === 0) {
			return;
		}
		const current = options.indexOf(document.activeElement as HTMLButtonElement);
		const direction = event.key === 'ArrowDown' ? 1 : -1;
		const next = (current + direction + options.length) % options.length;
		options[next].focus();
	}
</script>

<div class="player-menu">
	<button
		type="button"
		class="control-button menu-trigger"
		aria-label={label}
		aria-haspopup="menu"
		aria-expanded={open}
		aria-controls={open ? id : undefined}
		disabled={disabled}
		onclick={(event) => {
			event.stopPropagation();
			onToggle();
		}}
	>
		{triggerLabel}
		<span class="menu-caret" aria-hidden="true">▾</span>
	</button>
	<div
		bind:this={panel}
		id={id}
		class="menu-panel"
		role="menu"
		aria-label={label}
		hidden={!open}
		tabindex="-1"
		onkeydown={handleKeydown}
		onclick={(event) => event.stopPropagation()}
	>
		{#each items as item (item.value)}
			<button
				type="button"
				role="menuitemradio"
				aria-checked={item.checked}
				class:active={item.checked}
				onclick={() => onSelect(item.value)}
			>
				<span>{item.label}</span>
				{#if item.hint}<small>{item.hint}</small>{/if}
			</button>
		{/each}
	</div>
</div>

<style>
	.player-menu {
		position: relative;
	}
	.menu-trigger {
		display: inline-flex;
		align-items: center;
		min-width: 40px;
		min-height: 40px;
		padding: 0.35rem 0.5rem;
		border: 1px solid transparent;
		border-radius: 6px;
		background: transparent;
		color: inherit;
		font: inherit;
		font-size: 0.8rem;
		cursor: pointer;
	}
	.menu-trigger:hover,
	.menu-trigger:focus-visible {
		background: rgba(246, 234, 215, 0.16);
		border-color: rgba(246, 234, 215, 0.42);
		outline: none;
	}
	.menu-caret {
		margin-left: 0.2rem;
		font-size: 0.7rem;
	}
</style>
