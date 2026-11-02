<script lang="ts">
	import type { Session, SessionClient, SessionProfile } from '$lib/api';
	import { formatDateTime } from '$lib/media';

	type HiddenFields =
		| Record<string, string | number>
		| ((session: Session) => Record<string, string | number>);

	type ProfileOption = {
		value: string;
		label: string;
	};

	const MAX_QUALITY_OPTIONS: ProfileOption[] = [
		{ value: '', label: 'Sin límite' },
		{ value: '480', label: '480p' },
		{ value: '720', label: '720p' },
		{ value: '1080', label: '1080p' }
	];

	const LANGUAGE_OPTIONS: ProfileOption[] = [
		{ value: '', label: 'Sin preferencia' },
		{ value: 'es', label: 'Español' },
		{ value: 'ca', label: 'Catalán' },
		{ value: 'en', label: 'Inglés' },
		{ value: 'fr', label: 'Francés' },
		{ value: 'de', label: 'Alemán' },
		{ value: 'it', label: 'Italiano' },
		{ value: 'pt', label: 'Portugués' },
		{ value: 'ja', label: 'Japonés' }
	];

	const SUBTITLE_LANGUAGE_OPTIONS: ProfileOption[] = [
		{ value: '', label: 'Sin preferencia' },
		{ value: 'off', label: 'Desactivados' },
		...LANGUAGE_OPTIONS.slice(1)
	];

	let {
		sessions = [],
		formAction = '?/closeSession',
		hiddenFields = {},
		currentLabel = 'Este dispositivo',
		profileAction = null
	}: {
		sessions?: Session[] | null;
		formAction?: string;
		hiddenFields?: HiddenFields;
		currentLabel?: string;
		profileAction?: string | null;
	} = $props();

	function clientLabel(client: SessionClient): string {
		switch (client) {
			case 'android':
				return 'Android';
			case 'androidtv':
				return 'Android TV';
			default:
				return 'Navegador';
		}
	}

	function fieldsFor(session: Session): Record<string, string | number> {
		return typeof hiddenFields === 'function' ? hiddenFields(session) : hiddenFields;
	}

	function profileFor(session: Session): SessionProfile {
		return {
			max_quality: session.profile?.max_quality ?? '',
			audio_language: session.profile?.audio_language ?? '',
			subtitle_language: session.profile?.subtitle_language ?? ''
		};
	}

	function languageOptions(selected: string, subtitle: boolean): ProfileOption[] {
		const options = subtitle ? SUBTITLE_LANGUAGE_OPTIONS : LANGUAGE_OPTIONS;
		if (selected && !options.some((option) => option.value === selected)) {
			return [...options, { value: selected, label: selected }];
		}
		return options;
	}
</script>

