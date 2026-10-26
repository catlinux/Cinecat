<script lang="ts">
	type FormState = { message?: string; success?: boolean } | null;

	let { form = null }: { form?: FormState } = $props();
</script>

<svelte:head>
	<title>Tu cuenta · Cinecat</title>
</svelte:head>

<a class="back" href="/movies">← Volver a la cartelera</a>

<section class="login">
	<div class="login-card">
		<p class="kicker">Taquilla</p>
		<h1>Tu cuenta</h1>
		<p class="login-lede">
			Cambia la contraseña de tu biblioteca. La sesión abierta en este dispositivo seguirá
			funcionando.
		</p>

		{#if form?.message}
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
	</div>
</section>
