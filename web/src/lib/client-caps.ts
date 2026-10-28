export const CAPS_COOKIE = 'cinecat_caps';
export const HEVC_MIME = 'video/mp4; codecs="hvc1.1.6.L120.90"';
export const HEVC10_MIME = 'video/mp4; codecs="hvc1.2.4.L120.90"';
export const HEV1_MIME = 'video/mp4; codecs="hev1.1.6.L120.90"';

function supported(
	isTypeSupported: (type: string) => boolean,
	canPlayType: (type: string) => string,
	type: string
): boolean {
	try {
		if (isTypeSupported(type)) {
			return true;
		}
	} catch {
		// Algunos navegadores no implementan MediaSource.
	}
	try {
		return canPlayType(type) !== '';
	} catch {
		return false;
	}
}

function mseSupported(isTypeSupported: (type: string) => boolean, type: string): boolean {
	try {
		return Boolean(isTypeSupported(type));
	} catch {
		return false;
	}
}

function canPlay(canPlayType: (type: string) => string, type: string): boolean {
	try {
		return canPlayType(type) !== '';
	} catch {
		return false;
	}
}

export function detectClientCapabilities(
	isTypeSupported: (type: string) => boolean,
	canPlayType: (type: string) => string
): string {
	const hevc10 = supported(isTypeSupported, canPlayType, HEVC10_MIME);
	const hevc = hevc10 || supported(isTypeSupported, canPlayType, HEVC_MIME);
	const hev1 = canPlay(canPlayType, HEV1_MIME);
	// El HLS de Cinecat lleva HEVC en MPEG-TS y los iPhone sin MSE no lo
	// admiten, así que estos tokens miran solo MediaSource.
	const hevc10mse = mseSupported(isTypeSupported, HEVC10_MIME);
	const hevcmse = hevc10mse || mseSupported(isTypeSupported, HEVC_MIME);
	const values: string[] = [];
	if (hevc) {
		values.push('hevc');
	}
	if (hevc10) {
		values.push('hevc10');
	}
	if (hevcmse) {
		values.push('hevcmse');
	}
	if (hevc10mse) {
		values.push('hevc10mse');
	}
	if (hev1) {
		values.push('hev1');
	}
	return values.join(',');
}

export function sanitizeClientCapabilities(raw: string | null | undefined): string {
	const tokens = new Set<string>();
	for (const value of String(raw ?? '').split(',')) {
		const token = value.trim().toLowerCase();
		if (
			token === 'hevc' ||
			token === 'hevc10' ||
			token === 'hevcmse' ||
			token === 'hevc10mse' ||
			token === 'hev1'
		) {
			tokens.add(token);
		}
	}
	if (tokens.has('hevc10')) {
		tokens.add('hevc');
	}
	if (tokens.has('hevc10mse')) {
		tokens.add('hevcmse');
	}
	const values: string[] = [];
	if (tokens.has('hevc')) {
		values.push('hevc');
	}
	if (tokens.has('hevc10')) {
		values.push('hevc10');
	}
	if (tokens.has('hevcmse')) {
		values.push('hevcmse');
	}
	if (tokens.has('hevc10mse')) {
		values.push('hevc10mse');
	}
	if (tokens.has('hev1')) {
		values.push('hev1');
	}
	return values.join(',');
}

export function capsFromCookie(cookieHeader: string | null | undefined): string {
	const prefix = `${CAPS_COOKIE}=`;
	for (const part of String(cookieHeader ?? '').split(';')) {
		const item = part.trim();
		if (item.startsWith(prefix)) {
			let value = item.slice(prefix.length);
			try {
				value = decodeURIComponent(value);
			} catch {
				// Una cookie manipulada no debe romper el layout.
			}
			return sanitizeClientCapabilities(value);
		}
	}
	return '';
}

export function capsCookie(value: string): string {
	return `${CAPS_COOKIE}=${sanitizeClientCapabilities(value)}; path=/; max-age=31536000; SameSite=Lax`;
}
