# profile-cards

Small SVG widgets for my GitHub profile README, served from my own ASDL Hub instead of third-party card services. Plain Go, no dependencies.

| Endpoint | What |
|---|---|
| `/typing.svg?lines=one\|two&width=435` | Types each line out, holds, erases, next. `\|` separates lines. |
| `/stats.svg` | The stats card: stars, commits and PRs, issues, contributed-to, and the rank circle. |
| `/langs.svg` | Most used languages by bytes of code across my own (non-fork) repos. |
| `/streak.svg` | Total contributions, current streak and longest streak. |
| `/health` | `ok` |

The stats, languages and streak cards look like github-readme-stats and github-readme-streak-stats (same layout and grade formula), in the dark theme with the amber accent; the typing line follows the viewer's light or dark setting. GitHub answers are cached for an hour, and the last good answer is served if GitHub is unreachable.

Environment: `GITHUB_TOKEN` is required for `/stats.svg` and `/langs.svg` (GitHub's GraphQL API): a classic personal access token with **no scopes**, which can read public data only. `/streak.svg` reads the public contributions page and needs no token. Optional: `GITHUB_USER`, `PORT`.
