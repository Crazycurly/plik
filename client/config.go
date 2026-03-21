package main

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/docopt/docopt-go"
	"github.com/mitchellh/go-homedir"

	"github.com/root-gg/plik/plik"
	"github.com/root-gg/plik/server/common"
)

// CliConfig object
type CliConfig struct {
	Debug          bool
	Quiet          bool
	JSON           bool
	Yes            bool
	URL            string
	OneShot        bool
	Removable      bool
	Stream         bool
	Secure         bool
	SecureMethod   string
	SecureOptions  map[string]any
	Archive        bool
	ArchiveMethod  string
	ArchiveOptions map[string]any
	DownloadBinary string
	Comments       string
	Login          string
	Password       string
	TTL            int
	ExtendTTL      bool
	AutoUpdate     bool
	Token          string
	DisableStdin   bool
	Insecure       bool
	ConfigPath     string `toml:"-" profile:"-"`

	ActiveProfile     string   `toml:"-" profile:"-"`
	AvailableProfiles []string `toml:"-" profile:"-"`

	filePaths        []string
	filenameOverride string
}

// PlikrcFile is the on-disk representation of .plikrc.
// It embeds CliConfig for the top-level (default) fields and adds optional
// named profiles and a default profile selector.
type PlikrcFile struct {
	CliConfig
	Profiles       map[string]CliConfig `toml:"Profiles,omitempty"`
	DefaultProfile string               `toml:"DefaultProfile,omitempty"`
}

// NewUploadConfig construct a new configuration with default values
func NewUploadConfig() (config *CliConfig) {
	config = new(CliConfig)
	config.URL = "http://127.0.0.1:8080"
	config.ArchiveMethod = "tar"
	config.ArchiveOptions = make(map[string]any)
	config.ArchiveOptions["Tar"] = "/bin/tar"
	config.ArchiveOptions["Compress"] = "gzip"
	config.ArchiveOptions["Options"] = ""
	config.SecureMethod = "age"
	config.SecureOptions = make(map[string]any)
	config.DownloadBinary = "curl"
	return
}

// LoadConfigFromFile loads a TOML config file with optional profile selection.
// If profileName is empty, the profile is resolved from PLIK_PROFILE env var
// or the DefaultProfile field in the config file.
func LoadConfigFromFile(path string, profileName string) (*CliConfig, error) {
	var plikrc PlikrcFile
	plikrc.CliConfig = *NewUploadConfig()

	md, err := toml.DecodeFile(path, &plikrc)
	if err != nil {
		return nil, fmt.Errorf("Failed to deserialize ~/.plikrc : %s", err)
	}

	config := &plikrc.CliConfig

	// Populate available profile names (sorted for stable output)
	for name := range plikrc.Profiles {
		config.AvailableProfiles = append(config.AvailableProfiles, name)
	}
	sort.Strings(config.AvailableProfiles)

	// Resolve profile name: CLI flag > env var > config DefaultProfile
	if profileName == "" {
		profileName = os.Getenv("PLIK_PROFILE")
	}
	if profileName == "" {
		profileName = plikrc.DefaultProfile
	}

	// Apply profile if one is selected
	if profileName != "" {
		profile, ok := plikrc.Profiles[profileName]
		if !ok {
			if len(config.AvailableProfiles) == 0 {
				return nil, fmt.Errorf("Profile %q not found (no profiles defined in config)", profileName)
			}
			return nil, fmt.Errorf("Profile %q not found (available: %s)", profileName, strings.Join(config.AvailableProfiles, ", "))
		}
		mergeProfile(config, &profile, md, profileName)
		config.ActiveProfile = profileName
	}

	// Sanitize URL
	config.URL = strings.TrimSuffix(config.URL, "/")

	config.ConfigPath = path

	return config, nil
}

// mergeProfile overlays explicitly-set profile fields onto the base config.
// It uses TOML metadata to distinguish "field not present" from "field set to zero value".
// Fields tagged `profile:"-"` and unexported fields are skipped.
func mergeProfile(base *CliConfig, profile *CliConfig, md toml.MetaData, profileName string) {
	baseVal := reflect.ValueOf(base).Elem()
	profVal := reflect.ValueOf(profile).Elem()
	baseType := baseVal.Type()

	for i := 0; i < baseType.NumField(); i++ {
		field := baseType.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Skip fields tagged profile:"-"
		if field.Tag.Get("profile") == "-" {
			continue
		}

		if md.IsDefined("Profiles", profileName, field.Name) {
			baseVal.Field(i).Set(profVal.Field(i))
		}
	}
}