<div class="session-list">
	{#if sessions === null}
		<p class="error" role="alert">No se pudieron cargar las sesiones.</p>
	{:else if sessions.length === 0}
		<p class="admin-note">No hay sesiones abiertas.</p>
	{:else}
		<ul class="session-items">
			{#each sessions as session (session.id)}
				<li class="session-item">
					<div class="session-info">
						<p class="session-device">
							<strong>{session.device_name || 'Dispositivo sin nombre'}</strong>
							{#if session.current}
								<span class="session-badge">{currentLabel}</span>
							{/if}
						</p>
						<p class="session-meta">
							<span>{clientLabel(session.client)}</span>
							<span>Última actividad: {formatDateTime(session.last_seen_at) || '—'}</span>
							<span>Caduca: {formatDateTime(session.expires_at) || '—'}</span>
						</p>
					</div>
					<form class="session-close" method="POST" action={formAction}>
						{#each Object.entries(fieldsFor(session)) as [name, value] (name)}
							<input type="hidden" {name} value={value} />
						{/each}
						<button class="button button-secondary" type="submit">
							{session.current ? 'Cerrar esta sesión' : 'Cerrar'}
						</button>
					</form>
					{#if profileAction}
						{@const profile = profileFor(session)}
						<details class="session-profile">
							<summary class="button button-secondary session-settings">Ajustes</summary>
							<form method="POST" action={profileAction} class="session-profile-form">
								<input type="hidden" name="id" value={session.id} />
								<label class="session-profile-field">
									<span>Calidad máxima</span>
									<select name="max_quality">
										{#each MAX_QUALITY_OPTIONS as option (option.value)}
											<option value={option.value} selected={option.value === profile.max_quality}>
												{option.label}
											</option>
										{/each}
									</select>
								</label>
								<label class="session-profile-field">
									<span>Idioma de audio preferido</span>
									<select name="audio_language">
										{#each languageOptions(profile.audio_language, false) as option (option.value)}
											<option value={option.value} selected={option.value === profile.audio_language}>
												{option.label}
											</option>
										{/each}
									</select>
								</label>
								<label class="session-profile-field">
									<span>Subtítulos por defecto</span>
									<select name="subtitle_language">
										{#each languageOptions(profile.subtitle_language, true) as option (option.value)}
											<option value={option.value} selected={option.value === profile.subtitle_language}>
												{option.label}
											</option>
										{/each}
									</select>
								</label>
								<button class="button session-profile-save" type="submit">Guardar</button>
								<p class="session-profile-help">Se aplican al reproducir en este dispositivo cuando no hayas elegido otra cosa en la película.</p>
							</form>
						</details>
					{/if}
				</li>
			{/each}
		</ul>
	{/if}
</div>

<style>
	.session-list {
		margin-top: 1rem;
		text-align: left;
	}

	.session-items {
		display: grid;
		gap: 0.6rem;
		margin: 0;
		padding: 0;
		list-style: none;
	}

	.session-item {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		justify-content: space-between;
		gap: 0.75rem;
		padding: 0.75rem;
		border: 1px solid var(--borde-suave);
		border-radius: 4px;
		background: var(--superficie-suave);
	}

	.session-info {
		min-width: 0;
	}

	.session-device {
		margin: 0;
		color: var(--granate);
		font-family: 'Fraunces', Georgia, serif;
		font-size: 1rem;
		font-weight: 600;
		overflow-wrap: anywhere;
	}

	.session-badge {
		display: inline-block;
		margin-left: 0.4rem;
		padding: 0.12rem 0.45rem;
		border-radius: 999px;
		background: var(--tinte);
		color: var(--granate);
		font-family: 'Inter', system-ui, sans-serif;
		font-size: 0.58rem;
		font-weight: 700;
		letter-spacing: 0.08em;
		text-transform: uppercase;
	}

	.session-meta {
		display: flex;
		flex-wrap: wrap;
		gap: 0.2rem 0.8rem;
		margin: 0.25rem 0 0;
		color: var(--texto-apagado);
		font-size: 0.78rem;
		line-height: 1.45;
	}

	.session-close {
		margin: 0;
	}

	.session-close .button {
		margin: 0;
		padding: 0.5rem 0.8rem;
		font-size: 0.68rem;
	}

	.session-profile {
		flex-basis: 100%;
		margin: 0;
	}

	.session-profile > summary {
		display: inline-flex;
		align-items: center;
		margin: 0;
		padding: 0.5rem 0.8rem;
		font-size: 0.68rem;
		list-style: none;
		cursor: pointer;
	}

	.session-profile > summary::-webkit-details-marker {
		display: none;
	}

	.session-profile[open] > summary {
		margin-bottom: 0.7rem;
	}

	.session-profile-form {
		display: grid;
		gap: 0.7rem;
		padding: 0.8rem;
		border: 1px solid var(--borde-suave);
		border-radius: 4px;
		background: var(--superficie);
	}

	.session-profile-field {
		display: grid;
		gap: 0.3rem;
		color: var(--texto-apagado);
		font-size: 0.76rem;
	}

	.session-profile-field select {
		width: 100%;
		padding: 0.5rem 0.6rem;
		border: 1px solid var(--borde-medio);
		border-radius: 3px;
		background: var(--superficie);
		color: var(--texto-suave);
		font: inherit;
		font-size: 0.88rem;
	}

	.session-profile-save {
		justify-self: start;
		margin: 0;
		padding: 0.5rem 0.8rem;
		font-size: 0.68rem;
	}

	.session-profile-help {
		margin: 0;
		color: var(--texto-apagado);
		font-size: 0.74rem;
		line-height: 1.45;
	}

	@media (max-width: 520px) {
		.session-item {
			align-items: flex-start;
			flex-direction: column;
		}
	}
</style>
