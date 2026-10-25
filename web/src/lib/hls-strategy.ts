export type HlsStrategy = 'native' | 'hlsjs' | 'unsupported';

// Decide cómo reproducir un remux HLS: HLS nativo (Safari/iOS), hls.js en el
// resto de navegadores compatibles, o error si no hay ninguna opción.
export function chooseHlsStrategy(
	nativeHlsSupported: boolean,
	hlsJsSupported: boolean
): HlsStrategy {
	if (nativeHlsSupported) {
		return 'native';
	}
	if (hlsJsSupported) {
		return 'hlsjs';
	}
	return 'unsupported';
}
