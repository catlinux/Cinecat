export type HlsStrategy = 'native' | 'hlsjs' | 'unsupported';

// Decide cómo reproducir un HLS: hls.js siempre que el navegador tenga MSE, y
// HLS nativo solo si no (iPhone sin MSE). Chrome en Android y en escritorio
// también dicen soportar HLS nativo, pero sus saltos y errores son mucho menos
// fiables que los de hls.js.
export function chooseHlsStrategy(
	nativeHlsSupported: boolean,
	hlsJsSupported: boolean
): HlsStrategy {
	if (hlsJsSupported) {
		return 'hlsjs';
	}
	if (nativeHlsSupported) {
		return 'native';
	}
	return 'unsupported';
}
