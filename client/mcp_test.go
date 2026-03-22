package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

func TestClientForProfile_Empty(t *testing.T) {
	cfg := &CliConfig{ConfigPath: "/nonexistent"}
	defaultClient := clientFromConfig(cfg)

	// Empty profile should return the default client
	client, err := clientForProfile(cfg, defaultClient, "")
	require.NoError(t, err)
	require.Same(t, defaultClient, client, "empty profile should return same pointer")
}

func TestClientForProfile_ValidProfile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".plikrc")
	err := os.WriteFile(path, []byte(`
URL = "https://base.example.com"

[Profiles.work]
URL = "https://work.example.com"
Token = "work-token-123"
`), 0600)
	require.NoError(t, err)

	cfg := &CliConfig{ConfigPath: path, URL: "https://base.example.com"}
	defaultClient := clientFromConfig(cfg)

	client, err := clientForProfile(cfg, defaultClient, "work")
	require.NoError(t, err)
	require.NotSame(t, defaultClient, client, "profile client should be a different pointer")
	require.Equal(t, "https://work.example.com", client.URL)
	require.Equal(t, "work-token-123", client.Token)
}

func TestClientForProfile_InvalidProfile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".plikrc")
	err := os.WriteFile(path, []byte(`
URL = "https://base.example.com"

[Profiles.work]
URL = "https://work.example.com"
Token = ""
`), 0600)
	require.NoError(t, err)

	cfg := &CliConfig{ConfigPath: path}
	defaultClient := clientFromConfig(cfg)

	_, err = clientForProfile(cfg, defaultClient, "typo")
	require.Error(t, err)
	require.Contains(t, err.Error(), "typo")
	require.Contains(t, err.Error(), "not found")
}

func TestClientForProfile_LockedByFlag(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".plikrc")
	err := os.WriteFile(path, []byte(`
URL = "https://base.example.com"

[Profiles.work]
URL = "https://work.example.com"
Token = ""

[Profiles.local]
URL = "http://localhost:8080"
Token = ""
`), 0600)
	require.NoError(t, err)

	// Simulate MCP started with -P work
	cfg := &CliConfig{ConfigPath: path, ActiveProfiles: []string{"work"}}
	defaultClient := clientFromConfig(cfg)

	// Switching to a different profile should be rejected
	_, err = clientForProfile(cfg, defaultClient, "local")
	require.Error(t, err)
	require.Contains(t, err.Error(), "profile switching is not available")
	require.Contains(t, err.Error(), "-P work")

	// Empty profile (use default) should still work
	client, err := clientForProfile(cfg, defaultClient, "")
	require.NoError(t, err)
	require.Same(t, defaultClient, client)
}

func TestClientForProfile_Composition(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".plikrc")
	err := os.WriteFile(path, []byte(`
URL = "https://base.example.com"

[Profiles.work]
URL = "https://work.example.com"
Token = "work-token"

[Profiles.zip]
OneShot = true
`), 0600)
	require.NoError(t, err)

	cfg := &CliConfig{ConfigPath: path}
	defaultClient := clientFromConfig(cfg)

	// "work,zip" should merge: work URL + work token, zip OneShot
	client, err := clientForProfile(cfg, defaultClient, "work,zip")
	require.NoError(t, err)
	require.Equal(t, "https://work.example.com", client.URL)
	require.Equal(t, "work-token", client.Token)
}

func TestListProfiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".plikrc")
	err := os.WriteFile(path, []byte(`
URL = "https://base.example.com"
DefaultProfile = "work"

[Profiles.work]
URL = "https://work.example.com"
Token = ""

[Profiles.local]
URL = "http://localhost:8080"
Token = ""

[Profiles.zip]
Archive = true
ArchiveMethod = "zip"
`), 0600)
	require.NoError(t, err)

	t.Run("unlocked", func(t *testing.T) {
		// No ActiveProfiles → all profiles visible
		plikrc, _, err := loadPlikrc(path)
		require.NoError(t, err)
		require.Equal(t, "work", plikrc.DefaultProfile)
		require.Len(t, plikrc.Profiles, 3)

		// Verify profile URLs (zip has no URL, inherits base)
		require.Equal(t, "https://work.example.com", plikrc.Profiles["work"].URL)
		require.Equal(t, "http://localhost:8080", plikrc.Profiles["local"].URL)
		require.Equal(t, "", plikrc.Profiles["zip"].URL, "zip has no URL override")
	})

	t.Run("locked_by_flag", func(t *testing.T) {
		// When ActiveProfiles is set, the handler returns empty (no profiles listed)
		cfg := &CliConfig{ConfigPath: path, ActiveProfiles: []string{"work"}}
		handler := makeListProfilesHandler(cfg)
		result, _, err := handler(context.Background(), nil, ListProfilesInput{})
		require.NoError(t, err)
		require.False(t, result.IsError)
		// Should contain empty JSON with null profiles
		text := result.Content[0].(*mcp.TextContent).Text
		require.Contains(t, text, `"profiles": null`)
		require.NotContains(t, text, `"default_profile"`)
	})
}
