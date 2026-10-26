declare global {
	const __APP_VERSION__: string;

	namespace App {
		interface Locals {
			sessionToken: string | null;
		}
	}
}

export {};
