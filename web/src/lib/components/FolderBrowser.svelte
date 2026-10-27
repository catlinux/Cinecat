<script lang="ts">
	import { untrack } from 'svelte';
	import type { FSBrowseResponse, FSEntry, FSRoot } from '$lib/api';
	import { breadcrumbs, browserParent } from '$lib/libraries';

	let {
		open = false,
		roots: rootsProp = null,
		listing: listingProp = null,
		error: errorProp = '',
		loading: loadingProp = false,
		onclose = () => {},
		onselect = (_path: string) => {}
	}: {
		open?: boolean;
		roots?: FSRoot[] | null;
		listing?: FSBrowseResponse | null;
		error?: string;
		loading?: boolean;
		onclose?: () => void;
		onselect?: (path: string) => void;
	} = $props();

	let dialogEl: HTMLDialogElement | null = $state(null);
	let fetchedRoots = $state<FSRoot[]>([]);
	let fetchedListing = $state<FSBrowseResponse | null>(null);
	let error = $state(untrack(() => errorProp));
	let loading = $state(untrack(() => loadingProp));
	let wasOpen = false;

	const roots = $derived(rootsProp ?? fetchedRoots);
	const listing = $derived(listingProp ?? fetchedListing);
	const crumbs = $derived(listing ? breadcrumbs(listing.path) : []);
	const parent = $derived(browserParent(listing));

	$effect(() => {
		if (typeof window === 'undefined') {
			return;
		}

		if (open && !wasOpen) {
			error = errorProp;
			if (!listingProp) {
				fetchedListing = null;
			}
			if (!rootsProp && fetchedRoots.length === 0) {
				void loadRoots();
			}
			if (dialogEl) {
				try {
					if (!dialogEl.open || !dialogEl.matches(':modal')) {
						dialogEl.showModal();
					}
				} catch {
					// El navegador ya lo ha abierto o no admite diálogos modales.
				}
				queueMicrotask(() => focusFirst());
			}
		}

		if (!open && wasOpen && dialogEl?.open) {
			dialogEl.close();
		}
		wasOpen = open;
	});

	async function loadRoots() {
		loading = true;
		error = '';
		try {
			const response = await fetch('/admin/fs', {
				headers: { Accept: 'application/json' },
				cache: 'no-store'
			});
			const data = (await response.json()) as unknown;
			if (!response.ok) {
				throw new Error(errorMessage(data, 'No se pudieron cargar las carpetas.'));
			}
			fetchedRoots = ((data as { roots?: FSRoot[] }).roots ?? []) as FSRoot[];
		} catch (reason) {
			error = reason instanceof Error ? reason.message : 'No se pudieron cargar las carpetas.';
		} finally {
			loading = false;
		}
	}

	async function browse(path: string) {
		loading = true;
		error = '';
		try {
			const response = await fetch(`/admin/fs?path=${encodeURIComponent(path)}`, {
				headers: { Accept: 'application/json' },
				cache: 'no-store'
			});
			const data = (await response.json()) as unknown;
			if (!response.ok) {
				throw new Error(errorMessage(data, 'No se pudo abrir la carpeta.'));
			}
			fetchedListing = data as FSBrowseResponse;
		} catch (reason) {
			error = reason instanceof Error ? reason.message : 'No se pudo abrir la carpeta.';
		} finally {
			loading = false;
		}
	}

	function errorMessage(data: unknown, fallback: string): string {
		if (typeof data === 'object' && data !== null) {
			const record = data as Record<string, unknown>;
			if (typeof record.error === 'string' && record.error.trim() !== '') {
				return record.error;
			}
			if (typeof record.message === 'string' && record.message.trim() !== '') {
				return record.message;
			}
		}
		return fallback;
	}

	function choose() {
		if (!listing) {
			return;
		}
		onselect(listing.path);
		onclose();
	}

	function goUp() {
		if (parent) {
			void browse(parent);
		}
	}

	function close() {
		onclose();
	}

	function handleCancel(event: Event) {
		event.preventDefault();
		close();
	}

	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Escape') {
			event.preventDefault();
			close();
			return;
		}

		if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
			const entries = focusableEntries();
			if (entries.length === 0) {
				return;
			}
			event.preventDefault();
			const current = entries.indexOf(document.activeElement as HTMLElement);
			const next = event.key === 'ArrowDown' ? current + 1 : current - 1;
			const index = current < 0 ? 0 : (next + entries.length) % entries.length;
			entries[index].focus();
			return;
		}

		if (event.key === 'Tab' && dialogEl) {
			const focusable = focusableElements();
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
	}

	function focusFirst() {
		const first = focusableEntries()[0] ?? focusableElements()[0];
		first?.focus();
	}

	function focusableEntries(): HTMLElement[] {
		return dialogEl
			? Array.from(dialogEl.querySelectorAll<HTMLElement>('[data-fs-entry]')).filter(
					(element) => !element.hasAttribute('disabled')
				)
			: [];
	}

	function focusableElements(): HTMLElement[] {
		if (!dialogEl) {
			return [];
		}
		return Array.from(
			dialogEl.querySelectorAll<HTMLElement>(
				'button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'
			)
		).filter((element) => element.offsetParent !== null || element === document.activeElement);
	}

	function entryLabel(entry: FSEntry): string {
		return entry.name;
	}
