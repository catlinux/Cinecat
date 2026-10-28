<script lang="ts">
	import BackupPanel from '$lib/components/BackupPanel.svelte';
	import PortalRowsEditor from '$lib/components/PortalRowsEditor.svelte';
	import SettingsForm from '$lib/components/SettingsForm.svelte';
	import UpdatePanel from '$lib/components/UpdatePanel.svelte';
	import { formatDateTime } from '$lib/media';
	import { parsePortalRows, type PortalRow, type SettingsFormState } from '$lib/settings';
	import type { UpdateActionState } from '$lib/updates';
	import type { PageData } from './$types';

	interface BackupActionState {
		backupAction?: boolean;
		success?: boolean;
		message?: string;
	}

	interface PortalRowsActionState {
		portalRowsAction?: boolean;
		success?: boolean;
		message?: string;
		portalRows?: PortalRow[];
	}

	interface SettingsPageForm
		extends SettingsFormState,
			UpdateActionState,
			BackupActionState,
			PortalRowsActionState {}

	let { data, form = null }: { data: PageData; form?: SettingsPageForm | null } = $props();

	const section = $derived(data.section);
	const updateForm = $derived(form?.updateAction ? form : null);
	const backupForm = $derived(form?.backupAction ? form : null);
	const portalRowsForm = $derived(form?.portalRowsAction ? form : null);
	const settingsForm = $derived(
		updateForm
			? {
					success: updateForm.success,
					fields: updateForm.fields,
					values: updateForm.values
				}
			: backupForm || portalRowsForm
				? null
				: form
	);
	const portalRows = $derived.by(() => {
		const setting = section.settings.find((item) => item.key === 'ui.portal_rows');
		return parsePortalRows(setting?.value);
	});
</script>

<svelte:head>
	<title>{section.label} · Administración · {data.ui?.server_name ?? 'Cinecat'}</title>
</svelte:head>

<section class="admin settings-page">
	<div class="admin-head">
		<p class="kicker">Administración</p>
		<h1>{section.label}</h1>
		<p class="admin-note">Los cambios se aplican sin reiniciar el servidor.</p>
	</div>

	<SettingsForm settings={section.settings} form={settingsForm as SettingsFormState | null} />

	{#if section.key === 'ui'}
		<section class="admin-card system-card">
			<p class="kicker">Interfaz</p>
			<h2>Filas del portal</h2>
			<p class="admin-note">
				Ordena, oculta o cambia los títulos de las filas de la página de inicio.
			</p>
			<PortalRowsEditor rows={portalRows} form={portalRowsForm} />
		</section>
	{/if}

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

			<UpdatePanel
				info={data.updateInfo}
				currentVersion={data.serverInfo?.server ?? data.webVersion}
				form={updateForm}
			/>
		</section>

		<section class="admin-card system-card">
			<p class="kicker">Servidor</p>
			<h2>Copias de seguridad</h2>
			<BackupPanel
				info={data.backupInfo}
				loadError={data.backupLoadError}
				form={backupForm}
			/>
		</section>
	{/if}
</section>
