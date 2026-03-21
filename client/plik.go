package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/docopt/docopt-go"

	"github.com/root-gg/plik/plik"
	"github.com/root-gg/plik/server/common"
)

// Main
func main() {

	// Usage /!\ INDENT THIS WITH SPACES NOT TABS /!\
	usage := `plik

Usage:
  plik [options] [FILE] ...

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
  -P, --profile PROFILE     Use a named profile from ~/.plikrc (see Profiles section)
  --insecure                (TLS) Do not verify the server's certificate chain and hostname
  --update                  Update client
  --login                   Authenticate CLI with the server (opens browser)
  --mcp                     Start as MCP (Model Context Protocol) server over stdio
  -j, --json                Output upload metadata as JSON (implies --quiet)
  -q --quiet                Enable quiet mode
  -y --yes                  Auto-accept confirmation prompts (non-interactive mode)
  -d --debug                Enable debug mode
  -v --version              Show client version
  -i --info                 Show client and server information
  -h --help                 Show this help
`
	// Parse command line arguments
	arguments, _ := docopt.ParseDoc(usage)

	if arguments["--version"].(bool) {
		fmt.Printf("Plik client %s\n", common.GetBuildInfo())
		os.Exit(0)
	}

	// Load config
	config, err := LoadConfig(arguments)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to load configuration : %s\n", err)
		os.Exit(1)
	}

	// Load arguments
	err = config.UnmarshalArgs(arguments)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

	// MCP server mode
	if arguments["--mcp"].(bool) {
		err = RunMCPServer(config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "MCP server error: %s\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	cli := NewPlikCLI(config, arguments)

	client := plik.NewClient(config.URL)
	client.Debug = config.Debug
	client.ClientName = "plik_cli"

	// Insecure TLS mode
	if config.Insecure || arguments["--insecure"].(bool) {
		client.Insecure()
	}

	// Display info
	if arguments["--info"].(bool) {
		err = cli.info(client)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Update
	updateFlag := arguments["--update"].(bool)
	err = cli.update(client, updateFlag)
	if err == nil {
		if updateFlag {
			os.Exit(0)
		}
	} else {
		fmt.Fprintf(os.Stderr, "Unable to update Plik client : \n")
		fmt.Fprintf(os.Stderr, "%s\n", err)
		if updateFlag {
			os.Exit(1)
		}
	}

	// Login
	if arguments["--login"].(bool) {
		if arguments["--server"] != nil && arguments["--server"].(string) != "" {
			fmt.Fprintf(os.Stderr, "Cannot use --login with --server: the login flow saves the token to ~/.plikrc and must use the server URL configured there.\n")
			os.Exit(1)
		}
		// --login requires a single profile: it saves a token to one profile section
		if _, err := config.SingleProfile(); err != nil {
			fmt.Fprintf(os.Stderr, "Cannot use --login with multiple profiles: %s\n", err)
			os.Exit(1)
		}
		if len(config.ActiveProfiles) == 1 {
			fmt.Fprintf(os.Stderr, "Authenticating profile %q (%s)...\n", config.ActiveProfiles[0], config.URL)
		}
		err = login(config, client)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Login failed: %s\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Detect STDIN type
	// --> If from pipe : ok, doing nothing
	// --> If not from pipe, and no files in arguments : printing help
	fi, _ := os.Stdin.Stat()

	if runtime.GOOS != "windows" {
		if (fi.Mode()&os.ModeCharDevice) != 0 && len(arguments["FILE"].([]string)) == 0 {
			fmt.Println(usage)
			os.Exit(1)
		}
	} else {
		if len(arguments["FILE"].([]string)) == 0 {
			fmt.Println(usage)
			os.Exit(1)
		}
	}

	// Run the main upload flow
	err = cli.Run(client)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
