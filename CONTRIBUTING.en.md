# How to contribute

[Versión en español](CONTRIBUTING.md)

Cinecat's server and web client are in active development. Suggestions, corrections and comments on the documents are welcome through *issues*. Code *pull requests* are not accepted without prior agreement with the maintainer.

## Project rules

- **Authorship:** the author and maintainer is **Catlinux**. Accepted contributions are integrated without adding other users as authors of the repository.
- **Languages:** all documentation is in **Spanish and English**, with both versions kept in sync. **Commit messages are written in Spanish.**
- **Versions:** [Semantic Versioning](https://semver.org/) `x.y.z`. Each version is recorded in the [CHANGELOG](CHANGELOG.en.md) and tagged as `vX.Y.Z`.
- **Commits:** proposed format `type: description in Spanish`, with English types (`feat`, `fix`, `docs`, `refactor`, `test`, `chore`). Example: `docs: añade la guía de reproducción`.
- **Credits:** if a proposal is based on code or ideas from another project, say so and respect its license. See [licenses and credits](docs/en/licenses-and-credits.md).

## Recommended Git setup for the maintainer

```bash
git config user.name "Catlinux"
git config user.email "<id>+catlinux@users.noreply.github.com"   # GitHub noreply email
```

If any tool adds co-author lines (`Co-Authored-By`) to commits, disable them so that no other users show up in the repository.