// LoadConfig creates a new default configuration and override it with .plikrc file.
// If .plikrc does not exist, ask domain, and create a new one in user HOMEDIR
func LoadConfig(opts docopt.Opts) (config *CliConfig, err error) {
	// Resolve profile name from CLI flag (env var / config default handled in LoadConfigFromFile)
	var profileName string
	if opts["--profile"] != nil && opts["--profile"].(string) != "" {
		profileName = opts["--profile"].(string)
	}

	// Load config file from environment variable
	path := os.Getenv("PLIKRC")
	if path != "" {
		_, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("Plikrc file %s not found", path)
		}
		return LoadConfigFromFile(path, profileName)
	}

	// Detect home dir
	home, err := homedir.Dir()
	if err != nil {
		home = os.Getenv("HOME")
		if home == "" {
			home = "."
		}
	}

	// Load config file from ~/.plikrc
	path = home + "/.plikrc"
	_, err = os.Stat(path)
	if err == nil {
		config, err = LoadConfigFromFile(path, profileName)
		if err == nil {
			return config, nil
		}
	} else {
		// Load global config file from /etc directory
		path = "/etc/plik/plikrc"
		_, err = os.Stat(path)
		if err == nil {
			config, err = LoadConfigFromFile(path, profileName)
			if err == nil {
				return config, nil
			}
		}
	}

	config = NewUploadConfig()

	// Bypass ~/.plikrc file creation if quiet mode, --yes mode, and/or --server flag
	if opts["--quiet"].(bool) || opts["--yes"].(bool) || (opts["--server"] != nil && opts["--server"].(string) != "") {
		return config, nil
	}

	// Config file not found. Create one.
	path = home + "/.plikrc"

	// Ask for domain
	var domain string
	fmt.Println("Please enter your plik domain [default:http://127.0.0.1:8080] : ")
	_, err = fmt.Scanf("%s", &domain)
	if err == nil {
		domain = strings.TrimRight(domain, "/")
		parsedDomain, err := url.Parse(domain)
		if err == nil {
			if parsedDomain.Scheme == "" {
				parsedDomain.Scheme = "http"
			}
			config.URL = parsedDomain.String()
		}
	}

	// Try to HEAD the site to see if we have a redirection
	client := plik.NewClient(config.URL)
	client.Insecure()
	resp, err := client.HTTPClient.Head(config.URL)
	if err != nil {
		return nil, err
	}

	finalURL := resp.Request.URL.String()
	if finalURL != "" && finalURL != config.URL {
		fmt.Printf("We have been redirected to : %s\n", finalURL)
		fmt.Printf("Replace current url (%s) with the new one ? [Y/n] ", config.URL)

		ok, err := common.AskConfirmation(true)
		if err != nil {
			return nil, fmt.Errorf("Unable to ask for confirmation : %s", err)
		}
		if ok {
			config.URL = strings.TrimSuffix(finalURL, "/")
		}
	}

	// Try to get server config to sync default values
	serverConfig, err := client.GetServerConfig()
	if err != nil {
		fmt.Printf("Unable to get server configuration : %s", err)
	} else {
		config.OneShot = common.IsFeatureDefault(serverConfig.FeatureOneShot)
		config.Removable = common.IsFeatureDefault(serverConfig.FeatureRemovable)
		config.Stream = common.IsFeatureDefault(serverConfig.FeatureStream)
		config.ExtendTTL = common.IsFeatureDefault(serverConfig.FeatureExtendTTL)

		// Skip interactive login during first-run setup when --login is set,
		// the --login handler in main() will perform the login flow.
		if !opts["--login"].(bool) {
			switch serverConfig.FeatureAuthentication {
			case common.FeatureForced:
				fmt.Printf("\nAuthentication is required on this server.\n")
				fmt.Printf("Would you like to authenticate with your browser? [Y/n] ")
				ok, err := common.AskConfirmation(true)
				if err != nil {
					return nil, fmt.Errorf("Unable to ask for confirmation : %s", err)
				}
				if ok {
					loginClient := plik.NewClient(config.URL)
					loginClient.Insecure()
					err = login(config, loginClient)
					if err != nil {
						fmt.Printf("Login failed: %s\n", err)
						fmt.Printf("You can provide a token manually instead.\n")
						fmt.Printf("Please enter a valid user token : \n")
						var token string
						_, err = fmt.Scanf("%s", &token)
						if err == nil {
							config.Token = token
						}
					}
				} else {
					fmt.Printf("Please enter a valid user token : \n")
					var token string
					_, err = fmt.Scanf("%s", &token)
					if err == nil {
						config.Token = token
					}
				}
			case common.FeatureEnabled:
				fmt.Printf("\nAuthentication is available on this server.\n")
				fmt.Printf("Would you like to authenticate with your browser? [y/N] ")
				ok, err := common.AskConfirmation(false)
				if err != nil {
					return nil, fmt.Errorf("Unable to ask for confirmation : %s", err)
				}
				if ok {
					loginClient := plik.NewClient(config.URL)
					loginClient.Insecure()
					err = login(config, loginClient)
					if err != nil {
						fmt.Printf("Login failed: %s\n", err)
					}
				}
			}
		}
	}

	// Enable client updates ?
	fmt.Println("Do you want to enable client auto update ? [Y/n] ")
	ok, err := common.AskConfirmation(true)
	if err != nil {
		return nil, fmt.Errorf("Unable to ask for confirmation : %s", err)
	}
	if ok {
		config.AutoUpdate = true
	}

	// Encode in TOML (wrap in PlikrcFile for forward compatibility)
	plikrc := &PlikrcFile{CliConfig: *config}
	buf := new(bytes.Buffer)
	if err = toml.NewEncoder(buf).Encode(plikrc); err != nil {
		return nil, fmt.Errorf("Failed to serialize ~/.plikrc : %s", err)
	}

	// Write file
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("Failed to save ~/.plikrc : %s", err)
	}

	_, _ = f.Write(buf.Bytes())
	_ = f.Close()

	fmt.Println("Plik client settings successfully saved to " + path)
	return config, nil
}

