<script lang="ts">
	import { versionMismatch } from '$lib/version';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	function countLabel(value: number | null): string {
		return value === null ? '—' : String(value);
	}

	function relativeTime(value: string | null): string {
		if (!value) {
			return 'sin fecha';
		}
		const timestamp = Date.parse(value);
		if (Number.isNaN(timestamp)) {
			return value;
		}
		const seconds = Math.max(0, Math.floor((Date.now() - timestamp) / 1000));
		if (seconds < 60) {
			return `hace ${seconds} s`;
		}
		const minutes = Math.floor(seconds / 60);
		if (minutes < 60) {
			return `hace ${minutes} min`;
		}
		const hours = Math.floor(minutes / 60);
		if (hours < 24) {
			return `hace ${hours} h`;
		}
		return `hace ${Math.floor(hours / 24)} días`;
	}

	const counts = $derived(data.counts);
	const mismatch = $derived(
		versionMismatch(data.serverInfo?.server ?? null, data.webVersion)
	);
</script>

<svelte:head>
	<title>Administración · {data.ui?.server_name ?? 'Cinecat'}</title>
</svelte:head>

<section class="admin admin-summary">
	<div class="admin-head">
		<p class="kicker">Administración</p>
		<h1>Resumen</h1>
		<p class="admin-note">El estado del servidor y de la biblioteca, de un vistazo.</p>
	</div>

	<div class="admin-summary-grid">
		<article class="admin-card admin-stat-card">
			<p class="kicker">Versiones</p>
			<h2>Servidor y web</h2>
			<dl class="admin-facts">
				<div>
					<dt>Servidor</dt>
					<dd>{data.serverInfo?.server ?? '—'}</dd>
				</div>
				<div>
					<dt>Web</dt>
					<dd>{data.webVersion}</dd>
				</div>
				{#if data.serverInfo?.commit}
					<div>
						<dt>Commit</dt>
						<dd>{data.serverInfo.commit}</dd>
					</div>
				{/if}
			</dl>
			{#if mismatch}
				<p class="version-warning" role="status">
					La web y el servidor tienen versiones distintas: recarga o actualiza.
				</p>
			{/if}
		</article>

		<article class="admin-card admin-stat-card admin-scan-card">
			<p class="kicker">Cabina de proyección</p>
			<h2>Último escaneo</h2>
			{#if data.scan}
				{#if data.scan.running}
					<p class="scan-state scan-state--running" role="status">Escaneo en curso…</p>
				{:else}
					<p class="scan-state">
						Terminado {relativeTime(data.scan.finished_at)}
					</p>
					<p class="admin-scan-counts">
						<strong>{data.scan.files}</strong> archivos ·
						<strong>{data.scan.identified}</strong> películas identificadas ·
						<strong>{data.scan.unidentified}</strong> sin identificar
					</p>
					{#if data.scan.series_identified !== undefined || data.scan.series_unidentified !== undefined}
						<p class="admin-scan-counts">
							<strong>{data.scan.series_identified ?? 0}</strong> series identificadas ·
							<strong>{data.scan.series_unidentified ?? 0}</strong> sin identificar
						</p>
					{/if}
					{#if data.scan.error}
						<p class="error" role="alert">{data.scan.error}</p>
					{/if}
				{/if}
			{:else}
				<p class="admin-note">Sin datos del último escaneo.</p>
			{/if}
			<a class="button" href="/admin/libraries">Ir a Bibliotecas</a>
		</article>

		<article class="admin-card admin-stat-card">
			<p class="kicker">Estantes</p>
			<h2>Biblioteca</h2>
			<dl class="admin-facts">
				<div>
					<dt>Películas</dt>
					<dd><a href="/movies">{countLabel(counts.movies)}</a></dd>
				</div>
				<div>
					<dt>Series</dt>
					<dd><a href="/series">{countLabel(counts.series)}</a></dd>
				</div>
				<div>
					<dt>Colecciones</dt>
					<dd><a href="/collections">{countLabel(counts.collections)}</a></dd>
				</div>
			</dl>
			{#if counts.unmatchedMovies !== null && counts.unmatchedMovies > 0}
				<p class="admin-note">
					<a class="text-link" href="/movies?unmatched=1">
						{counts.unmatchedMovies} películas sin identificar
					</a>
				</p>
			{/if}
			{#if counts.unmatchedSeries !== null && counts.unmatchedSeries > 0}
				<p class="admin-note">
					<a class="text-link" href="/series?unmatched=1">
						{counts.unmatchedSeries} series sin identificar
					</a>
				</p>
			{/if}
		</article>
	</div>
</section>
