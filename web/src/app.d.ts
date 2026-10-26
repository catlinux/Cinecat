import type { PublicUI } from '$lib/settings';

declare global {
	const __APP_VERSION__: string;

	namespace App {
		interface Locals {
			sessionToken: string | null;
			publicUI?: PublicUI;
		}
	}
}

export {};
