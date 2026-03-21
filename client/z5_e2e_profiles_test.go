package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCLI_Profile_Upload verifies that profile settings flow through the full
// upload path. A profile that sets OneShot = true should produce server metadata
// with oneShot: true.
func TestCLI_Profile_Upload(t *testing.T) {
	dir := t.TempDir()
	createTestFile(t, dir, "FILE1", testContent)

	plikrc := filepath.Join(dir, ".plikrc")
	err := os.WriteFile(plikrc, []byte(`
URL = "https://should-be-overridden.example.com"

[Profiles.ogg]
OneShot = true
`), 0600)
	require.NoError(t, err)

	config, err := LoadConfigFromFile(plikrc, "ogg")
	require.NoError(t, err)

	// Override URL to point at the test server
	config.URL = testServerURL
	config.Quiet = true

	result := runCLI(t, config, map[string]any{
		"FILE": []string{dir + "/FILE1"},
	})

	meta := getUploadMetadata(t, result.Stdout)
	require.Equal(t, true, meta["oneShot"], "profile-set OneShot should be reflected in upload metadata")
}

// TestCLI_Profile_Upload_InheritsBase verifies that profile inheritance works
// through the full upload path. A profile that only overrides URL should
// inherit TTL from the base config.
func TestCLI_Profile_Upload_InheritsBase(t *testing.T) {
	dir := t.TempDir()
	createTestFile(t, dir, "FILE1", testContent)

	plikrc := filepath.Join(dir, ".plikrc")
	err := os.WriteFile(plikrc, []byte(`
URL = "https://should-be-overridden.example.com"
TTL = 3600

[Profiles.minimal]
URL = "http://also-overridden.local"
`), 0600)
	require.NoError(t, err)

	config, err := LoadConfigFromFile(plikrc, "minimal")
	require.NoError(t, err)

	config.URL = testServerURL
	config.Quiet = true

	result := runCLI(t, config, map[string]any{
		"FILE": []string{dir + "/FILE1"},
	})

	meta := getUploadMetadata(t, result.Stdout)
	ttl, ok := meta["ttl"].(float64)
	require.True(t, ok, "ttl should be a number")
	require.Equal(t, float64(3600), ttl, "TTL should be inherited from base config")
}

// TestCLI_Profile_Info verifies that info output includes profile information
// when profiles are configured.
func TestCLI_Profile_Info(t *testing.T) {
	dir := t.TempDir()

	plikrc := filepath.Join(dir, ".plikrc")
	err := os.WriteFile(plikrc, []byte(`
URL = "https://should-be-overridden.example.com"

[Profiles.local]
URL = "http://localhost:8080"

[Profiles.work]
URL = "https://plik.work.corp"
`), 0600)
	require.NoError(t, err)

	config, err := LoadConfigFromFile(plikrc, "local")
	require.NoError(t, err)

	config.URL = testServerURL

	output := runInfo(t, config)

	require.Contains(t, output, "Active profile")
	require.Contains(t, output, "local")
	require.Contains(t, output, "Available profiles")
	require.Contains(t, output, "work")
}

// TestCLI_Profile_Info_NoProfile verifies that info output does NOT contain
// profile lines when no profiles are defined.
func TestCLI_Profile_Info_NoProfile(t *testing.T) {
	dir := t.TempDir()

	plikrc := filepath.Join(dir, ".plikrc")
	err := os.WriteFile(plikrc, []byte(`
URL = "https://should-be-overridden.example.com"
`), 0600)
	require.NoError(t, err)

	config, err := LoadConfigFromFile(plikrc, "")
	require.NoError(t, err)

	config.URL = testServerURL

	output := runInfo(t, config)

	require.NotContains(t, output, "Active profile")
	require.NotContains(t, output, "Available profiles")
}
