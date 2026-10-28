<script lang="ts">
	import SessionList from '$lib/components/SessionList.svelte';
	import type { Session } from '$lib/api';

	type FormState = { message?: string; success?: boolean; closeSession?: boolean } | null;

	let {
		data = {},
		form = null
	}: {
		data?: { ui?: { server_name: string }; sessions?: Session[] | null };
		form?: FormState;
	} = $props();
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

		{#if form?.message && !form.closeSession}
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
		<SessionList
			sessions={data.sessions}
			formAction="?/closeSession"
			hiddenFields={(session) => ({
				id: session.id,
				_current: session.current ? '1' : '0'
			})}
			currentLabel="Este dispositivo"
		/>
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
</style>
