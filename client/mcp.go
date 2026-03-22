package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/root-gg/plik/plik"
	"github.com/root-gg/plik/server/common"
)

// --- Tool input types ---

// UploadTextInput is the input schema for the upload_text tool
type UploadTextInput struct {
	plik.UploadParams
	Filename string `json:"filename" jsonschema:"Name for the uploaded file (e.g. snippet.go)"`
	Content  string `json:"content" jsonschema:"Text content to upload"`
	Profile  string `json:"profile,omitempty" jsonschema:"Profile name from .plikrc (e.g. 'work'). Supports composition ('work,zip'). Omit to use the default."`
}

// UploadFileInput is the input schema for the upload_file tool
type UploadFileInput struct {
	plik.UploadParams
	Path    string `json:"path" jsonschema:"Absolute path to the file to upload"`
	Profile string `json:"profile,omitempty" jsonschema:"Profile name from .plikrc (e.g. 'work'). Supports composition ('work,zip'). Omit to use the default."`
}

// UploadFilesInput is the input schema for the upload_files tool
type UploadFilesInput struct {
	plik.UploadParams
	Paths   []string `json:"paths" jsonschema:"List of absolute paths to files to upload"`
	Profile string   `json:"profile,omitempty" jsonschema:"Profile name from .plikrc (e.g. 'work'). Supports composition ('work,zip'). Omit to use the default."`
}

// ServerInfoInput is the input schema for the server_info tool (no params)
type ServerInfoInput struct{}

// ListProfilesInput is the input schema for the list_profiles tool (no params)
type ListProfilesInput struct{}

// --- Tool output helpers ---

func uploadResult(upload *plik.Upload) *mcp.CallToolResult {
	jsonBytes, _ := json.MarshalIndent(upload.WithURL(), "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(jsonBytes)},
		},
	}
}

func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg},
		},
		IsError: true,
	}
}

// --- Profile-aware client resolution ---

// clientFromConfig creates a plik.Client from a fully-resolved CliConfig,
// carrying over all upload defaults so profile settings apply automatically.
func clientFromConfig(cfg *CliConfig) *plik.Client {
	client := plik.NewClient(cfg.URL)
	client.Debug = cfg.Debug
	client.ClientName = "plik_mcp"

	// Carry over default upload params from config
	// Note: Stream is intentionally excluded — it blocks until someone
	// downloads, which would hang the MCP tool call indefinitely.
	client.Token = cfg.Token
	client.OneShot = cfg.OneShot
	client.Removable = cfg.Removable
	client.TTL = cfg.TTL
	client.ExtendTTL = cfg.ExtendTTL
	client.Comments = cfg.Comments
	client.Login = cfg.Login
	client.Password = cfg.Password

	if cfg.Insecure {
		client.Insecure()
	}

	return client
}

// clientForProfile returns a plik.Client for the given profile name.
// If profile is empty, it returns the default client (zero overhead).
// If the MCP server was started with -P, profile switching is rejected for safety.
// Otherwise it re-reads the config file and resolves the profile.
func clientForProfile(baseCfg *CliConfig, defaultClient *plik.Client, profile string) (*plik.Client, error) {
	if profile == "" {
		return defaultClient, nil
	}

	// Safety gate: if MCP was started with -P, lock to that profile
	if baseCfg.ProfileSource == "flag" {
		return nil, fmt.Errorf("profile switching is locked by -P %s", strings.Join(baseCfg.ActiveProfiles, ","))
	}

	cfg, err := LoadConfigFromFile(baseCfg.ConfigPath, profile)
	if err != nil {
		return nil, err
	}

	return clientFromConfig(cfg), nil
}

// --- Tool handlers ---

func makeUploadTextHandler(baseCfg *CliConfig, defaultClient *plik.Client) mcp.ToolHandlerFor[UploadTextInput, any] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input UploadTextInput) (*mcp.CallToolResult, any, error) {
		if input.Filename == "" {
			return errorResult("filename is required"), nil, nil
		}
		if input.Content == "" {
			return errorResult("content is required"), nil, nil
		}

		client, err := clientForProfile(baseCfg, defaultClient, input.Profile)
		if err != nil {
			return errorResult(fmt.Sprintf("profile error: %s", err)), nil, nil
		}

		upload := client.NewUpload()
		input.UploadParams.Apply(upload)

		upload.AddFileFromReader(input.Filename, strings.NewReader(input.Content))

		err = upload.Upload()
		if err != nil {
			return errorResult(fmt.Sprintf("upload failed: %s", err)), nil, nil
		}

		return uploadResult(upload), nil, nil
	}
}

