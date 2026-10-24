# profile-cards

Small SVG widgets for my GitHub profile README, served from my own ASDL Hub instead of third-party card services. Plain Go, no dependencies.

| Endpoint | What |
|---|---|
| `/typing.svg?lines=one\|two&width=435` | Types each line out, holds, erases, next. `\|` separates lines. |
| `/stats.svg` | Public repos, stars, followers, join year, latest release of the Hub. |
| `/langs.svg` | Top languages by bytes of code across my own (non-fork) public repos. |
| `/health` | `ok` |

Colours follow the viewer's light or dark setting. GitHub answers are cached for an hour, and the last good answer is served if GitHub is unreachable.

Environment (all optional): `GITHUB_USER`, `HUB_REPO`, `PORT`, `GITHUB_TOKEN` (only raises the API rate limit).
