# 1password-direnv-tool

<!-- BADGIE TIME -->



<!-- END BADGIE TIME -->

`op-direnv` loads environment variables from a 1Password or Proton Pass item and prints shell-safe `export` statements for use with `direnv`.

## Why this tool

- Keep secrets in 1Password or Proton Pass instead of `.env` files
- Generate exports at shell load time via `direnv`
- Support secret transforms through field modifiers
- Cache lookups for faster repeated loads
- Support encrypted item references (`encv1://...`) on macOS

## How it works

`op-direnv` reads a 1Password (`op://`) or Proton Pass (`pass://`) item reference, fetches fields from a target section (default: `Environment`), then writes:

```sh
export NAME='value'
```

You typically evaluate this output from `.envrc`.

## Requirements

- Go `1.26+` (for building from source)
- A configured 1Password desktop app session (for `op://` references) or the Proton Pass CLI (`pass-cli`, for `pass://` references)
- `direnv` installed and enabled in your shell

## Installation

### Option 1: Install with Go

```sh
go install github.com/na4ma4/1password-direnv-tool/cmd/op-direnv@latest
```

### Option 2: Build locally

```sh
git clone https://github.com/na4ma4/1password-direnv-tool.git
cd 1password-direnv-tool
go build -o op-direnv ./cmd/op-direnv
```

## Quick start with direnv

1. In 1Password (or Proton Pass), create (or reuse) an item with a section named `Environment`.
2. Add fields in that section where each field title is an env var name.
3. In your project, create `.envrc`:

```sh
export OP_ITEM_UUID="op://Engineering/my-service-secrets"
eval "$(op-direnv)"
```

4. Allow direnv:

```sh
direnv allow
```

You can also pass the item reference directly:

```sh
eval "$(op-direnv --item op://Engineering/my-service-secrets)"
```

For Proton Pass items, use a `pass://` reference instead:

```sh
eval "$(op-direnv --item pass://MyVault/my-service-secrets)"
```

## Item reference format

Supported item reference formats:

```text
op://vault-name-or-id/item-name-or-id
pass://vault-name-or-id/item-name-or-id
```

For `op://`, both vault and item are resolved by ID or case-insensitive title.
For `pass://`, the vault is a Proton Pass share and the item is resolved by ID or case-insensitive title.

## Field mapping and modifiers

Within the selected section (`--section`, default `Environment`), field titles are parsed as:

```text
VAR_NAME[:modifier1[:modifier2...]]
```

Examples:

- `DATABASE_URL:1password`
- `TLS_PUBLIC_KEY:1password,b64`
- `JWT_PUBLIC_KEY:b64`
- `CONFIG_JSON:optmpl`

Only valid shell variable names are exported (`^[a-zA-Z_][a-zA-Z0-9_]*$`). Invalid names are skipped.

### Supported modifiers

- `1password` / `op`
	- Treat the field value as an `op://...` secret reference and resolve it.
- `protonpass` / `pass`
	- Treat the field value as a `pass://...` secret reference and resolve it via the Proton Pass CLI.
- `b64` / `base64`
	- Base64-encode the field value.
- `b32` / `base32`
	- Base32-encode the field value.
- `op-tmpl` / `optmpl` / `1password-template`
	- Resolve `{{ op://vault/item/field }}` templates embedded inside a larger string.
- `pass-tmpl` / `passtmpl` / `protonpass-template`
	- Resolve `{{ pass://vault/item/field }}` templates embedded inside a larger string.

Modifiers are applied left-to-right.

### Cross-provider references

All modifiers are always available regardless of the item's primary provider. This means:

- A **1Password** field value can reference a `pass://...` secret using the `:protonpass` or `:passtmpl` modifier.
- A **Proton Pass** field value can reference an `op://...` secret using the `:1password` or `:optmpl` modifier.

Providers are lazily initialized — the secondary provider is only created when its resolver is first invoked, so you don't pay for what you don't use.

## Encrypted item references

On macOS, references can be encoded using a key stored in Keychain.

Encode a reference:

```sh
op-direnv encode op://Engineering/my-service-secrets
# => encv1://...
```

Use it in `.envrc`:

```sh
export OP_ITEM_UUID="encv1://..."
eval "$(op-direnv)"
```

Export/import key (for migration/backups):

