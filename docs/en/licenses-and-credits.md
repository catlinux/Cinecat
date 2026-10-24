# Licenses and credits

[Español](../es/licencias-y-creditos.md) · [Back to README](../../README.en.md)

> This is working documentation, **not legal advice**. A license review is advisable before the first public release.

## Status

- **Project license: GPL-3.0**, with an additional section 7 term requiring Catlinux's authorship notices to be preserved. See [LICENSE](../../LICENSE).
- As of this decision **no code has been copied** from other projects. Only literal code from GPL-3.0-or-later or MPL-2.0 projects (compatible) may be copied; GPL-2.0-only code (such as Jellyfin and its official clients, unless a specific file states "or later") may only be used as **inspiration**, rewritten from scratch.

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

## Decision (2026-10-24)

**GPL-3.0, with the additional section 7 term** requiring Catlinux's authorship notices to be preserved in any copy or modified version (About screens, credits, legal notices). Full text in [LICENSE](../../LICENSE).

Licenses verified for the candidate projects (checked on their official repository, not from memory):

| Project | Confirmed license | What it means for Cinecat |
|---|---|---|
| Jellyfin (server) | GPL-2.0 | Inspiration only; no literal copying without checking file-by-file for an "or later" mark |
| Jellyfin Android TV | GPL-2.0 | Inspiration only |
| Wholphin | GPL-2.0 | Inspiration only |
| Findroid | GPL-3.0 | Literal copying allowed, with credits and copyright notices |
| Intro Skipper | GPL-3.0 | Literal copying allowed, with credits and copyright notices |
| Streamyfin | MPL-2.0 | Compatible; literal copying allowed respecting MPL-2.0 in the affected files |
| Media3/ExoPlayer, hls.js, Shaka Player | Apache-2.0 | Compatible; literal copying allowed with their notices |
| Vidstack | MIT | Compatible; literal copying allowed with their notices |

It is applied with a root `LICENSE` file, SPDX headers (`SPDX-License-Identifier: GPL-3.0-or-later`) in our own source files, and the license field in package metadata where applicable.

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
