<script lang="ts">
	import type { ScanStatus } from '$lib/api';

	let { status }: { status: ScanStatus } = $props();
</script>

{#if status.running}
	<div class="scan-result scan-running" role="status">
		<p><strong>Escaneo en curso…</strong></p>
	</div>
{:else if status.started_at}
	<div class="scan-result" role="status">
		{#if status.error}
			<p class="error" role="alert">{status.error}</p>
		{/if}

		<p class="scan-counts">
			<strong>{status.files}</strong> archivos encontrados ·
			<strong>{status.upserted}</strong> entradas actualizadas ·
			<strong>{status.identified}</strong> identificadas ·
			<strong>{status.unidentified}</strong> sin identificar
		</p>

		<details class="scan-warnings">
			<summary>Avisos ({status.warnings.length})</summary>
			{#if status.warnings.length > 0}
				<ul>
					{#each status.warnings as warning}
						<li>{warning}</li>
					{/each}
				</ul>
			{:else}
				<p class="admin-note">Sin avisos.</p>
			{/if}
		</details>
	</div>
{/if}

<style>
	.scan-counts {
		line-height: 1.7;
	}

	.scan-warnings {
		margin-top: 0.75rem;
	}

	.scan-warnings summary {
		cursor: pointer;
		font-size: 0.82rem;
		font-weight: 600;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		color: var(--granate);
	}

	.scan-warnings ul {
		font-size: 0.85rem;
		line-height: 1.5;
	}
</style>
