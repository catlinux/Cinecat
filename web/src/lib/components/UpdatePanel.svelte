<script lang="ts">
	import { enhance } from '$app/forms';
	import { onMount } from 'svelte';
	import type { SubmitFunction } from '@sveltejs/kit';
	import type { UpdateInfo } from '$lib/api';
	import { formatDateTime } from '$lib/media';
	import {
		UPDATE_INSTALL_COMMAND,
		UPDATE_MANUAL_COMMAND,
		UPDATE_TRACKING_LIMIT,
		elapsedSeconds,
		formatElapsed,
		isUpdateInProgress,
		phaseLabel,
		pollDelay,
		trackingExpired,
		updateOutcome,
		updatedTag,
		type UpdateActionState
	} from '$lib/updates';

	let {
		info = null,
		currentVersion = '',
		form = null,
		pollUrl = '/admin/update',
		initialPollError = false
	}: {
		info?: UpdateInfo | null;
		currentVersion?: string;
		form?: UpdateActionState | null;
		pollUrl?: string;
		initialPollError?: boolean;
	} = $props();

	let live = $state<UpdateInfo | null>(null);
	let tracking = $state(false);
	let restarting = $state(false);
	let pollFailures = $state(0);
	let timedOut = $state(false);
	let pollError = $state('');
	let elapsed = $state(0);
	let startedAt = $state(Date.now());
	let autoStarted = $state(false);
	let pollTimer: ReturnType<typeof setTimeout> | null = null;
	let clockTimer: ReturnType<typeof setInterval> | null = null;
	let reloadTimer: ReturnType<typeof setTimeout> | null = null;
	let dialogEl: HTMLDialogElement | null = $state(null);
	let openButton: HTMLButtonElement | null = $state(null);
	let cancelButton: HTMLButtonElement | null = $state(null);
	let jsReady = $state(false);
	let submitting = $state(false);
	let dialogOpen = $state(false);

	const currentInfo = $derived(live ?? form?.updateInfo ?? info);
	const current = $derived(currentInfo?.current || currentVersion || '—');
	const latest = $derived(currentInfo?.latest ?? '');
	const available = $derived(Boolean(currentInfo?.available && latest));
	const checkedAt = $derived(currentInfo?.checked_at ?? '');
	const notes = $derived(currentInfo?.notes ?? '');
	const applySupported = $derived(Boolean(currentInfo?.apply_supported));
	const showInstallHelp = $derived(
		Boolean(form?.installHelp) || (currentInfo !== null && !applySupported)
	);
	const checkEnabled = $derived(currentInfo?.check_enabled !== false);
	const running = $derived(isUpdateInProgress(currentInfo?.status));
	const expectedTag = $derived(form?.appliedTag ?? null);
	const showTracking = $derived(tracking || running || Boolean(form?.appliedTag));
	const isRestarting = $derived(restarting || (initialPollError && live === null));
	const outcome = $derived(updateOutcome(currentInfo, expectedTag));
	const doneTag = $derived(updatedTag(currentInfo, expectedTag));
	const trackingPhase = $derived(
		expectedTag && currentInfo?.status.result ? '' : currentInfo?.status.phase
	);

	$effect(() => {
		const runningNow = form?.updateInfo ? isUpdateInProgress(form.updateInfo.status) : false;
		if ((form?.appliedTag || runningNow) && !tracking && !autoStarted) {
			autoStarted = true;
			beginTracking();
		}
	});

	onMount(() => {
		jsReady = true;
		if (initialPollError) {
			restarting = true;
			pollFailures = 1;
		}
		const runningInitial = isUpdateInProgress(info?.status);
		if (runningInitial || form?.appliedTag) {
			autoStarted = true;
			tracking = true;
			startTimers(runningInitial ? undefined : Date.now());
			startPolling();
		}
		return stopAllTimers;
	});

	function startTimers(startOverride?: number): void {
		if (typeof window === 'undefined') {
			return;
		}
		stopTrackingTimers();
		const parsed = Date.parse(currentInfo?.status?.started_at ?? '');
		startedAt = startOverride ?? (Number.isNaN(parsed) ? Date.now() : parsed);
		elapsed = elapsedSeconds(new Date(startedAt).toISOString(), Date.now());
		clockTimer = setInterval(() => {
			elapsed = elapsedSeconds(new Date(startedAt).toISOString(), Date.now());
			if (trackingExpired(new Date(startedAt).toISOString(), Date.now())) {
				timedOut = true;
				stopTrackingTimers();
			}
		}, 1000);
	}

	function startPolling(): void {
		pollFailures = 0;
		schedulePoll(0);
	}

	function beginTracking(): void {
		if (typeof window === 'undefined') {
			return;
		}
		tracking = true;
		restarting = false;
		timedOut = false;
		pollError = '';
		startedAt = Date.now();
		startTimers(startedAt);
		startPolling();
	}

	function schedulePoll(delay: number): void {
		if (typeof window === 'undefined' || !tracking) {
			return;
		}
		if (pollTimer !== null) {
			clearTimeout(pollTimer);
		}
		pollTimer = setTimeout(() => {
			void poll();
		}, delay);
	}

	async function poll(): Promise<void> {
		if (!tracking) {
			return;
		}
		if (Date.now() - startedAt >= UPDATE_TRACKING_LIMIT) {
			timedOut = true;
			stopTrackingTimers();
			return;
		}

		try {
			const response = await fetch(pollUrl, {
				headers: { Accept: 'application/json' },
				cache: 'no-store'
			});
			if (!response.ok) {
				// Apache responde 503 y no 502 mientras el contenedor web se recrea.
				if (response.status === 502 || response.status === 503 || response.status === 504) {
					restarting = true;
					pollFailures += 1;
					schedulePoll(pollDelay(pollFailures));
					return;
				}
				pollError =
					response.status === 401
						? 'La sesión ha caducado. Vuelve a iniciar sesión.'
						: 'No se pudo consultar el estado de la actualización.';
				stopTrackingTimers();
				return;
			}

			const data = (await response.json()) as UpdateInfo;
			live = data;
			restarting = false;
			pollFailures = 0;
			pollError = '';

			const status = data.status;
			if (isUpdateInProgress(status)) {
				schedulePoll(pollDelay(0));
				return;
			}
			const expected = form?.appliedTag ?? null;
			if (expected) {
				const result = updateOutcome(data, expected);
				if (result === 'ok') {
					tracking = false;
					stopTrackingTimers();
					reloadTimer = setTimeout(() => window.location.reload(), 5000);
					return;
				}
				if (result === 'failed' || result === 'rolled_back') {
					stopTrackingTimers();
					return;
				}
				schedulePoll(pollDelay(0));
				return;
			}
			if (status.result === 'ok' && updatedTag(data)) {
				tracking = false;
				stopTrackingTimers();
				reloadTimer = setTimeout(() => window.location.reload(), 5000);
				return;
			}
			stopTrackingTimers();
		} catch {
			restarting = true;
			pollFailures += 1;
			schedulePoll(pollDelay(pollFailures));
		}
	}

	function stopTrackingTimers(): void {
		if (pollTimer !== null) {
			clearTimeout(pollTimer);
			pollTimer = null;
		}
		if (clockTimer !== null) {
			clearInterval(clockTimer);
			clockTimer = null;
		}
	}

	function stopAllTimers(): void {
		stopTrackingTimers();
		if (reloadTimer !== null) {
			clearTimeout(reloadTimer);
			reloadTimer = null;
		}
	}

	function openDialog(): void {
		if (!dialogEl || submitting) {
			return;
		}
		submitting = false;
		dialogOpen = true;
		try {
			if (typeof dialogEl.showModal === 'function') {
				dialogEl.showModal();
			}
		} catch {
			// El atributo open deja el diálogo disponible si showModal no funciona.
		}
		queueMicrotask(() => cancelButton?.focus());
	}

	function closeDialog(): void {
		if (!dialogEl) {
			return;
		}
		dialogOpen = false;
		if (typeof dialogEl.close === 'function' && dialogEl.open) {
			dialogEl.close();
		}
		openButton?.focus();
	}

	function handleDialogClose(): void {
		dialogOpen = false;
		openButton?.focus();
	}

	function handleDialogCancel(event: Event): void {
		event.preventDefault();
		closeDialog();
	}

	function handleDialogClick(event: MouseEvent): void {
		if (event.target === dialogEl) {
			closeDialog();
		}
	}

	function dialogFocusables(): HTMLElement[] {
		if (!dialogEl) {
			return [];
		}
		return Array.from(
			dialogEl.querySelectorAll<HTMLElement>(
				'button:not([disabled]), [href], input:not([disabled]):not([type="hidden"]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'
			)
		).filter((element) => !element.hasAttribute('hidden'));
	}

	function handleDialogKeydown(event: KeyboardEvent): void {
		if (event.key === 'Escape') {
			event.preventDefault();
			closeDialog();
			return;
		}
		if (event.key !== 'Tab') {
			return;
		}
		const focusable = dialogFocusables();
		if (focusable.length === 0) {
			return;
		}
		const first = focusable[0];
		const last = focusable[focusable.length - 1];
		if (event.shiftKey && document.activeElement === first) {
			event.preventDefault();
			last.focus();
		} else if (!event.shiftKey && document.activeElement === last) {
			event.preventDefault();
			first.focus();
		}
	}

	function handleApplySubmit(event: SubmitEvent): void {
		event.preventDefault();
		if (submitting) {
			return;
		}
		submitting = true;
		closeDialog();
	}

	const applyUpdateEnhance: SubmitFunction = () => {
		submitting = true;
		closeDialog();
		return async ({ update }) => {
			try {
				await update();
			} finally {
				submitting = false;
				openButton?.focus();
			}
		};
	};

	function reloadPage(): void {
		window.location.reload();
	}
