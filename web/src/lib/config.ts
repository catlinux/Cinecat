import { env } from '$env/dynamic/public';

// URL base del servidor Cinecat (API REST). Se configura con PUBLIC_API_URL
// (por ejemplo en web/.env); nunca se escribe directamente en el código.
export const apiBaseUrl = env.PUBLIC_API_URL ?? '';
