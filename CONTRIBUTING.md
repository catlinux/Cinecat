# Cómo contribuir

[English version](CONTRIBUTING.en.md)

Cinecat está en **fase de diseño**: de momento solo hay documentación. Las sugerencias, correcciones y comentarios sobre los documentos son bienvenidos mediante *issues*. Aún no se aceptan *pull requests* de código.

## Normas del proyecto

- **Autoría:** el autor y mantenedor es **Catlinux**. Las contribuciones aceptadas se integran sin añadir otros usuarios como autores del repositorio.
- **Idiomas:** toda la documentación va en **castellano e inglés**, con ambas versiones sincronizadas. Los mensajes de **commit van en castellano**.
- **Versiones:** [Versionado Semántico](https://semver.org/lang/es/) `x.y.z`. Cada versión se anota en el [CHANGELOG](CHANGELOG.md) y se etiqueta como `vX.Y.Z`.
- **Commits:** propuesta de formato `tipo: descripción en castellano`, con tipos en inglés (`feat`, `fix`, `docs`, `refactor`, `test`, `chore`). Ejemplo: `docs: añade la guía de reproducción`.
- **Créditos:** si una propuesta se basa en código o ideas de otro proyecto, hay que indicarlo y respetar su licencia. Ver [licencias y créditos](docs/es/licencias-y-creditos.md).

## Configuración de Git recomendada para el mantenedor

```bash
git config user.name "Catlinux"
git config user.email "<id>+catlinux@users.noreply.github.com"   # correo noreply de GitHub
```

Si se usan herramientas que añaden líneas de coautoría (`Co-Authored-By`) a los commits, desactívalas para que no aparezcan otros usuarios en el repositorio.
