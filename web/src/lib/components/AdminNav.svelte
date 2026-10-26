<script module lang="ts">
	export interface AdminNavItem {
		href: string;
		label: string;
		soon?: boolean;
	}

	export const adminNavItems: AdminNavItem[] = [
		{ href: '/admin', label: 'Resumen' },
		{ href: '/admin/libraries', label: 'Bibliotecas' },
		{ href: '/admin/settings/scan', label: 'Escaneo' },
		{ href: '/admin/settings/metadata', label: 'Metadatos' },
		{ href: '/admin/settings/ui', label: 'Interfaz' },
		{ href: '/admin/settings/playback', label: 'Reproducción' },
		{ href: '/admin/settings/system', label: 'Sistema' },
		{ href: '/admin/users', label: 'Usuarios', soon: true }
	];
</script>

<script lang="ts">
	let { pathname = '' }: { pathname?: string } = $props();

	function isActive(href: string): boolean {
		if (href === '/admin') {
			return pathname === '/admin';
		}
		return pathname === href || pathname.startsWith(`${href}/`);
	}
</script>

<nav class="admin-nav" aria-label="Administración">
	<p class="admin-nav-title">Administración</p>
	<ul>
		{#each adminNavItems as item (item.href)}
			<li>
				{#if item.soon}
					<span class="admin-nav-link admin-nav-soon" aria-disabled="true">
						{item.label}
						<span class="admin-nav-soon-tag">Pronto</span>
					</span>
				{:else}
					<a
						class="admin-nav-link"
						class:admin-nav-active={isActive(item.href)}
						href={item.href}
						aria-current={isActive(item.href) ? 'page' : undefined}
					>
						{item.label}
					</a>
				{/if}
			</li>
		{/each}
	</ul>
</nav>
