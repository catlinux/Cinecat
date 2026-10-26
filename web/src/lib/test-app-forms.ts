// Sustituto de $app/forms para las pruebas con Vitest: el render de servidor
// no ejecuta acciones de formulario.
export function enhance() {
	return {};
}