func makeUploadFileHandler(baseCfg *CliConfig, defaultClient *plik.Client) mcp.ToolHandlerFor[UploadFileInput, any] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input UploadFileInput) (*mcp.CallToolResult, any, error) {
		if input.Path == "" {
			return errorResult("path is required"), nil, nil
		}

		// Verify file exists
		if _, err := os.Stat(input.Path); err != nil {
			return errorResult(fmt.Sprintf("file not found: %s", input.Path)), nil, nil
		}

		client, err := clientForProfile(baseCfg, defaultClient, input.Profile)
		if err != nil {
			return errorResult(fmt.Sprintf("profile error: %s", err)), nil, nil
		}

		upload := client.NewUpload()
		input.UploadParams.Apply(upload)

		if _, err := upload.AddFileFromPath(input.Path); err != nil {
			return errorResult(fmt.Sprintf("failed to add file: %s", err)), nil, nil
		}

		if err := upload.Upload(); err != nil {
			return errorResult(fmt.Sprintf("upload failed: %s", err)), nil, nil
		}

		return uploadResult(upload), nil, nil
	}
}

func makeUploadFilesHandler(baseCfg *CliConfig, defaultClient *plik.Client) mcp.ToolHandlerFor[UploadFilesInput, any] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input UploadFilesInput) (*mcp.CallToolResult, any, error) {
		if len(input.Paths) == 0 {
			return errorResult("at least one path is required"), nil, nil
		}

		client, err := clientForProfile(baseCfg, defaultClient, input.Profile)
		if err != nil {
			return errorResult(fmt.Sprintf("profile error: %s", err)), nil, nil
		}

		upload := client.NewUpload()
		input.UploadParams.Apply(upload)

		for _, path := range input.Paths {
			if _, err := os.Stat(path); err != nil {
				return errorResult(fmt.Sprintf("file not found: %s", path)), nil, nil
			}
			if _, err := upload.AddFileFromPath(path); err != nil {
				return errorResult(fmt.Sprintf("failed to add file %s: %s", path, err)), nil, nil
			}
		}

		if err := upload.Upload(); err != nil {
			return errorResult(fmt.Sprintf("upload failed: %s", err)), nil, nil
		}

		return uploadResult(upload), nil, nil
	}
}

func makeServerInfoHandler(baseCfg *CliConfig, defaultClient *plik.Client) mcp.ToolHandlerFor[ServerInfoInput, any] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input ServerInfoInput) (*mcp.CallToolResult, any, error) {
		type serverInfo struct {
			ServerURL         string                `json:"server_url"`
			ActiveProfiles    []string              `json:"active_profiles,omitempty"`
			AvailableProfiles []string              `json:"available_profiles,omitempty"`
			Version           *common.BuildInfo     `json:"version,omitempty"`
			Config            *common.Configuration `json:"config,omitempty"`
		}

		info := &serverInfo{
			ServerURL:         defaultClient.URL,
			ActiveProfiles:    baseCfg.ActiveProfiles,
			AvailableProfiles: baseCfg.AvailableProfiles,
		}

		version, err := defaultClient.GetServerVersion()
		if err != nil {
			return errorResult(fmt.Sprintf("failed to get server version: %s", err)), nil, nil
		}
		info.Version = version

		cfg, err := defaultClient.GetServerConfig()
		if err != nil {
			return errorResult(fmt.Sprintf("failed to get server config: %s", err)), nil, nil
		}
		info.Config = cfg

		jsonBytes, _ := json.MarshalIndent(info, "", "  ")
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: string(jsonBytes)},
			},
		}, nil, nil
	}
}

func makeListProfilesHandler(baseCfg *CliConfig) mcp.ToolHandlerFor[ListProfilesInput, any] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input ListProfilesInput) (*mcp.CallToolResult, any, error) {
		type profileInfo struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		}

		type listProfilesOutput struct {
			DefaultProfile string        `json:"default_profile,omitempty"`
			Profiles       []profileInfo `json:"profiles"`
		}

		// If MCP was started with -P, return empty to discourage profile switching
		if baseCfg.ProfileSource == "flag" {
			jsonBytes, _ := json.MarshalIndent(&listProfilesOutput{}, "", "  ")
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: string(jsonBytes)},
				},
			}, nil, nil
		}

		// Re-read the config file to get the current profile definitions
		plikrc, _, err := loadPlikrc(baseCfg.ConfigPath)
		if err != nil {
			return errorResult(fmt.Sprintf("failed to load config: %s", err)), nil, nil
		}

		output := &listProfilesOutput{
			DefaultProfile: plikrc.DefaultProfile,
		}

		// Sort profile names for deterministic output
		names := make([]string, 0, len(plikrc.Profiles))
		for name := range plikrc.Profiles {
			names = append(names, name)
		}
		sort.Strings(names)

		for _, name := range names {
			profile := plikrc.Profiles[name]
			url := profile.URL
			if url == "" {
				url = plikrc.CliConfig.URL // inherit from base
			}
			output.Profiles = append(output.Profiles, profileInfo{
				Name: name,
				URL:  url,
			})
		}

		jsonBytes, _ := json.MarshalIndent(output, "", "  ")
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: string(jsonBytes)},
			},
		}, nil, nil
	}
}

