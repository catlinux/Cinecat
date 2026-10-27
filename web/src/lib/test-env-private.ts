// Sustituto de $env/dynamic/private solo para las pruebas con Vitest. Usa
// process.env para que vi.stubEnv y las variables del proceso funcionen igual
// que en adapter-node.
// @ts-expect-error falta la declaración de node:process
import process from 'node:process';

export const env = process.env as Record<string, string | undefined>;
