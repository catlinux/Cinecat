<script lang="ts">
	type FormState = { username?: string; message?: string } | null;

	let {
		data = { hasSession: false },
		form = null
	}: {
		data?: { hasSession: boolean; ui?: { server_name: string } };
		form?: FormState;
	} = $props();
</script>

<svelte:head>
	<title>Entrar · {data?.ui?.server_name ?? 'Cinecat'}</title>
</svelte:head>

<section class="login">
	<div class="login-card">
		<p class="kicker">Taquilla</p>
		<h1>Entrar</h1>
		<p class="login-lede">
			Tu butaca te espera. Accede con el usuario y la contraseña de tu biblioteca Cinecat.
		</p>

		{#if form?.message}
			<p class="error" role="alert">{form.message}</p>
		{/if}

		<form method="POST" action="?/login" class="login-form">
			<label>
				<span>Usuario</span>
				<input
					type="text"
					name="username"
					autocomplete="username"
					autocapitalize="none"
					required
					value={form?.username ?? ''}
				/>
			</label>

			<label>
				<span>Contraseña</span>
				<input type="password" name="password" autocomplete="current-password" required />
			</label>

			<button type="submit">Entrar en la sala</button>
		</form>

		{#if data?.hasSession}
			<form method="POST" action="?/logout" class="logout-form">
				<button type="submit" class="secondary">Cerrar sesión</button>
			</form>
		{/if}

		<p class="login-note">Sesión de vida corta · Cookie httpOnly</p>
	</div>
</section>