// --- Prompts ---

var uploadGuidePrompt = `You have access to Plik file upload tools. Here's how to use them:

## Uploading text content
Use the upload_text tool to upload generated text, code snippets, logs, or any text content.
This avoids creating temporary files on the filesystem.

## Uploading files from disk
Use upload_file for a single file, or upload_files for multiple files.
Pass the absolute file path(s) — the server reads them directly from the local filesystem.
There is no file size limit.

## Upload options
All upload tools support these optional parameters:
- ttl: Time to live in seconds (0 = server default)
- one_shot: Delete the file after it's downloaded once
- removable: Allow anyone to delete the file
- stream: Don't store the file on the server, stream directly to downloader (blocking)
- extend_ttl: Extend upload expiration date by TTL when accessed
- comments: Arbitrary markdown comment to attach to the upload
- login / password: HTTP basic auth protection for the upload
- token: Authentication token to link the upload to a specific user

Note: Some features may or may not be enabled on the server. Use the server_info tool to discover the server configuration.

## Using profiles
If the user has multiple profiles defined in ~/.plikrc, you can target a specific server
by passing the "profile" parameter on any upload tool. Use the list_profiles tool to discover
available profiles and their server URLs. Profile composition is supported: profile "work,zip"
applies the work profile settings first, then zip overrides on top.

## Getting server info
Use the server_info tool to check the server's configuration, version, and capabilities.
It returns:
- server_url: The configured Plik server URL
- active_profiles: Currently active profile(s) for the MCP session
- available_profiles: All profiles defined in ~/.plikrc
- version: Server build info (version, commit, date)
- config: Server configuration including:
  - maxFileSize: Maximum file size in bytes (0 = unlimited)
  - maxFilePerUpload: Maximum number of files per upload
  - defaultTTL / maxTTL: Default and maximum TTL in seconds (0 = unlimited)
  - feature_*: Feature flags that control which upload options are available

Feature flags can be "enabled", "disabled", or "forced" and map to upload parameters:
  - feature_one_shot → one_shot
  - feature_removable → removable
  - feature_stream → stream
  - feature_password → login / password
  - feature_comments → comments
  - feature_set_ttl → ttl
  - feature_extend_ttl → extend_ttl
  - feature_authentication → token

## Understanding the result
Upload tools return a JSON object with:
- upload_url: Link to the upload page (shows all files, can be shared)
- files: Array of objects with name and download_url (direct download link for each file)
- ttl: Server-applied time to live in seconds (may differ from requested value)
- expires_at: ISO 8601 expiration timestamp (if TTL > 0)

Share the upload_url for a web page view, or individual download_url for direct file downloads.`

// --- RunMCPServer ---

// newMCPServer creates the MCP server with all tools and prompts registered.
// It is separated from RunMCPServer so it can be used in tests with custom transports.
func newMCPServer(baseCfg *CliConfig, defaultClient *plik.Client) *mcp.Server {
	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "plik",
			Version: common.GetBuildInfo().Version,
		},
		nil,
	)

	// Register tools
	mcp.AddTool(server, &mcp.Tool{
		Name:        "upload_text",
		Description: "Upload text content as a file to Plik. Use this to upload generated text, code snippets, or any text content without creating temporary files.",
	}, makeUploadTextHandler(baseCfg, defaultClient))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "upload_file",
		Description: "Upload a single file from a local filesystem path to Plik.",
	}, makeUploadFileHandler(baseCfg, defaultClient))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "upload_files",
		Description: "Upload multiple files from local filesystem paths to Plik in a single upload.",
	}, makeUploadFilesHandler(baseCfg, defaultClient))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "server_info",
		Description: "Get the Plik server version, configuration, and capabilities.",
	}, makeServerInfoHandler(baseCfg, defaultClient))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_profiles",
		Description: "List available profiles from ~/.plikrc with their server URLs. Use this to discover which profiles can be passed to upload tools.",
	}, makeListProfilesHandler(baseCfg))

	// Register prompts
	server.AddPrompt(&mcp.Prompt{
		Name:        "upload_guide",
		Description: "Instructions on how to upload files to Plik",
	}, func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		return &mcp.GetPromptResult{
			Description: "How to upload files to Plik",
			Messages: []*mcp.PromptMessage{
				{Role: "user", Content: &mcp.TextContent{Text: uploadGuidePrompt}},
			},
		}, nil
	})

	return server
}

// RunMCPServer starts the MCP server over stdio.
func RunMCPServer(cfg *CliConfig) error {
	defaultClient := clientFromConfig(cfg)
	server := newMCPServer(cfg, defaultClient)

	// Run server over stdio
	fmt.Fprintf(os.Stderr, "Plik MCP server starting (server: %s)\n", cfg.URL)
	return server.Run(context.Background(), &mcp.StdioTransport{})
}