</script>

<div class="update-panel">
	<p class="update-current">Versión actual: <strong>{current}</strong></p>

	<form method="POST" action="?/checkUpdates" class="system-check" use:enhance>
		<button class="button button-secondary" type="submit">Buscar actualizaciones</button>
	</form>
	{#if !checkEnabled}
		<p class="admin-note update-hint">
			La búsqueda automática está desactivada; puedes comprobar a mano
		</p>
	{/if}

	{#if form?.message}
		<p class={form.success ? 'success' : 'error'} role={form.success ? 'status' : 'alert'}>
			{form.message}
		</p>
	{:else if pollError}
		<p class="error" role="alert">{pollError}</p>
	{/if}

	{#if checkedAt}
		<p class={available ? 'update-result update-result--new' : 'update-result'} role="status">
			{#if available && latest}
				Hay una versión nueva: {latest}
			{:else}
				Estás al día ({current})
			{/if}
		</p>
		<p class="admin-note">Comprobado: {formatDateTime(checkedAt)}</p>
	{/if}

	{#if available}
		<details class="update-notes">
			<summary>Notas de la versión</summary>
			<pre>{notes || 'Sin notas disponibles.'}</pre>
		</details>
	{/if}

	{#if available && applySupported && !showTracking}
		<button
			class="button update-js-trigger"
			type="button"
			hidden={!jsReady}
			disabled={submitting}
			aria-haspopup="dialog"
			aria-controls="update-dialog"
			bind:this={openButton}
			onclick={openDialog}>Actualizar ahora</button
		>

		<dialog
			bind:this={dialogEl}
			id="update-dialog"
			class="update-dialog"
			open={dialogOpen}
			aria-labelledby="update-dialog-title"
			aria-describedby="update-dialog-description"
			aria-modal="true"
			oncancel={handleDialogCancel}
			onclose={handleDialogClose}
			onclick={handleDialogClick}
			onkeydown={handleDialogKeydown}
		>
			<div class="update-dialog-body">
				<h2 id="update-dialog-title">Actualizar a la versión {latest}</h2>
				<p id="update-dialog-description">
					Se hará una copia de la base de datos, se reiniciará el servidor durante unos
					minutos y, si algo falla, se vuelve a la versión anterior. ¿Continuar?
				</p>
				<div class="update-dialog-actions">
					<button
						class="button button-secondary"
						type="button"
						bind:this={cancelButton}
						onclick={closeDialog}>Cancelar</button
					>
					<form
						class="update-dialog-form"
						method="POST"
						action="?/applyUpdate"
						use:enhance={applyUpdateEnhance}
						onsubmit={handleApplySubmit}
					>
						<input type="hidden" name="tag" value={latest} />
						<button class="button" type="submit" disabled={submitting}>
							Actualizar ahora
						</button>
					</form>
				</div>
			</div>
		</dialog>

		<noscript hidden={jsReady}>
			<details class="update-confirm update-fallback">
				<summary class="update-fallback-toggle">Actualizar ahora</summary>
				<div class="update-confirm-body">
					<p>
						Se hará una copia de la base de datos, se reiniciará el servidor durante unos
						minutos y, si algo falla, se vuelve a la versión anterior. ¿Continuar?
					</p>
					<form method="POST" action="?/applyUpdate">
						<input type="hidden" name="tag" value={latest} />
						<button class="button" type="submit">Actualizar ahora</button>
					</form>
				</div>
			</details>
		</noscript>
	{/if}

	{#if showInstallHelp}
		<div class="update-unsupported">
			<p class="admin-note">La actualización automática no está instalada en este servidor</p>
			<p class="admin-note">Para instalarla:</p>
			<pre class="update-command"><code>{UPDATE_INSTALL_COMMAND}</code></pre>
			<p class="admin-note">Para actualizar a mano:</p>
			<pre class="update-command"><code>{UPDATE_MANUAL_COMMAND}</code></pre>
		</div>
	{/if}

	{#if timedOut}
		<p class="error" role="alert">
			La actualización lleva más de 30 minutos sin terminar. Revisa el servicio en el servidor.
		</p>
	{:else if outcome === 'failed'}
		<p class="error" role="alert">La actualización falló. Revisa el servicio en el servidor.</p>
		{#if currentInfo?.status.message}
			<p class="admin-note">{currentInfo.status.message}</p>
		{/if}
	{:else if outcome === 'rolled_back'}
		<p class="error" role="alert">
			La actualización falló y se volvió a la versión anterior
		</p>
		{#if currentInfo?.status.message}
			<p class="admin-note">{currentInfo.status.message}</p>
		{/if}
	{:else if doneTag}
		<p class="success" role="status">Actualizado a {doneTag}</p>
		<button class="button" type="button" onclick={reloadPage}>Recargar la página</button>
	{:else if showTracking}
		<div class="update-tracking">
			{#if isRestarting}
				<p class="update-phase" aria-live="polite" aria-atomic="true">
					Reiniciando el servidor…
				</p>
				<p class="admin-note">Se está aplicando la actualización; es normal que la web no responda.</p>
			{:else}
				<p class="update-phase" aria-live="polite" aria-atomic="true">
					{phaseLabel(trackingPhase)}
				</p>
				<p class="admin-note">Tiempo transcurrido: {formatElapsed(elapsed)}</p>
			{/if}
			<div
				class="update-progress"
				role="progressbar"
				aria-label="Actualización en curso"
			></div>
		</div>
	{:else if currentInfo?.status.result === 'ok'}
		<p class="success" role="status">
			La actualización terminó. Recarga la página para comprobar la versión.
		</p>
		<button class="button" type="button" onclick={reloadPage}>Recargar la página</button>
	{/if}

	{#if showTracking || outcome}
		<noscript>
			<a class="text-link update-refresh" href="?">Actualizar estado</a>
		</noscript>
	{/if}
</div>
