<script lang="ts">
	import { enhance } from '$app/forms';
	import { onMount } from 'svelte';
	import type { SubmitFunction } from '@sveltejs/kit';

	let {
		id,
		triggerLabel,
		title,
		description,
		confirmLabel = 'Confirmar',
		action,
		fields = {},
		triggerClass = 'button button-secondary',
		danger = false
	}: {
		id: string;
		triggerLabel: string;
		title: string;
		description: string;
		confirmLabel?: string;
		action: string;
		fields?: Record<string, string | number>;
		triggerClass?: string;
		danger?: boolean;
	} = $props();

	let dialogEl: HTMLDialogElement | null = $state(null);
	let openButton: HTMLButtonElement | null = $state(null);
	let cancelButton: HTMLButtonElement | null = $state(null);
	let jsReady = $state(false);
	let submitting = $state(false);
	let dialogOpen = $state(false);
	const titleID = $derived(`${id}-title`);
	const descriptionID = $derived(`${id}-description`);

	onMount(() => {
		jsReady = true;
	});

	function openDialog(): void {
		if (!dialogEl || submitting) {
			return;
		}
		dialogOpen = true;
		try {
			if (typeof dialogEl.showModal === 'function') {
				dialogEl.showModal();
			}
		} catch {
			// El atributo open mantiene el diálogo visible si showModal falla.
		}
		queueMicrotask(() => cancelButton?.focus());
	}

	function closeDialog(): void {
		if (!dialogEl) {
			return;
		}
		dialogOpen = false;
		if (typeof dialogEl.close === 'function' && dialogEl.open) {
			dialogEl.close();
		}
		openButton?.focus();
	}

	function handleClose(): void {
		dialogOpen = false;
		openButton?.focus();
	}

	function handleCancel(event: Event): void {
		event.preventDefault();
		closeDialog();
	}

	function handleClick(event: MouseEvent): void {
		if (event.target === dialogEl) {
			closeDialog();
		}
	}

	function focusables(): HTMLElement[] {
		if (!dialogEl) {
			return [];
		}
		return Array.from(
			dialogEl.querySelectorAll<HTMLElement>(
				'button:not([disabled]), [href], input:not([disabled]):not([type="hidden"]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'
			)
		);
	}

	function handleKeydown(event: KeyboardEvent): void {
		if (event.key === 'Escape') {
			event.preventDefault();
			closeDialog();
			return;
		}
		if (event.key !== 'Tab') {
			return;
		}
		const items = focusables();
		if (items.length === 0) {
			return;
		}
		const first = items[0];
		const last = items[items.length - 1];
		if (event.shiftKey && document.activeElement === first) {
			event.preventDefault();
			last.focus();
		} else if (!event.shiftKey && document.activeElement === last) {
			event.preventDefault();
			first.focus();
		}
	}

	function handleSubmit(event: SubmitEvent): void {
		event.preventDefault();
		if (submitting) {
			return;
		}
		submitting = true;
		closeDialog();
	}

	const submit: SubmitFunction = () => {
		submitting = true;
		closeDialog();
		return async ({ update }) => {
			try {
				await update();
			} finally {
				submitting = false;
				openButton?.focus();
			}
		};
	};
</script>

<span class="confirm-trigger-wrap" hidden={!jsReady}>
	<button
		class="confirm-dialog-trigger {triggerClass}"
		type="button"
		disabled={submitting}
		aria-haspopup="dialog"
		aria-controls={id}
		bind:this={openButton}
		onclick={openDialog}>{triggerLabel}</button
	>
</span>

<dialog
	bind:this={dialogEl}
	{id}
	class="confirm-dialog"
	open={dialogOpen}
	aria-labelledby={titleID}
	aria-describedby={descriptionID}
	aria-modal="true"
	onclick={handleClick}
	onkeydown={handleKeydown}
	oncancel={handleCancel}
	onclose={handleClose}
