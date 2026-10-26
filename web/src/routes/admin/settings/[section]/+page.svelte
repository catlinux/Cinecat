<script lang="ts">
	import SettingsForm from '$lib/components/SettingsForm.svelte';
	import { formatDateTime } from '$lib/media';
	import { UPDATE_COMMAND, type UpdateCheck } from '$lib/updates';
	import type { SettingsFormState } from '$lib/settings';
	import type { PageData } from './$types';

	interface SettingsPageForm extends SettingsFormState {
		update?: UpdateCheck | null;
		command?: string;
	}

	let { data, form = null }: { data: PageData; form?: SettingsPageForm | null } = $props();

	const section = $derived(data.section);
	const checkUpdatesEnabled = $derived(
		section.settings.find((setting) => setting.key === 'system.check_updates')?.value === true
	);
	const checkResult = $derived(form && 'update' in form ? form : null);
	const settingsForm = $derived(
		checkResult ? { success: checkResult.success, fields: checkResult.fields, values: checkResult.values } : form
	);
</script>

<svelte:head>
	<title>{section.label} · Administración · Cinecat</title>
</svelte:head>

<section class="admin settings-page">
	<div class="admin-head">
		<p class="kicker">Administración</p>
		<h1>{section.label}</h1>
		<p class="admin-note">Los cambios se aplican sin reiniciar el servidor.</p>
	</div>

	<SettingsForm settings={section.settings} form={settingsForm as SettingsFormState | null} />

	{#if section.key === 'system'}
		<section class="admin-card system-card">
			<p class="kicker">Servidor</p>
			<h2>Versiones y actualizaciones</h2>
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
				{#if data.serverInfo?.built_at}
					<div>
						<dt>Compilada</dt>
						<dd>{formatDateTime(data.serverInfo.built_at)}</dd>
					</div>
				{/if}
			</dl>

			{#if checkUpdatesEnabled}
				<form method="POST" action="?/checkUpdates" class="system-check">
					<button class="button button-secondary" type="submit">Comprobar actualizaciones</button>
				</form>
				{#if checkResult?.update}
					<p class="success" role="status">{checkResult.update.message}</p>
					{#if !checkResult.update.upToDate}
						<p class="admin-note">
							Para actualizar, ejecuta <code>{checkResult.command ?? UPDATE_COMMAND}</code>.
						</p>
					{/if}
				{:else if checkResult && checkResult.message}
					<p class="error" role="alert">{checkResult.message}</p>
				{/if}
			{:else}
				<p class="admin-note">
					La búsqueda de actualizaciones es una conexión saliente opcional. Activa «Buscar
					actualizaciones» y guarda los cambios para poder comprobarla.
				</p>
			{/if}
		</section>
	{/if}
</section>
