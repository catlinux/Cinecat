<script lang="ts">
	import SessionList from '$lib/components/SessionList.svelte';
	import type { HomePreferenceRow, Session } from '$lib/api';

	type FormState = {
		message?: string;
		success?: boolean;
		closeSession?: boolean;
		deviceProfile?: boolean;
		portalRows?: boolean;
	} | null;

	let {
		data = {},
		form = null
	}: {
		data?: {
			ui?: { server_name: string };
			sessions?: Session[] | null;
			portalRows?: HomePreferenceRow[] | null;
		};
		form?: FormState;
	} = $props();

	const portalRows = $derived(Array.isArray(data.portalRows) ? data.portalRows : []);
	const portalUnavailable = $derived(!Array.isArray(data.portalRows));
</script>

<svelte:head>
	<title>Tu cuenta · {data.ui?.server_name ?? 'Cinecat'}</title>
</svelte:head>

<a class="back" href="/movies">← Volver a la cartelera</a>

<section class="login">
	<div class="login-card">
		<p class="kicker">Taquilla</p>
		<h1>Tu cuenta</h1>
		<p class="login-lede">
			Cambia tu contraseña. Se cerrarán tus sesiones abiertas en otros dispositivos; la de este
			dispositivo seguirá abierta.
		</p>

		{#if form?.message && !form.closeSession && !form.deviceProfile && !form.portalRows}
			<p class={form.success ? 'success' : 'error'} role={form.success ? 'status' : 'alert'}>
				{form.message}
			</p>
		{/if}

		<form method="POST" class="login-form">
			<label>
				<span>Contraseña actual</span>
				<input
					type="password"
					name="current_password"
					autocomplete="current-password"
					required
				/>
			</label>

			<label>
				<span>Contraseña nueva</span>
				<input
					type="password"
					name="new_password"
					autocomplete="new-password"
					minlength="8"
					required
				/>
			</label>

			<label>
				<span>Repetir contraseña nueva</span>
				<input
					type="password"
					name="repeat_password"
					autocomplete="new-password"
					minlength="8"
					required
				/>
			</label>

			<button type="submit">Guardar contraseña</button>
		</form>

		<p class="login-note">Mínimo 8 caracteres · Cookie httpOnly</p>

		<form method="POST" action="/login?/logout" class="logout-form">
			<button type="submit" class="secondary">Cerrar sesión</button>
		</form>
	</div>
</section>

<section class="login account-sessions-section">
	<div class="login-card account-sessions-card">
		<p class="kicker">Seguridad</p>
		<h2>Sesiones abiertas</h2>
		{#if form?.closeSession && form.message}
			<p class={form.success ? 'success' : 'error'} role={form.success ? 'status' : 'alert'}>
				{form.message}
			</p>
		{/if}
		{#if form?.deviceProfile && form.message}
			<p class={form.success ? 'success' : 'error'} role={form.success ? 'status' : 'alert'}>
				{form.message}
			</p>
		{/if}
		<SessionList
			sessions={data.sessions}
			formAction="?/closeSession"
			hiddenFields={(session) => ({
				id: session.id,
				_current: session.current ? '1' : '0'
			})}
			currentLabel="Este dispositivo"
			profileAction="?/saveDeviceProfile"
		/>
		<p class="account-vincular">
			<a class="text-link" href="/vincular">Vincular un dispositivo</a>
		</p>
	</div>
</section>

<section class="login account-portal-section">
	<div class="login-card account-portal-card">
		<p class="kicker">Inicio</p>
		<h2>Portal de inicio</h2>
		<p class="admin-note">Marca las filas que quieres ocultar en tu portal.</p>
		{#if form?.portalRows && form.message}
			<p class={form.success ? 'success' : 'error'} role={form.success ? 'status' : 'alert'}>
				{form.message}
			</p>
		{/if}
		{#if portalUnavailable}
			<p class="admin-note">No se pudieron cargar las filas del portal.</p>
		{:else}
			<form method="POST" action="?/portal" class="portal-rows-form">
				<div class="portal-row-list">
					{#each portalRows as row (row.id)}
						<label class="portal-row-choice">
							<input type="checkbox" name="hidden" value={row.id} checked={row.hidden} />
							<span>{row.title}</span>
						</label>
					{/each}
				</div>
				<button type="submit" class="secondary">Guardar filas del portal</button>
			</form>
		{/if}
	</div>
</section>

<style>
	.account-sessions-section {
		width: min(38rem, 100%);
		margin-top: 1.5rem;
	}

	.account-sessions-card {
		text-align: left;
	}

	.account-sessions-card h2 {
		margin: 0.3rem 0 0.6rem;
		font-family: 'Fraunces', Georgia, serif;
		font-size: clamp(1.6rem, 4vw, 2.2rem);
		font-weight: 600;
		color: var(--granate);
	}

	.account-vincular {
		margin: 1rem 0 0;
	}

	.account-portal-section {
		width: min(38rem, 100%);
		margin-top: 1.5rem;
	}

	.account-portal-card {
		text-align: left;
	}

	.account-portal-card h2 {
		margin: 0.3rem 0 0.6rem;
		font-family: 'Fraunces', Georgia, serif;
		font-size: clamp(1.6rem, 4vw, 2.2rem);
		font-weight: 600;
		color: var(--granate);
	}

	.portal-rows-form {
		display: grid;
		gap: 0.9rem;
		margin-top: 0.8rem;
	}

	.portal-row-list {
		display: grid;
		gap: 0.35rem;
	}

	.portal-row-choice {
		display: flex;
		align-items: center;
		gap: 0.55rem;
		padding: 0.4rem 0.45rem;
		border: 1px solid var(--borde-suave);
		border-radius: 4px;
		color: var(--texto-suave);
		font-size: 0.88rem;
	}

	.portal-row-choice input {
		accent-color: var(--granate);
	}
</style>
