<script lang="ts">
	let { art }: { art: 'movies' | 'collections' | 'series' | 'videos' } = $props();
</script>

<div class="category-art" aria-hidden="true">
	<svg viewBox="0 0 320 220" role="presentation" focusable="false">
		{#if art === 'movies'}
			<rect x="18" y="20" width="284" height="180" rx="4" fill="var(--granate-oscuro)" opacity="0.62" />
			<rect x="34" y="36" width="252" height="148" fill="none" stroke="var(--dorado)" stroke-width="2" />
			<path d="M58 56h204M58 164h204" stroke="var(--dorado)" stroke-width="1" opacity="0.55" />
			{#each [48, 86, 124, 162, 200, 238, 276] as x}
				<circle cx={x} cy="46" r="3" fill="var(--dorado)" />
				<circle cx={x} cy="174" r="3" fill="var(--dorado)" />
			{/each}
			<rect x="82" y="72" width="156" height="82" fill="var(--crema-clara)" opacity="0.12" />
			<path
				d="M160 76l12 26 28 3-21 19 6 27-25-14-25 14 6-27-21-19 28-3z"
				fill="var(--dorado)"
				opacity="0.86"
			/>
			<path d="M96 190h128" stroke="var(--crema-clara)" stroke-width="2" opacity="0.35" />
		{:else if art === 'collections'}
			<g transform="rotate(-5 160 110)">
				<rect x="78" y="58" width="174" height="112" rx="3" fill="none" stroke="var(--dorado)" stroke-width="2" opacity="0.55" />
			</g>
			<g transform="rotate(3 160 110)">
				<rect x="64" y="50" width="184" height="122" rx="3" fill="var(--granate-oscuro)" stroke="var(--dorado)" stroke-width="2" />
				<path d="M82 68v86M230 68v86" stroke="var(--dorado)" stroke-width="1" opacity="0.5" />
				<circle cx="96" cy="76" r="4" fill="var(--crema-clara)" opacity="0.7" />
				<circle cx="216" cy="76" r="4" fill="var(--crema-clara)" opacity="0.7" />
				<circle cx="96" cy="148" r="4" fill="var(--crema-clara)" opacity="0.7" />
				<circle cx="216" cy="148" r="4" fill="var(--crema-clara)" opacity="0.7" />
				<path d="M120 88h72v48h-72z" fill="var(--crema-clara)" opacity="0.12" />
				<path d="M140 112h32M156 96v32" stroke="var(--dorado)" stroke-width="2" opacity="0.8" />
			</g>
			<path d="M44 184h232" stroke="var(--dorado)" stroke-width="2" opacity="0.45" />
		{:else if art === 'series'}
			<path d="M160 42l-44-24M160 42l48-26" stroke="var(--dorado)" stroke-width="3" stroke-linecap="round" />
			<circle cx="112" cy="16" r="5" fill="var(--dorado)" />
			<circle cx="212" cy="13" r="5" fill="var(--dorado)" />
			<rect x="52" y="46" width="216" height="140" rx="14" fill="var(--granate-oscuro)" stroke="var(--dorado)" stroke-width="2" />
			<rect x="70" y="64" width="180" height="94" rx="8" fill="var(--crema-clara)" opacity="0.12" />
			<path d="M92 92h136M92 112h100M92 132h122" stroke="var(--crema-clara)" stroke-width="2" opacity="0.4" />
			<circle cx="84" cy="172" r="3" fill="var(--dorado)" />
			<circle cx="100" cy="172" r="3" fill="var(--dorado)" />
			<path d="M230 158h12" stroke="var(--dorado)" stroke-width="3" stroke-linecap="round" />
			<path d="M268 82c14 14 14 42 0 56M282 68c22 22 22 66 0 88" fill="none" stroke="var(--dorado)" stroke-width="2" opacity="0.55" />
		{:else}
			<rect x="36" y="44" width="248" height="142" rx="10" fill="var(--granate-oscuro)" stroke="var(--dorado)" stroke-width="2" />
			<rect x="52" y="60" width="216" height="110" rx="4" fill="var(--crema-clara)" opacity="0.08" />
			<path d="M148 82l58 33-58 33z" fill="var(--dorado)" opacity="0.9" />
			<path d="M58 36h204" stroke="var(--dorado)" stroke-width="3" opacity="0.65" />
			{#each [74, 112, 150, 188, 226] as x}
				<rect x={x} y="28" width="18" height="16" fill="var(--granate-oscuro)" stroke="var(--dorado)" stroke-width="1.5" />
			{/each}
			<path d="M74 196h172" stroke="var(--dorado)" stroke-width="2" opacity="0.45" />
		{/if}
	</svg>
</div>

<style>
	.category-art {
		position: absolute;
		inset: 0;
		opacity: 0.95;
		pointer-events: none;
	}

	svg {
		width: 100%;
		height: 100%;
		display: block;
	}
</style>
