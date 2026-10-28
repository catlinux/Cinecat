<script lang="ts">
	import type { Session, SessionClient } from '$lib/api';
	import { formatDateTime } from '$lib/media';

	type HiddenFields =
		| Record<string, string | number>
		| ((session: Session) => Record<string, string | number>);

	let {
		sessions = [],
		formAction = '?/closeSession',
		hiddenFields = {},
		currentLabel = 'Este dispositivo'
	}: {
		sessions?: Session[] | null;
		formAction?: string;
		hiddenFields?: HiddenFields;
		currentLabel?: string;
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

	@media (max-width: 520px) {
		.session-item {
			align-items: flex-start;
			flex-direction: column;
		}
	}
</style>
