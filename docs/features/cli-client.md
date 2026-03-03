# CLI

Plik ships with a powerful cross-platform CLI client written in Go.

## Installation

### From GitHub Releases

Download the latest client binary for your platform directly from the [GitHub releases page](https://github.com/root-gg/plik/releases):

```bash
# Linux (amd64)
wget https://github.com/root-gg/plik/releases/download/__VERSION__/plik-__VERSION__-linux-amd64
chmod +x plik-__VERSION__-linux-amd64
sudo mv plik-__VERSION__-linux-amd64 /usr/local/bin/plik

# macOS (amd64)
curl -L -o plik https://github.com/root-gg/plik/releases/download/__VERSION__/plik-__VERSION__-darwin-amd64
chmod +x plik
sudo mv plik /usr/local/bin/plik

# Windows (amd64)
# Download plik-__VERSION__-windows-amd64.exe from the release page
```

Available platforms: `linux-amd64`, `linux-386`, `linux-arm`, `linux-arm64`, `darwin-amd64`, `freebsd-amd64`, `freebsd-386`, `openbsd-amd64`, `openbsd-386`, `windows-amd64`, `windows-386`

### From Plik Web UI

Any running Plik instance serves its client binaries through the web interface. Navigate to your Plik server and download the client for your platform.


## Usage

```
plik [options] [FILE] ...
```

```
Options:
  -o, --oneshot             Enable OneShot ( Each file will be deleted on first download )
  -r, --removable           Enable Removable upload ( Each file can be deleted by anyone at any moment )
  -S, --stream              Enable Streaming ( It will block until remote user starts downloading )
  -t, --ttl TTL             Time before expiration (Upload will be removed in m|h|d)
  --extend-ttl              Extend upload expiration date by TTL when accessed
  -n, --name NAME           Set file name when piping from STDIN
  --stdin                   Enable pipe from stdin explicitly when DisableStdin is set in .plikrc
  --server SERVER           Overrides server url
  --token TOKEN             Specify an upload token ( if '-' prompt for value )
  --comments COMMENT        Set comments of the upload ( MarkDown compatible )
  -p                        Protect the upload with login and password ( be prompted )
  --password PASSWD         Protect the upload with "login:password" ( if omitted default login is "plik" )
  -a                        Archive upload using default archive params ( see ~/.plikrc )
  --archive MODE            Archive upload using the specified archive backend : tar|zip
  --compress MODE           [tar] Compression codec : gzip|bzip2|xz|lzip|lzma|lzop|compress|no
  --archive-options OPTIONS [tar|zip] Additional command line options
  -s                        Encrypt upload using the default encryption parameters ( see ~/.plikrc )
  --not-secure              Do not encrypt upload files regardless of the ~/.plikrc configurations
  --secure MODE             Encrypt upload files using the specified crypto backend : openssl|pgp|age (default: age)
  --cipher CIPHER           [openssl] Openssl cipher to use ( see openssl help )
  --passphrase PASSPHRASE   [openssl|age] Passphrase or '-' to be prompted for a passphrase
  --recipient RECIPIENT     [pgp|age] Set recipient ( pgp: name, age: @github_user, ssh://host, URL, ssh key, or age1... )
  --secure-options OPTIONS  [openssl|pgp] Additional command line options
  --insecure                (TLS) Do not verify the server's certificate chain and hostname
  --update                  Update client
  --login                   Authenticate with the Plik server via browser
  --mcp                     Start as MCP (Model Context Protocol) server over stdio
  -j --json                Output upload metadata as JSON (implies --quiet)
  -q --quiet                Enable quiet mode
  -y --yes                  Auto-accept confirmation prompts (non-interactive mode)
  -d --debug                Enable debug mode
  -v --version              Show client version
  -i --info                 Show client and server information
  -h --help                 Show this help
```

### Examples

Upload a file:
```bash
🪂 ➜  plik git:(master) ✗ plik README.md
Upload successfully created at Sat, 21 Feb 2026 09:02:54 CET :
    http://127.0.0.1:8080/#/?id=vDPmPEUqc5oCt31T

README.md :  2.56 KiB / 2.56 KiB [=========================================] 100.00% 719.15 KiB/s 0s

Commands :
curl -s "http:/127.0.0.1:8080/file/vDPmPEUqc5oCt31T/UZzSdZ7zPgfRiTem/README.md" > 'README.md'
```

Create an encrypted archive:
```bash
plik -a -s mydirectory/
```

Upload with expiration:
```bash
plik --ttl 24h document.pdf
```

## Quick Upload with curl

No CLI needed — upload with a single curl command:

```bash
curl --form 'file=@/path/to/file' http://127.0.0.1:8080
```

With authentication token:
```bash
curl --form 'file=@/path/to/file' \
     --header 'X-PlikToken: xxxx-xxx-xxxx-xxxxx-xxxxxxxx' \
     http://127.0.0.1:8080
```

::: tip
The `DownloadDomain` configuration option must be set for quick upload to work properly.
:::

## CLI Authentication

When authentication is enabled on the server, you can authenticate the CLI client using `--login`:

```bash
plik --login
```

This starts a device authorization flow:
1. The CLI displays a **verification code** and opens a URL in your browser
2. In the browser, log in (if needed) and **approve** the CLI login by confirming the code
3. The CLI automatically receives a token and saves it to `~/.plikrc`

::: tip
The token created via `--login` is identical to tokens created in the web UI. It appears in your token list and can be revoked from the web UI at any time.
:::

### First-run experience

When running `plik` for the first time and the server has authentication enabled, the CLI will prompt you to authenticate via browser:
- If authentication is **forced**: you'll be prompted with a default of **Yes**
- If authentication is **enabled**: you'll be prompted with a default of **No**

You can always authenticate later with `plik --login`.

::: tip Non-interactive mode
Use `plik --yes` to auto-accept all confirmation prompts (first-run wizard, updates, HTTP key fetch warnings). This is useful for scripting and CI/CD pipelines.
:::

### Manual token configuration

Alternatively, you can create a token manually in the web UI and add it to your configuration:

```toml
# ~/.plikrc
Token = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
```

Or pass it on the command line:

```bash
plik --token xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx myfile.txt
```

## Configuration (.plikrc)

The client configuration is a TOML file loaded from:
1. `PLIKRC` environment variable
2. `~/.plikrc`
3. `/etc/plik/plikrc`

Key settings:

```toml
Debug = false                   # be more verbose
Quiet = false                   # be less verbose
URL = "https://plik.root.gg"    # URL of the plik server
OneShot = false                 # Set the uploads to be one shot by default  (if available server side)
Removable = false               # Set the uploads to be removable by default (if available server side)
Stream = false                  # Set the uploads to be stream by default    (if available server side)
Secure = false                  # Set the uploads to be encrypted by default
SecureMethod = "age"            # Set the default encryption method (age|openssl|pgp)
Archive = false                 # Set the uploads to be archives by default
ArchiveMethod = "tar"           # Set the default archive method
DownloadBinary = "curl"         # Set the default download command (curl / wget)
Comments = ""                   # Set the default upload comments
Login = ""                      # Set the default upload login (http basic auth)
Password = ""                   # Set the default upload password (http basic auth)
TTL = 0                         # Set the default upload TTL (0 for server default)
ExtendTTL = false               # Set the uploads to extend TTL by default   (if available server side)
AutoUpdate = true               # Enable/Disable auto update mechanism
Token = ""                      # Set the Authentication Token (can be created from the UI)
DisableStdin = false            # Disable STDIN input
Insecure = false                # Disable HTTPS certificate validation

[ArchiveOptions]
  Compress = "gzip"
  Options = ""
  Tar = "/bin/tar"
```

See the [full .plikrc template](https://github.com/root-gg/plik/blob/master/client/.plikrc) for all available options.

## Tips & Tricks

### Screenshot Upload (Linux)

Upload screenshots directly to clipboard (requires `scrot` and `xclip`):

```bash
alias pshot="scrot -s -e 'plik -q \$f | xclip ; xclip -o ; rm \$f'"
```

### Windows "Send to Plik"

Upload files to Plik directly from the Windows Explorer right-click menu. See the [dedicated guide](/guide/windows-send-to) for step-by-step instructions.

## Bash Client (Lightweight Alternative)

A minimal bash client (`plik.sh`) is available for environments where installing a Go binary is not practical. It requires only `bash`, `curl`, and optionally `openssl`.

```bash
# From a running Plik server
curl -o plik.sh https://your-plik-server/clients/bash/plik.sh
chmod +x plik.sh

# Or from GitHub releases
wget https://github.com/root-gg/plik/releases/download/__VERSION__/plik-__VERSION__.sh
```

Run `plik.sh -h` for the full list of supported options. The bash client supports most upload features (oneshot, removable, stream, TTL, comments, basic auth, encryption) but does not support STDIN piping, JSON output, auto-update, browser login, or age/PGP encryption.