// UnmarshalArgs turns command line arguments into upload settings
// Command line arguments override config file settings
func (config *CliConfig) UnmarshalArgs(opts docopt.Opts) (err error) {
	if opts["--debug"].(bool) {
		config.Debug = true
	}
	if opts["--yes"].(bool) {
		config.Yes = true
	}
	if opts["--quiet"].(bool) {
		config.Quiet = true
	}
	if opts["--json"].(bool) {
		config.JSON = true
		config.Quiet = true
	}

	// Plik server url
	if opts["--server"] != nil && opts["--server"].(string) != "" {
		config.URL = opts["--server"].(string)
	}

	// Paths
	if _, ok := opts["FILE"].([]string); ok {
		config.filePaths = opts["FILE"].([]string)
	} else {
		return fmt.Errorf("No files specified")
	}

	for _, path := range config.filePaths {
		// Test if file exists
		fileInfo, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("File %s not found", path)
		}

		// Automatically enable archive mode is at least one file is a directory
		if fileInfo.IsDir() {
			config.Archive = true
		}
	}

	// Override file name if specified
	if opts["--name"] != nil && opts["--name"].(string) != "" {
		config.filenameOverride = opts["--name"].(string)
	}

	// Upload options
	if opts["--oneshot"].(bool) {
		config.OneShot = true
	}
	if opts["--removable"].(bool) {
		config.Removable = true
	}

	if opts["--stream"].(bool) {
		config.Stream = true
	}

	if opts["--comments"] != nil && opts["--comments"].(string) != "" {
		config.Comments = opts["--comments"].(string)
	}

	// Configure upload expire date
	if opts["--ttl"] != nil && opts["--ttl"].(string) != "" {
		ttlStr := opts["--ttl"].(string)
		mul := 1
		if string(ttlStr[len(ttlStr)-1]) == "m" {
			mul = 60
		} else if string(ttlStr[len(ttlStr)-1]) == "h" {
			mul = 3600
		} else if string(ttlStr[len(ttlStr)-1]) == "d" {
			mul = 86400
		}
		if mul != 1 {
			ttlStr = ttlStr[:len(ttlStr)-1]
		}
		ttl, err := strconv.Atoi(ttlStr)
		if err != nil {
			return fmt.Errorf("Invalid TTL %s", opts["--ttl"].(string))
		}
		config.TTL = ttl * mul
	}

	if opts["--extend-ttl"].(bool) {
		config.ExtendTTL = true
	}

	// Enable archive mode ?
	if opts["-a"].(bool) || opts["--archive"] != nil || config.Archive {
		config.Archive = true

		if opts["--archive"] != nil && opts["--archive"] != "" {
			config.ArchiveMethod = opts["--archive"].(string)
		}
	}

	// Enable secure mode ?
	if opts["--not-secure"].(bool) {
		config.Secure = false
	} else if opts["-s"].(bool) || opts["--secure"] != nil || config.Secure {
		config.Secure = true
		if opts["--secure"] != nil && opts["--secure"].(string) != "" {
			config.SecureMethod = opts["--secure"].(string)
		}
	}

	// Enable password protection ?
	if opts["-p"].(bool) {
		fmt.Printf("Login [plik]: ")
		var err error
		_, err = fmt.Scanln(&config.Login)
		if err != nil && err.Error() != "unexpected newline" {
			return fmt.Errorf("Unable to get login : %s", err)
		}
		if config.Login == "" {
			config.Login = "plik"
		}
		fmt.Printf("Password: ")
		_, err = fmt.Scanln(&config.Password)
		if err != nil {
			return fmt.Errorf("Unable to get password : %s", err)
		}
	} else if opts["--password"] != nil && opts["--password"].(string) != "" {
		credentials := opts["--password"].(string)
		sepIndex := strings.Index(credentials, ":")
		var login, password string
		if sepIndex > 0 {
			login = credentials[:sepIndex]
			password = credentials[sepIndex+1:]
		} else {
			login = "plik"
			password = credentials
		}
		config.Login = login
		config.Password = password
	}

	// Override upload token ?
	if opts["--token"] != nil && opts["--token"].(string) != "" {
		config.Token = opts["--token"].(string)
	}

	// Ask for token
	if config.Token == "-" {
		fmt.Printf("Token: ")
		var err error
		_, err = fmt.Scanln(&config.Token)
		if err != nil {
			return fmt.Errorf("Unable to get token : %s", err)
		}
	}

	if opts["--stdin"].(bool) {
		config.DisableStdin = false
	}

	return
}