```sh
op-direnv export-key
# => encv1-key://...

op-direnv import-key
# prompts: Enter the encryption key:
```

Notes:

- On non-macOS platforms, codec behavior is no-op.
- `import-key` is only meaningful with the macOS Keychain codec.

## CLI usage

```text
op-direnv [flags]
op-direnv [command]

Commands:
	clean       Clean cached values
	encode      Encode a 1Password item reference
	export-key  Export item-reference encryption key
	import-key  Import item-reference encryption key
	k8s         Run as a Kubernetes exec-plugin for kubectl authentication
```

### Main flags

- `-a, --1password-account-name`
	- 1Password desktop account name (default fallback: `my`)
- `-i, --item`
	- 1Password item reference
- `-s, --section`
	- Item section to read (default `Environment`)
- `-c, --cache`
	- Enable/disable cache (default `true`)
- `--cache-age`
	- Cache TTL window (default `8h`)
- `-t, --timeout`
	- Global operation timeout (default `2m`)
- `-d, --debug`
	- Debug logging

## Kubernetes exec-plugin

`op-direnv` can serve as a Kubernetes client credential provider (exec-plugin) for `kubectl` authentication. It reads client certificate, client key, and CA data from a 1Password or Proton Pass item and outputs an `ExecCredential`.

### Configure kubectl

Add the following to your kubeconfig:

```yaml
users:
- name: my-cluster-user
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1
      command: op-direnv
      args:
        - k8s
      env:
        - name: OP_ITEM_UUID
          value: op://Engineering/my-k8s-credential
```

### Item structure

The `k8s` command reads a Kubernetes credential item. The item should contain:

| Field name                | Purpose                    |
| ------------------------- | -------------------------- |
| `client-certificate`      | TLS client certificate     |
| `client-key`              | TLS client key             |
| `server`                  | (optional) API server URL  |
| `certificate-authority-data` | (optional) CA cert     |

Fields are resolved by short reference (case-insensitive). If `client-certificate` or `client-key` are missing the command will return an error; `server` and `certificate-authority-data` are optional.

The command works with both `op://` and `pass://` item references.

## Configuration

Config file name: `op-direnv.toml`

Search paths:

- `${HOME}/.config`
- `/etc/op-direnv`

Example `${HOME}/.config/op-direnv.toml`:

```toml
[cache]
enabled = true
path = "${HOME}/.cache/op-direnv"
age = "8h"
```

### Environment variables

- `OP_ITEM_UUID`
- `OP_SECTION`
- `OP_CACHE_ENABLED`
- `OP_CACHE_AGE`
- `OP_DIRENV_CACHE_PATH`
- `OP_TIMEOUT`
- `1PASSWORD_ACCOUNT_NAME`
- `DEBUG`

### Item reference resolution order

The first available source is used:

1. `OP_ITEM_UUID`
2. Git config `1password.envrc-item`
3. Viper `item` (for example: `--item` or config file)

Set Git config per repo:

```sh
git config 1password.envrc-item op://Engineering/my-service-secrets
```

## Cache behavior

- Cache path defaults to `${HOME}/.cache/op-direnv`
- Disk cache files are mode `0600`
- Cache entries are expired opportunistically on run based on `cache-age`
- `op-direnv clean` clears all cached entries
- Cached values are wrapped by codec-based encryption/decryption in the cache layer

## Development

Run tests:

```sh
go test ./...
```

Build:

```sh
go build ./cmd/op-direnv
```

The repository also includes Mage tasks under `magefiles/`.

## Troubleshooting

- **No variables exported**
	- Confirm the item has a section matching `--section` (default `Environment`).
	- Confirm field names are valid env variable names.
- **Vault/item not found**
	- Verify the `op://vault/item` reference and account context.
- **Auth/client errors**
	- Ensure the 1Password desktop app is running and signed in.
- **Unexpected values**
	- Check modifier order in field titles.
- **Cross-provider references not resolving**
	- Confirm the referenced provider's CLI/tool is available (1Password desktop app for `op://`, `pass-cli` for `pass://`).
- **Proton Pass not found**
	- Ensure `pass-cli` is installed and in your `PATH` (on macOS Homebrew: `brew install proton-pass-cli`).

## License

Licensed under MIT. See `LICENSE`.
