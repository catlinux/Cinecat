# Licenses and credits

[Español](../es/licencias-y-creditos.md) · [Back to README](../../README.en.md)

> This is working documentation, **not legal advice**. A license review is advisable before the first public release.

## Status

- **Project license: to be decided.** It is chosen **before** the first line of code is published.
- As of 0.0.1 **no code has been copied** from other projects.

## What we want

1. To **reuse everything that is legal** from other free projects, always citing the origin.
2. Our code to be **shareable, modifiable and studiable**, **always preserving authorship (Catlinux) and credits**, which cannot be removed.
3. A license that does not oblige us to anything we cannot comply with.

## What reusing GPL code implies

Several reference projects (Jellyfin and its official clients, Wholphin, Findroid) use **GPL** licenses. If GPL code is copied, the combined result must also be under GPL. Practical consequences:

- Cinecat's license will be **GPL or GPL-compatible**.
- **GPL-2.0 and GPL-3.0 cannot always be mixed**: a file that is "2.0 only" cannot be combined with "3.0 only" code. The license tag of **each file** (for example the SPDX line) must be checked before copying.
- Code under **permissive or weak-copyleft** licenses (Apache-2.0, MIT, MPL-2.0) keeps its notices and conditions, and is compatible with a GPL license.
- Whatever is not copied but only **taken as an idea** is rewritten from scratch and cited as inspiration.

## Credits that cannot be removed

- Copyright and license notices of reused projects **are preserved** in every file and in [CREDITS.md](../../CREDITS.md).
- For our own code, GPLv3 allows adding, as an additional condition, **preservation of author attributions** in the legal notices that applications display. This is one of the options to weigh when choosing the license.
- The apps will have an **"About"** screen with author, version, license and third-party credits (including TMDB).

## License options to weigh (Phase 0)

| Option | Advantages | To keep in mind |
|---|---|---|
| GPL-3.0 (with additional attributions) | Strong copyleft; allows a condition to preserve authorship. | Not mixable with "GPL-2.0 only" code. |
| GPL-2.0 | Compatible with Jellyfin code if it is "2.0 only". | Older; without GPLv3's patent clauses. |
| AGPL-3.0 | Closes the network-service gap (useful for a server). | Same mixing limits as GPL-3.0. |

The decision is made after listing **which specific code** will be reused. It is applied with `LICENSE`, SPDX headers in files and the license field in package metadata.

## Rules for reusing third-party code

1. Verify the license **in the exact version** being copied (not on the project's website).
2. Record in `CREDITS.md` and `THIRD_PARTY_NOTICES.md`: project, URL, version or commit, license, affected files.
3. Preserve the original copyright notices in the files.
4. Do not fork or import the history of other repositories: code is brought in through new commits of this repository (Cinecat's history shows only Catlinux).
5. What cannot be legally reused is rewritten (without copying its structure).

## Data sources and trademarks

- **TMDB**, **TheTVDB**, **Fanart.tv**: use subject to their terms. The required attribution is displayed.
- Third-party trademarks and logos belong to their owners.
- **FFmpeg** is used as an external tool and is not distributed in the repository.

## Project list

See [CREDITS.md](../../CREDITS.md).
