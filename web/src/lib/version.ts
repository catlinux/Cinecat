// La versión la inyecta Vite desde el VERSION de la raíz al compilar.
export const webVersion: string = __APP_VERSION__;

// Solo avisamos si el servidor respondió y su versión no coincide.
export function versionMismatch(serverVersion: string | null, localVersion: string): boolean {
	return serverVersion !== null && serverVersion !== localVersion;
}
