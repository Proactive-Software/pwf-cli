# pwf

CLI for ProWorkflow — add & select items, move stages, open in browser, and copy uniqueTokens without leaving the terminal.

## Commands

```
pwf                      # list your active items (default)
pwf list [--all]         # list active items; --all shows entire account
pwf start [--open]       # fuzzy picker → set global active item
pwf active               # show current active item
pwf clear                # unset active item
pwf stage <name>         # move active item to a workstage (fuzzy matched)
pwf open [id] [--project]  # open active item in browser; --project opens its project
pwf token [id]           # print uniqueToken + copy to clipboard
pwf new "<title>"        # create item in active item's project & phase
pwf desc                 # show active item's description (rendered)
pwf sync                 # pull latest workstages from API
pwf init                 # one-time setup (OAuth login)
```

## Setup

```sh
pwf init
```

Prompts for:
- **API base URL** — e.g. `https://api.proworkflow.com/api/v4`
- **App base URL** — e.g. `https://app.proworkflow.com`
- **Subdomain** — e.g. `your_subdomain`
- **OAuth client ID** — registered in common.OAuthClient
- **OAuth client secret** — registered in common.OAuthClient

Then opens a browser for login. After authenticating, tokens are stored in the OS keyring automatically. No manual token handling required.

Workstages are synced from the API automatically on init. Re-run `pwf sync` whenever stages change.

## Config files

```
~/.config/pwf/config.toml        # api_base, app_base, subdomain, oauth_client_id, oauth_client_secret
~/.config/pwf/workstages.toml    # id → name map (auto-managed, do not hand-edit)
~/.config/pwf/active.json        # current active item (id, title, token, project info)
OS keyring                        # OAuth tokens JSON (service: pwf-cli, account: oauth-tokens)
```

`config.toml` is safe to edit manually:

```toml
api_base = "https://api.proworkflow.com/api/v4"
app_base = "https://app.proworkflow.com"
subdomain = "your_account"
oauth_client_id = "your-client-id"
oauth_client_secret = "your-client-secret"
```

## Auth

Uses OAuth 2.0 authorization code flow via `https://identity.proworkflow.com`.

`pwf init` opens a browser, you log in, and the CLI receives an access token + refresh token via a local callback server on `localhost:9876`. Access tokens expire after 5 minutes — the CLI refreshes them automatically on each command. Refresh tokens last 14 days.

OAuth client must be registered with `redirect_uri=http://localhost:9876/callback` (exact match).

## Active item

Global — not per-repo. Stored in `~/.config/pwf/active.json`. All commands (`stage`, `open`, `token`, `new`) operate on whichever item is active.

```sh
pwf start        # pick from your items
pwf active       # check what's set
pwf clear        # unset it
```

## Workstages

`pwf stage` fuzzy-matches by name — prefix or substring is enough:

```sh
pwf stage done
pwf stage "in progress"
pwf stage test     # matches "Needs Testing" etc.
```

If a stage name is ambiguous (duplicate names with different IDs), the first match wins.

## Browser URLs

`pwf open` opens:
```
{app_base}/{subdomain}/?fuseaction=trackedprojects&fusesubaction=details&Jobs_currentJobID={projectid}&item={itemid}
```

`pwf open --project` drops the `&item=` part.

## Shell completion

Zsh completions installed at `/opt/homebrew/share/zsh/site-functions/_pwf`.

Subcommands, flags, and workstage names (for `pwf stage <TAB>`) all complete.

To regenerate after updating the binary:

```sh
pwf completion zsh > /opt/homebrew/share/zsh/site-functions/_pwf
```

## Installation

### macOS (Homebrew)

```sh
brew tap Proactive-Software/pwf-cli
brew install pwf
```

### Manual (macOS / Linux / Windows)

Download the latest archive for your platform from the [releases page](https://github.com/Proactive-Software/pwf-cli/releases), then:

```sh
# macOS / Linux
tar -xzf pwf_darwin_arm64.tar.gz
chmod +x pwf
mv pwf /usr/local/bin/pwf
```

Run `pwf init` after installing.

## Releasing

Tag the commit and push — GitHub Actions handles the rest:

```sh
git tag v1.2.3
git push origin v1.2.3
```

This triggers the release workflow which:
- Builds binaries for macOS (arm64, amd64), Linux (arm64, amd64), Windows (amd64)
- Creates a GitHub release with archives + checksums attached
- Updates the Homebrew formula in `Proactive-Software/homebrew-pwf-cli`

## Development

```sh
cd ~/repos/pwf-cli
go build -o pwf ./cmd/pwf    # local binary
go install ./cmd/pwf         # install to ~/go/bin/pwf (on PATH)
```

### Project layout

```
cmd/pwf/main.go
internal/
  api/          # typed HTTP client for ~6 endpoints
  auth/         # OAuth 2.0 flow, token refresh, keyring marshalling
  config/       # config.toml + workstages.toml load/save
  state/        # active.json read/write
  keyring/      # OAuth tokens via OS keyring
  picker/       # Bubble Tea fuzzy picker
  commands/     # cobra command implementations
```

### API notes

- Auth header: `Authorization: Bearer <token>` (JWT access token from identity server)
- Items endpoint: `/projectitems` (not `/items`)
- Workstages: `GET /settings/workstages/item`
- Pagination required on list endpoints: must pass both `pagesize` and `pagenumber`
- Item detail (`/projectitems/{id}`) includes `uniquetoken` and `itemcollectionid`
- No `activeworkstagename` in any response — names resolved locally from workstages.toml
