<script lang="ts">
	import { enhance } from '$app/forms';
	import { untrack } from 'svelte';
	import SessionList from '$lib/components/SessionList.svelte';
	import { formatDateTime } from '$lib/media';
	import type { Library, ManagedUser, Session, UserRole } from '$lib/api';
	import type { PageData } from './$types';

	type ManagedUserWithSessions = ManagedUser & { sessions?: Session[] | null };

	type CreateState = {
		username: string;
		role: string;
		all_libraries: boolean;
		library_ids: number[];
	};

	type UsersForm = {
		success?: boolean;
		message?: string;
		action?: string;
		userId?: number;
		create?: CreateState;
	} | null;

	let { data, form = null }: { data: PageData; form?: UsersForm } = $props();

	const libraries = $derived((data.libraries ?? []) as Library[]);
	const users = $derived((data.users ?? []) as ManagedUserWithSessions[]);

	let createUsername = $state(untrack(() => form?.create?.username ?? ''));
	let createPassword = $state('');
	let createRepeatPassword = $state('');
	let createRole = $state<UserRole>(
		untrack(() => (form?.create?.role === 'admin' ? 'admin' : 'user'))
	);
	let createAllLibraries = $state(untrack(() => form?.create?.all_libraries ?? false));
	let createLibraryIDs = $state<number[]>(
		untrack(() => form?.create?.library_ids ?? [])
	);
	let createError = $state('');

	$effect(() => {
		if (!form || form.action !== 'create') {
			return;
		}
		if (form.success) {
			createUsername = '';
			createPassword = '';
			createRepeatPassword = '';
			createRole = 'user';
			createAllLibraries = false;
			createLibraryIDs = [];
			createError = '';
			return;
		}
		if (form.create) {
			createUsername = form.create.username;
			createRole = form.create.role === 'admin' ? 'admin' : 'user';
			createAllLibraries = form.create.all_libraries;
			createLibraryIDs = [...form.create.library_ids];
		}
	});

	const isOnlyAdmin = $derived(users.filter((user) => user.role === 'admin' && !user.disabled).length === 1);
	const currentUsername = $derived(data.currentUser?.username ?? '');

	function libraryName(id: number): string {
		return libraries.find((library) => library.id === id)?.name ?? `Biblioteca ${id}`;
	}

	function librarySummary(user: ManagedUser): string {
		if (user.all_libraries) {
			return 'Todas';
		}
		if (!user.library_ids || user.library_ids.length === 0) {
			return 'Ninguna';
		}
		return user.library_ids.map(libraryName).join(', ');
	}

	function roleLabel(role: UserRole): string {
		return role === 'admin' ? 'Administrador' : 'Usuario';
	}

	function isCurrent(user: ManagedUser): boolean {
		return user.username === currentUsername;
	}

	function toggleCreateLibrary(id: number, checked: boolean): void {
		createLibraryIDs = checked
			? [...createLibraryIDs.filter((current) => current !== id), id]
			: createLibraryIDs.filter((current) => current !== id);
	}

	function validateCreate(event: SubmitEvent): void {
		createError = '';
		if (createRole === 'user' && !createAllLibraries && createLibraryIDs.length === 0) {
			event.preventDefault();
			createError = 'Elige al menos una biblioteca o marca «Todas, también las que se creen en el futuro».';
		}
	}

	function submitEnhance() {
		return async ({ update }: { update: () => Promise<void> }) => {
			await update();
		};
	}
</script>

<svelte:head>
	<title>Usuarios · Administración · {data.ui?.server_name ?? 'Cinecat'}</title>
</svelte:head>

