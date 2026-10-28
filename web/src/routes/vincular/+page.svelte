<script lang="ts">
	import { formatDateTime } from '$lib/media';
	import type { PageData } from './$types';

	type PairForm = {
		step?: 'preview' | 'approved';
		code?: string;
		deviceName?: string;
		client?: 'android' | 'androidtv';
		expiresAt?: string;
		message?: string;
	} | null;

	let { data, form = null }: { data: PageData; form?: PairForm } = $props();

	let errorElement = $state<HTMLParagraphElement | null>(null);

	const deviceName = $derived(form?.deviceName || 'Dispositivo sin nombre');
	const initialCode = $derived(form?.code ?? data.initialCode ?? '');

	$effect(() => {
		if (form?.message) {
			errorElement?.focus();
		}
	});

	function clientLabel(client: string | undefined): string {
		return client === 'androidtv' ? 'Android TV' : 'Android';
	}

	function formatCode(event: Event): void {
		const input = event.currentTarget as HTMLInputElement;
		const clean = input.value
			.toUpperCase()
			.replace(/[^A-Z0-9]/g, '')
			.slice(0, 8);
		input.value = clean.length > 4 ? `${clean.slice(0, 4)}-${clean.slice(4)}` : clean;
	}
</script>

<svelte:head>
	<title>Vincular dispositivo · {data.ui?.server_name ?? 'Cinecat'}</title>
</svelte:head>

<section class="login vincular-page">
	<div class="login-card">
		<p class="kicker">Acceso</p>
		<h1>Vincular dispositivo</h1>

		{#if form?.step === 'approved'}
			<p class="success" role="status">
				Listo. Ya puedes usar Cinecat en «{deviceName}». Puedes cerrar esta sesión cuando quieras en
				<a class="text-link" href="/account">Tu cuenta</a>.
			</p>
		{:else if form?.step === 'preview'}
			<p class="login-lede">Revisa el dispositivo antes de darle acceso a tu cuenta.</p>
			<dl class="admin-facts pair-facts">
				<div>
					<dt>Dispositivo</dt>
					<dd>{deviceName}</dd>
				</div>
				<div>
					<dt>Tipo</dt>
					<dd>{clientLabel(form.client)}</dd>
				</div>
				{#if form.expiresAt}
					<div>
						<dt>Caduca</dt>
						<dd>{formatDateTime(form.expiresAt)}</dd>
					</div>
				{/if}
			</dl>

			<div class="pair-warning">
				<p>
					Aprueba solo un código que aparezca ahora mismo en la pantalla de tu dispositivo. Si alguien te ha pasado este código, no lo apruebes: daría acceso a tu cuenta.
				</p>
			</div>

			<div class="pair-actions">
				<form method="POST" action="?/approve">
					<input type="hidden" name="code" value={form.code ?? ''} />
					<button class="button" type="submit">Vincular</button>
				</form>
				<form method="GET" action="/vincular">
					<button class="button button-secondary" type="submit">Cancelar</button>
				</form>
			</div>
		{:else}
			<p class="login-lede">
				Escribe el código que aparece en la pantalla de tu televisor o dispositivo Android.
			</p>

			{#if form?.message}
				<p class="error" role="alert" tabindex="-1" bind:this={errorElement}>
					{form.message}
				</p>
			{/if}

			<form method="POST" action="?/preview" class="login-form">
				<label>
					<span>Código del dispositivo</span>
					<input
						type="text"
						name="code"
						value={initialCode}
						oninput={formatCode}
						autocapitalize="characters"
						autocomplete="off"
						inputmode="text"
						maxlength="9"
						placeholder="XXXX-XXXX"
						required
					/>
				</label>
				<button type="submit">Continuar</button>
			</form>
		{/if}
	</div>
</section>

<style>
	.vincular-page {
		width: min(34rem, 100%);
	}

	.vincular-page .login-card {
		text-align: left;
	}

	.vincular-page h1 {
		font-size: clamp(2.2rem, 7vw, 3.4rem);
	}

	.pair-facts {
		margin-top: 1rem;
	}

	.pair-warning {
		margin-top: 1rem;
		padding: 0.8rem;
		border-left: 4px solid var(--naranja);
		border-radius: 2px;
		background: var(--tinte);
	}

	.pair-warning p {
		margin: 0;
		color: var(--texto-tenue);
		font-size: 0.88rem;
		line-height: 1.5;
	}

	.pair-actions {
		display: flex;
		flex-wrap: wrap;
		gap: 0.65rem;
		margin-top: 1rem;
	}

	.pair-actions form {
		margin: 0;
	}

	.pair-actions .button {
		margin: 0;
	}

	@media (max-width: 460px) {
		.pair-actions {
			flex-direction: column;
		}

		.pair-actions .button {
			width: 100%;
		}
	}
</style>
