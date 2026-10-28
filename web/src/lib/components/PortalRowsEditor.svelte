<script lang="ts">
	import { untrack } from 'svelte';
	import { movePortalRow, type PortalRow } from '$lib/settings';

	type PortalRowsForm = {
		portalRowsAction?: boolean;
		success?: boolean;
		message?: string;
		portalRows?: PortalRow[];
	} | null;

	let {
		rows,
		form = null
	}: {
		rows: PortalRow[];
		form?: PortalRowsForm;
	} = $props();

	let ordered = $state<PortalRow[]>(untrack(() => rows.map((row) => ({ ...row }))));
	let announcement = $state('');

	$effect(() => {
		const incoming = form?.portalRows ?? rows;
		ordered = incoming.map((row) => ({ ...row }));
	});

	function move(index: number, direction: -1 | 1): void {
		const next = movePortalRow(ordered, ordered[index].id, direction);
		if (!next) {
			return;
		}
		const target = index + direction;
		ordered = next;
		announcement = `«${next[target].title}» ahora está en la posición ${target + 1}.`;
	}

	function moveWithKeyboard(event: MouseEvent, index: number, direction: -1 | 1): void {
		event.preventDefault();
		move(index, direction);
	}
</script>

<form method="POST" action="?/portalRows" class="portal-rows-editor">
	<input type="hidden" name="row_count" value={ordered.length} />
	<input type="hidden" name="portal_rows" value={JSON.stringify(ordered)} />

	{#if form?.message}
		<p class={form.success ? 'success' : 'error'} role={form.success ? 'status' : 'alert'}>
			{form.message}
		</p>
	{/if}

	<p class="portal-rows-announcement" aria-live="polite">{announcement}</p>

	<div class="portal-row-list">
		{#each ordered as row, index (row.id)}
			<fieldset class="portal-row-editor">
				<legend>{row.title || row.id}</legend>
				<label class="portal-row-toggle">
					<input type="hidden" name={`row_id_${index}`} value={row.id} />
					<input type="checkbox" name={`row_visible_${index}`} value="1" bind:checked={row.visible} />
					<span>Mostrar</span>
				</label>
				<label>
					<span>Título</span>
					<input
						type="text"
						name={`row_title_${index}`}
						bind:value={row.title}
						maxlength="60"
					/>
				</label>
				<label>
					<span>Antetítulo</span>
					<input
						type="text"
						name={`row_kicker_${index}`}
						bind:value={row.kicker}
						maxlength="40"
					/>
				</label>
				<div class="portal-row-actions">
					<button
						class="button button-secondary"
						type="submit"
						name="move"
						value={`up:${row.id}`}
						disabled={index === 0}
						aria-label={`Subir ${row.title}`}
						onclick={(event) => moveWithKeyboard(event, index, -1)}>Subir</button
					>
					<button
						class="button button-secondary"
						type="submit"
						name="move"
						value={`down:${row.id}`}
						disabled={index === ordered.length - 1}
						aria-label={`Bajar ${row.title}`}
						onclick={(event) => moveWithKeyboard(event, index, 1)}>Bajar</button
					>
				</div>
			</fieldset>
		{/each}
	</div>

	<div class="portal-rows-actions">
		<button class="button" type="submit">Guardar filas</button>
		<button
			class="button button-secondary"
			type="submit"
			formaction="?/reset"
			formnovalidate
			name="key"
			value="ui.portal_rows">Restablecer por defecto</button
		>
	</div>
</form>

<style>
	.portal-rows-editor {
		display: grid;
		gap: 0.8rem;
		margin-top: 1rem;
	}

	.portal-row-list {
		display: grid;
		gap: 0.7rem;
	}

	.portal-row-editor {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(12rem, 1fr));
		gap: 0.65rem;
		margin: 0;
		padding: 0.8rem;
		border: 1px solid var(--borde-suave);
		border-radius: 4px;
		background: var(--superficie-suave);
	}

	.portal-row-editor legend {
		padding: 0 0.25rem;
		color: var(--granate);
		font-family: 'Fraunces', Georgia, serif;
		font-size: 1rem;
		font-weight: 600;
	}

	.portal-row-editor > label {
		display: grid;
		gap: 0.25rem;
		color: var(--granate);
		font-size: 0.65rem;
		font-weight: 700;
		letter-spacing: 0.1em;
		text-transform: uppercase;
	}

	.portal-row-editor input:not([type='checkbox']) {
		width: 100%;
		padding: 0.5rem 0.6rem;
		border: 1px solid var(--borde-medio);
		border-radius: 3px;
		background: var(--crema-clara);
		color: var(--tinta);
		font-family: 'Inter', system-ui, sans-serif;
		font-size: 0.92rem;
		letter-spacing: normal;
		text-transform: none;
	}

	.portal-row-toggle {
		display: flex !important;
		align-items: center;
		gap: 0.4rem;
	}

	.portal-row-toggle input {
		width: 1rem;
		height: 1rem;
		accent-color: var(--naranja);
	}

	.portal-row-actions {
		display: flex;
		flex-wrap: wrap;
		gap: 0.4rem;
		align-items: center;
	}

	.portal-row-actions .button {
		margin: 0;
		padding: 0.45rem 0.65rem;
		font-size: 0.64rem;
	}

	.portal-rows-announcement {
		margin: 0;
		min-height: 1.2rem;
		color: var(--texto-apagado);
		font-size: 0.8rem;
	}

	.portal-rows-actions {
		display: flex;
		flex-wrap: wrap;
		gap: 0.65rem;
	}

	.portal-rows-actions .button {
		margin: 0;
	}

	@media (max-width: 520px) {
		.portal-rows-actions {
			flex-direction: column;
		}

		.portal-rows-actions .button {
			width: 100%;
		}
	}
</style>
