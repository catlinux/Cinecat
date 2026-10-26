<script lang="ts">
	import type { Setting } from '$lib/api';
	import {
		formatSettingDefault,
		formatSettingInputValue,
		settingControlId,
		settingDescriptionId,
		settingErrorId,
		type SettingsFormState
	} from '$lib/settings';

	let {
		settings,
		form = null
	}: {
		settings: Setting[];
		form?: SettingsFormState | null;
	} = $props();

	const errors = $derived(form?.fields ?? {});
	const enteredValues = $derived(form?.values ?? {});

	function valueFor(setting: Setting): unknown {
		if (Object.prototype.hasOwnProperty.call(enteredValues, setting.key)) {
			return enteredValues[setting.key];
		}
		return setting.value;
	}

	function textValue(setting: Setting): string {
		const value = formatSettingInputValue(setting, valueFor(setting));
		return typeof value === 'boolean' ? '' : value;
	}

	function describedBy(setting: Setting): string {
		const ids = [settingDescriptionId(setting.key)];
		if (errors[setting.key]) {
			ids.push(settingErrorId(setting.key));
		}
		return ids.join(' ');
	}
</script>

<form method="POST" action="?/save" class="settings-form">
	<input type="hidden" name="__settings_form" value="1" />
	{#if form?.message}
		<p class={form.success ? 'success' : 'error'} role={form.success ? 'status' : 'alert'}>
			{form.message}
		</p>
	{/if}

	{#each settings as setting (setting.key)}
		<div class="setting-row" class:setting-row--error={Boolean(errors[setting.key])}>
			{#if setting.type === 'bool'}
				<label class="setting-check" for={settingControlId(setting.key)}>
					<input
						id={settingControlId(setting.key)}
						type="checkbox"
						name={setting.key}
						value="1"
						checked={Boolean(valueFor(setting))}
						aria-describedby={describedBy(setting)}
					/>
					<span class="setting-label">{setting.label}</span>
				</label>
			{:else}
				<label class="setting-label" for={settingControlId(setting.key)}>{setting.label}</label>
				{#if setting.type === 'enum'}
					<select
						id={settingControlId(setting.key)}
						name={setting.key}
						value={textValue(setting)}
						aria-describedby={describedBy(setting)}
					>
						{#each setting.options as option (option.value)}
							<option value={option.value}>{option.label}</option>
						{/each}
					</select>
				{:else if setting.type === 'int' || setting.type === 'float'}
					<input
						id={settingControlId(setting.key)}
						type="number"
						name={setting.key}
						value={textValue(setting)}
						min={setting.min !== 0 ? setting.min : undefined}
						max={setting.max !== 0 ? setting.max : undefined}
						step={setting.type === 'int' ? '1' : '0.01'}
						aria-describedby={describedBy(setting)}
					/>
				{:else}
					<input
						id={settingControlId(setting.key)}
						type="text"
						name={setting.key}
						value={textValue(setting)}
						minlength={setting.type === 'string' && setting.min > 0 ? setting.min : undefined}
						maxlength={setting.type === 'string' && setting.max > 0 ? setting.max : undefined}
						placeholder={setting.type === 'duration' ? '6h, 30m' : undefined}
						aria-describedby={describedBy(setting)}
					/>
				{/if}
			{/if}

			<p class="setting-description" id={settingDescriptionId(setting.key)}>
				{setting.description}
				{#if setting.type === 'duration'}
					<span class="setting-help">P. ej. 6h, 30m; 0 lo desactiva.</span>
				{/if}
			</p>
			<p class="setting-default">Por defecto: {formatSettingDefault(setting)}</p>

			{#if errors[setting.key]}
				<p class="setting-error" id={settingErrorId(setting.key)} role="alert">
					{errors[setting.key]}
				</p>
			{/if}

			{#if setting.modified}
				<button
					class="button-danger setting-reset"
					type="submit"
					formaction="?/reset"
					formnovalidate
					name="key"
					value={setting.key}
					aria-label={`Restablecer ${setting.label}`}
				>
					Restablecer
				</button>
			{/if}
		</div>
	{/each}

	<div class="settings-actions">
		<button class="button" type="submit">Guardar cambios</button>
	</div>
</form>