>
	<div class="confirm-dialog-body">
		<h2 id={titleID}>{title}</h2>
		<p id={descriptionID}>{description}</p>
		<div class="confirm-dialog-actions">
			<button
				class="button button-secondary"
				type="button"
				bind:this={cancelButton}
				onclick={closeDialog}>Cancelar</button
			>
			<form method="POST" action={action} use:enhance={submit} onsubmit={handleSubmit}>
				{#each Object.entries(fields) as [name, value] (name)}
					<input type="hidden" {name} value={String(value)} />
				{/each}
				<button
					class={danger ? 'button button-danger' : 'button'}
					type="submit"
					disabled={submitting}>{confirmLabel}</button
				>
			</form>
		</div>
	</div>
</dialog>

<noscript hidden={jsReady}>
	<details class="confirm-fallback">
		<summary class="confirm-fallback-toggle">{triggerLabel}</summary>
		<div class="confirm-fallback-body">
			<p>{description}</p>
			<form method="POST" action={action}>
				{#each Object.entries(fields) as [name, value] (name)}
					<input type="hidden" {name} value={String(value)} />
				{/each}
				<button class={danger ? 'button button-danger' : 'button'} type="submit">
					{confirmLabel}
				</button>
			</form>
		</div>
	</details>
</noscript>

<style>
	.confirm-trigger-wrap[hidden] {
		display: none;
	}

	.confirm-dialog {
		width: min(32rem, calc(100vw - 2rem));
		max-width: calc(100vw - 2rem);
		max-height: calc(100vh - 2rem);
		max-height: calc(100dvh - 2rem);
		margin: auto;
		padding: 0;
		overflow-y: auto;
		border: 1px solid var(--borde-suave);
		border-radius: 6px;
		background: var(--superficie);
		color: var(--texto-tenue);
		box-shadow: 0 1.5rem 3rem -1.5rem var(--sombra);
	}
	.confirm-dialog::backdrop {
		background: rgb(20 10 11 / 65%);
	}
	.confirm-dialog-body {
		padding: 1.2rem;
	}
	.confirm-dialog h2 {
		margin: 0;
		color: var(--granate);
		font-family: 'Fraunces', Georgia, serif;
		font-size: 1.2rem;
		line-height: 1.2;
	}
	.confirm-dialog p {
		margin: 0.8rem 0 0;
		color: var(--texto-tenue);
		font-size: 0.9rem;
		line-height: 1.55;
	}
	.confirm-dialog-actions {
		display: flex;
		flex-wrap: wrap;
		justify-content: flex-end;
		gap: 0.65rem;
		margin-top: 1.1rem;
	}
	.confirm-dialog-actions form {
		margin: 0;
	}
	.confirm-dialog-actions .button {
		margin-top: 0;
	}
	.confirm-fallback {
		margin-top: 1.5rem;
		border: 1px solid var(--borde-suave);
		border-radius: 4px;
		background: var(--superficie-suave);
	}
	.confirm-fallback-toggle {
		display: inline-flex;
		margin: 0.7rem;
		padding: 0.5rem 0.8rem;
		border-radius: 3px;
		background: var(--boton-fondo);
		color: var(--boton-texto);
		font-size: 0.72rem;
		font-weight: 600;
		letter-spacing: 0.1em;
		list-style: none;
		text-transform: uppercase;
	}
	.confirm-fallback-toggle::-webkit-details-marker {
		display: none;
	}
	.confirm-fallback-body {
		padding: 0 0.8rem 0.8rem;
	}
	.confirm-fallback-body p {
		margin: 0 0 0.8rem;
		color: var(--texto-tenue);
		font-size: 0.88rem;
		line-height: 1.5;
	}
	.confirm-fallback-body form {
		margin: 0;
	}
	@media (max-width: 420px) {
		.confirm-dialog-actions {
			flex-direction: column;
		}
		.confirm-dialog-actions .button {
			width: 100%;
		}
	}
</style>