<section class="admin users-page">
	<div class="admin-head">
		<p class="kicker">Administración</p>
		<h1>Usuarios</h1>
		<p class="admin-note">
			Crea cuentas y decide qué bibliotecas puede ver cada persona. El administrador ve todas.
		</p>
	</div>

	{#if form?.message}
		<p class={form.success ? 'success' : 'error'} role={form.success ? 'status' : 'alert'}>
			{form.message}
		</p>
	{/if}

	<div class="admin-card new-user-card">
		<h2>Nuevo usuario</h2>
		<form method="POST" action="?/create" class="login-form" use:enhance={submitEnhance} onsubmit={validateCreate}>
			<label>
				<span>Nombre</span>
				<input
					type="text"
					name="username"
					bind:value={createUsername}
					required
					maxlength="40"
					autocomplete="off"
				/>
			</label>

			<label>
				<span>Contraseña</span>
				<input
					type="password"
					name="password"
					bind:value={createPassword}
					required
					minlength="8"
					autocomplete="new-password"
				/>
			</label>

			<label>
				<span>Repetir contraseña</span>
				<input
					type="password"
					name="repeat_password"
					bind:value={createRepeatPassword}
					required
					minlength="8"
					autocomplete="new-password"
				/>
			</label>

			<label>
				<span>Rol</span>
				<select
					name="role"
					value={createRole}
					onchange={(event) => (createRole = (event.currentTarget as HTMLSelectElement).value as UserRole)}
				>
					<option value="user">Usuario</option>
					<option value="admin">Administrador</option>
				</select>
			</label>

			{#if createRole === 'admin'}
				<p class="admin-note role-warning" role="status">
					Un administrador puede cambiar todo el catálogo, las bibliotecas y el resto de cuentas.
				</p>
			{/if}

			<fieldset class="user-library-picker">
				<legend>Bibliotecas visibles</legend>
				<label class="check-line">
					<input
						type="checkbox"
						name="all_libraries"
						value="true"
						bind:checked={createAllLibraries}
					/>
					<span>Todas, también las que se creen en el futuro</span>
				</label>
				<div class="library-check-grid">
					{#each libraries as library (library.id)}
						<label class="check-line">
							<input
								type="checkbox"
								name="library_ids"
								value={library.id}
								checked={createLibraryIDs.includes(library.id)}
								onchange={(event) =>
									toggleCreateLibrary(
										library.id,
										(event.currentTarget as HTMLInputElement).checked
									)}
							/>
							<span>{library.name}</span>
						</label>
					{/each}
				</div>
				{#if libraries.length === 0}
					<p class="admin-note">Todavía no hay bibliotecas creadas.</p>
				{/if}
			</fieldset>

			{#if createError}
				<p class="error" role="alert">{createError}</p>
			{/if}

			<button type="submit">Crear usuario</button>
		</form>
	</div>

	{#if users.length === 0}
		<div class="admin-card">
			<h2>Sin usuarios</h2>
			<p class="admin-note">Todavía no hay usuarios. Crea el primero con el formulario de arriba.</p>
		</div>
	{:else}
		<ul class="admin-list user-list">
			{#each users as user (user.id)}
				<li class="admin-item user-card">
					<div class="user-heading">
						<div>
							<p class="admin-item-name">
								{user.username}
								{#if isCurrent(user)}
									<span class="user-badge">Tú</span>
								{/if}
								<span class="user-badge role-{user.role}">{roleLabel(user.role)}</span>
								<span class:user-disabled={user.disabled} class="user-badge">
									{user.disabled ? 'Desactivado' : 'Activo'}
								</span>
							</p>
							<p class="admin-item-path">
								Bibliotecas: {librarySummary(user)} · {user.active_sessions}
								{user.active_sessions === 1 ? 'sesión abierta' : 'sesiones abiertas'} · Alta:
								{user.created_at ? formatDateTime(user.created_at) : '—'}
							</p>
						</div>
					</div>

					<details class="user-sessions">
						<summary>
							Sesiones ({user.sessions === null ? '—' : (user.sessions?.length ?? 0)})
							{#if user.sessions === null}
								<span class="user-sessions-warning">No se pudieron cargar</span>
							{/if}
						</summary>
						<SessionList
							sessions={user.sessions}
							formAction="?/closeSession"
							hiddenFields={(session) => ({
								user_id: user.id,
								session_id: session.id
							})}
							currentLabel="Este dispositivo"
						/>
					</details>

					<div class="user-actions">
						<form method="POST" action="?/libraries" class="user-form">
							<input type="hidden" name="id" value={user.id} />
							<input type="hidden" name="all_libraries_present" value="1" />
							<fieldset class="user-library-picker compact">
								<legend>Bibliotecas</legend>
								<label class="check-line">
									<input
										type="checkbox"
										name="all_libraries"
										value="true"
										checked={user.all_libraries}
									/>
									<span>Todas, también las futuras</span>
								</label>
								<div class="library-check-grid">
									{#each libraries as library (library.id)}
										<label class="check-line">
											<input
												type="checkbox"
												name="library_ids"
												value={library.id}
												checked={(user.library_ids ?? []).includes(library.id)}
											/>
											<span>{library.name}</span>
										</label>
									{/each}
								</div>
							</fieldset>
							<button class="button button-secondary" type="submit">Guardar bibliotecas</button>
						</form>

						<form method="POST" action="?/password" class="user-form">
							<input type="hidden" name="id" value={user.id} />
							<p class="user-form-title">Restablecer contraseña</p>
							<p class="admin-note">Cierra todas sus sesiones.</p>
							<label>
								<span>Contraseña nueva</span>
								<input
									type="password"
									name="password"
									minlength="8"
									required
									autocomplete="new-password"
								/>
							</label>
							<label>
								<span>Repetir contraseña</span>
								<input
									type="password"
									name="repeat_password"
									minlength="8"
									required
									autocomplete="new-password"
								/>
							</label>
							<button class="button button-secondary" type="submit">Restablecer</button>
						</form>

						{#if !isCurrent(user)}
							<form method="POST" action="?/update" class="user-form">
								<input type="hidden" name="id" value={user.id} />
								<label>
									<span>Cambiar rol</span>
									<select name="role" value={user.role}>
										<option value="user">Usuario</option>
										<option value="admin">Administrador</option>
									</select>
								</label>
								<button class="button button-secondary" type="submit">Guardar rol</button>
								<p class="admin-note">Cambiar el rol cierra sus sesiones.</p>
							</form>

							<form method="POST" action="?/update" class="user-form">
								<input type="hidden" name="id" value={user.id} />
								<input type="hidden" name="disabled" value={user.disabled ? 'false' : 'true'} />
								<button class={user.disabled ? 'button' : 'button-danger'} type="submit">
									{user.disabled ? 'Activar cuenta' : 'Desactivar cuenta'}
								</button>
								<p class="admin-note">
									{user.disabled
										? 'La cuenta podrá volver a iniciar sesión.'
										: 'Se cerrarán sus sesiones y no podrá iniciar sesión.'}
								</p>
							</form>

							<details class="user-delete-confirm" id={`user-delete-${user.id}`}>
								<summary class="button-danger">Eliminar cuenta</summary>
								<p class="admin-note">
									Se borrarán la cuenta {user.username}, su visto, su progreso y sus listas.
								</p>
								<form method="POST" action="?/delete" use:enhance={submitEnhance}>
									<input type="hidden" name="id" value={user.id} />
									<label class="delete-confirm-field">
										<span>Escribe «{user.username}» para confirmar</span>
										<input type="text" name="confirmation" autocomplete="off" required />
									</label>
									<button class="button-danger" type="submit">
										Eliminar definitivamente
									</button>
								</form>
							</details>
						{/if}
					</div>
				</li>
			{/each}
		</ul>
	{/if}

	{#if isOnlyAdmin}
		<p class="admin-note">
			Solo queda un administrador activo: no se puede desactivar, borrar ni quitar el rol a esa
			cuenta.
		</p>
	{/if}
</section>

<style>
	.users-page {
		width: min(72rem, 100%);
	}
	.new-user-card {
		margin-bottom: 1.5rem;
	}
	.user-list {
		gap: 1rem;
	}
	.user-card {
		display: grid;
		gap: 1rem;
		padding: 1rem;
		border: 1px solid var(--borde-suave);
		border-radius: 4px;
		background: var(--superficie);
	}
	.user-heading {
		display: flex;
		justify-content: space-between;
		gap: 1rem;
	}
	.user-badge {
		display: inline-block;
		margin-left: 0.35rem;
		padding: 0.12rem 0.4rem;
		border-radius: 999px;
		background: var(--tinte);
		color: var(--texto-tenue);
		font-family: 'Inter', system-ui, sans-serif;
		font-size: 0.58rem;
		font-weight: 700;
		letter-spacing: 0.08em;
		text-transform: uppercase;
	}
	.role-admin {
		background: var(--granate);
		color: var(--crema-clara);
	}
	.role-user {
		background: var(--tinte);
		color: var(--granate);
	}
	.user-disabled {
		background: var(--naranja);
		color: var(--crema-clara);
	}
	.user-actions {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(14rem, 1fr));
		gap: 1rem;
	}
	.user-form {
		display: grid;
		align-content: start;
		gap: 0.55rem;
		padding: 0.8rem;
		border: 1px solid var(--borde-suave);
		border-radius: 4px;
		background: var(--superficie-suave);
	}
	.user-form > label {
		display: grid;
		gap: 0.25rem;
		font-size: 0.62rem;
		font-weight: 700;
		letter-spacing: 0.1em;
		text-transform: uppercase;
		color: var(--granate);
	}
	.user-form input:not([type='checkbox']),
	.user-form select {
		width: 100%;
		padding: 0.55rem 0.65rem;
		border: 1px solid var(--borde-medio);
		border-radius: 3px;
		background: var(--crema-clara);
		color: var(--tinta);
		font-family: 'Inter', system-ui, sans-serif;
		font-size: 0.92rem;
		letter-spacing: normal;
		text-transform: none;
	}
	.user-form-title {
		margin: 0;
		color: var(--granate);
		font-family: 'Fraunces', Georgia, serif;
		font-size: 1rem;
		font-weight: 600;
	}
	.user-form .button,
	.user-form .button-danger {
		width: fit-content;
		margin-top: 0.2rem;
	}
	.user-library-picker {
		display: grid;
		gap: 0.45rem;
		margin: 0;
		padding: 0;
		border: 0;
	}
	.user-library-picker legend {
		margin-bottom: 0.35rem;
		color: var(--granate);
		font-size: 0.65rem;
		font-weight: 700;
		letter-spacing: 0.12em;
		text-transform: uppercase;
	}
	.user-library-picker.compact {
		gap: 0.3rem;
	}
	.library-check-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(12rem, 1fr));
		gap: 0.3rem 0.8rem;
	}
	.check-line {
		display: flex;
		align-items: center;
		gap: 0.45rem;
		color: var(--texto-tenue);
		font-family: 'Inter', system-ui, sans-serif;
		font-size: 0.8rem;
		font-weight: 400;
		letter-spacing: normal;
		text-transform: none;
	}
	.check-line input {
		width: 1rem;
		height: 1rem;
		accent-color: var(--naranja);
	}
	.login-form .check-line,
	.user-form .check-line {
		display: flex;
		align-items: center;
		gap: 0.45rem;
		font-size: 0.8rem;
		font-weight: 400;
		letter-spacing: normal;
		text-transform: none;
	}
	.login-form .check-line input,
	.user-form .check-line input {
		width: 1rem;
		height: 1rem;
		padding: 0;
		border: 0;
		background: transparent;
		accent-color: var(--naranja);
	}
	.role-warning {
		padding: 0.6rem 0.7rem;
		border-left: 3px solid var(--naranja);
		background: var(--tinte);
	}
	.user-sessions {
		padding: 0.7rem 0.8rem;
		border: 1px solid var(--borde-suave);
		border-radius: 4px;
		background: var(--superficie-suave);
	}
	.user-sessions summary {
		cursor: pointer;
		color: var(--granate);
		font-size: 0.72rem;
		font-weight: 700;
		letter-spacing: 0.1em;
		text-transform: uppercase;
	}
	.user-sessions-warning {
		margin-left: 0.4rem;
		color: var(--naranja);
		font-size: 0.62rem;
		letter-spacing: 0.04em;
	}
	.user-delete-confirm {
		align-self: start;
		padding: 0.6rem 0.75rem;
		border: 1px solid var(--borde-suave);
		border-radius: 4px;
		background: var(--superficie-suave);
	}
	.user-delete-confirm summary {
		cursor: pointer;
		list-style: none;
	}
	.user-delete-confirm summary::-webkit-details-marker {
		display: none;
	}
	.user-delete-confirm form {
		display: grid;
		gap: 0.6rem;
		margin-top: 0.7rem;
	}
	.delete-confirm-field {
		display: grid;
		gap: 0.3rem;
		font-size: 0.68rem;
		font-weight: 700;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		color: var(--granate);
	}
	.delete-confirm-field input {
		padding: 0.55rem 0.65rem;
		border: 1px solid var(--borde-medio);
		border-radius: 3px;
		background: var(--crema-clara);
		color: var(--tinta);
		font-family: 'Inter', system-ui, sans-serif;
		font-size: 0.92rem;
		letter-spacing: normal;
		text-transform: none;
	}
	@media (max-width: 620px) {
		.user-actions {
			grid-template-columns: 1fr;
		}
	}
</style>