</script>

<dialog
	bind:this={dialogEl}
	open={open}
	class="folder-browser"
	aria-label="Elegir carpeta"
	aria-modal="true"
	onkeydown={handleKeydown}
	oncancel={handleCancel}
>
	<div class="folder-browser-body">
		<div class="folder-browser-head">
			<div>
				<p class="kicker">Servidor</p>
				<h2>Elegir carpeta</h2>
			</div>
			<button type="button" class="button button-secondary folder-close" onclick={close}>
				Cerrar
			</button>
		</div>

		{#if listing}
			<nav class="folder-breadcrumbs" aria-label="Ruta de la carpeta">
				{#each crumbs as crumb, index (crumb.path)}
					{#if index > 0}
						<span aria-hidden="true">/</span>
					{/if}
					<button
						type="button"
						class="breadcrumb-button"
						onclick={() => void browse(crumb.path)}
					>
						{crumb.name}
					</button>
				{/each}
			</nav>

			<div class="folder-current">
				<button type="button" class="button button-secondary" disabled={!parent} onclick={goUp}>
					Subir
				</button>
				<span>{listing.path}</span>
			</div>
		{/if}

		{#if loading}
			<p class="admin-note" role="status">Cargando…</p>
		{/if}

		{#if error}
			<p class="error" role="alert">{error}</p>
		{/if}

		{#if listing}
			{#if listing.entries.length === 0}
				<p class="admin-note">No hay subcarpetas</p>
			{:else}
				<ul class="folder-entries">
					{#each listing.entries as entry (entry.path)}
						<li>
							<button
								type="button"
								class="folder-entry"
								data-fs-entry
								onclick={() => void browse(entry.path)}
							>
								<span>{entryLabel(entry)}</span>
								<span class="folder-entry-path">{entry.path}</span>
							</button>
						</li>
					{/each}
				</ul>
			{/if}

			{#if listing.truncated}
				<p class="admin-note">Se muestran las primeras 1000 carpetas</p>
			{/if}

			<button type="button" class="button folder-choose" onclick={choose}>
				Elegir esta carpeta
			</button>
		{:else}
			{#if !loading && !error && roots.length === 0}
				<p class="admin-note">No hay raíces disponibles.</p>
			{/if}
			{#if roots.length > 0}
				<p class="admin-note">Elige una carpeta raíz:</p>
				<ul class="folder-entries">
					{#each roots as root (root.path)}
						<li>
							<button
								type="button"
								class="folder-entry"
								data-fs-entry
								onclick={() => void browse(root.path)}
							>
								<span>{root.name}</span>
								<span class="folder-entry-path">{root.path}</span>
							</button>
						</li>
					{/each}
				</ul>
			{/if}
		{/if}
	</div>
</dialog>
