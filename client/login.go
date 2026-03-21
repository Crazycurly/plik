package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/BurntSushi/toml"

	"github.com/root-gg/plik/plik"
)

type cliAuthInitRequest struct {
	Hostname string `json:"hostname"`
}

type cliAuthInitResponse struct {
	Code      string `json:"code"`
	Secret    string `json:"secret"`
	VerifyURL string `json:"verifyURL"`
	ExpiresIn int    `json:"expiresIn"`
}

type cliAuthPollRequest struct {
	Code   string `json:"code"`
	Secret string `json:"secret"`
}

type cliAuthPollResponse struct {
	Status string `json:"status"`
	Token  string `json:"token,omitempty"`
}

// login performs the device authorization flow.
// It initiates a session on the server, displays a URL for the user to
// authenticate in their browser, polls for approval, and saves the token.
func login(cfg *CliConfig, client *plik.Client) error {
	// Get hostname for token comment
	hostname, _ := os.Hostname()

	// Step 1: Initiate the CLI auth session
	initReq := cliAuthInitRequest{Hostname: hostname}
	initBody, err := json.Marshal(initReq)
	if err != nil {
		return fmt.Errorf("unable to serialize init request: %s", err)
	}

	resp, err := client.HTTPClient.Post(cfg.URL+"/auth/cli/init", "application/json", bytes.NewBuffer(initBody))
	if err != nil {
		return fmt.Errorf("unable to contact server: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server error (%d): %s", resp.StatusCode, string(body))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("unable to read response: %s", err)
	}

	var initResp cliAuthInitResponse
	if err := json.Unmarshal(respBody, &initResp); err != nil {
		return fmt.Errorf("unable to parse response: %s", err)
	}

	// Step 2: Build verify URL from client config (not server response)
	// The server's VerifyURL may use its internal address (e.g. 127.0.0.1:8080)
	verifyURL := fmt.Sprintf("%s/#/cli-auth?code=%s", cfg.URL, initResp.Code)
	if hostname != "" {
		verifyURL += "&hostname=" + url.QueryEscape(hostname)
	}

	fmt.Printf("\n  Open this URL in your browser to authenticate:\n\n")
	fmt.Printf("    %s\n\n", verifyURL)
	fmt.Printf("  Your one-time code: %s\n\n", initResp.Code)
	fmt.Printf("  Waiting for authentication...")

	// Best-effort: try to open the browser
	openBrowser(verifyURL)

	// Step 3: Poll for approval
	timeout := time.Duration(initResp.ExpiresIn) * time.Second
	deadline := time.Now().Add(timeout)
	pollInterval := 2 * time.Second

	for time.Now().Before(deadline) {
		time.Sleep(pollInterval)

		token, err := pollForToken(client, cfg.URL, initResp.Code, initResp.Secret)
		if err != nil {
			return err
		}

		if token != "" {
			fmt.Printf("\n\n")

			// Step 4: Save token to config
			err := saveToken(cfg, token)
			if err != nil {
				// Still print the token so the user isn't locked out
				fmt.Printf("  Warning: unable to save token to config: %s\n", err)
				fmt.Printf("  Your token is: %s\n", token)
				fmt.Printf("  Add it to your .plikrc manually.\n")
				return nil
			}

			fmt.Printf("  ✓ Authenticated! Token saved to ~/.plikrc\n")
			fmt.Printf("  Token: %s\n\n", token)
			return nil
		}
	}

	fmt.Printf("\n\n")
	return fmt.Errorf("authentication timed out after %s — please try again", timeout)
}

func pollForToken(client *plik.Client, serverURL, code, secret string) (string, error) {
	pollReq := cliAuthPollRequest{Code: code, Secret: secret}
	pollBody, err := json.Marshal(pollReq)
	if err != nil {
		return "", fmt.Errorf("unable to serialize poll request: %s", err)
	}

	resp, err := client.HTTPClient.Post(serverURL+"/auth/cli/poll", "application/json", bytes.NewBuffer(pollBody))
	if err != nil {
		// Transient network error — keep polling
		return "", nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("poll error (%d): %s", resp.StatusCode, string(body))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil
	}

	var pollResp cliAuthPollResponse
	if err := json.Unmarshal(respBody, &pollResp); err != nil {
		return "", nil
	}

	if pollResp.Status == "approved" && pollResp.Token != "" {
		return pollResp.Token, nil
	}

	return "", nil
}

func saveToken(cfg *CliConfig, token string) error {
	// Use the path from the loaded config, or derive it
	path := cfg.ConfigPath
	if path == "" {
		path = configFilePath()
	}

	// Load the existing config file to preserve profiles
	var plikrc PlikrcFile
	plikrc.CliConfig = *NewUploadConfig()
	if _, err := os.Stat(path); err == nil {
		md, err := toml.DecodeFile(path, &plikrc)
		if err != nil {
			return fmt.Errorf("unable to read existing config: %s", err)
		}
		plikrc.metadata = md
	}

	// Save token to the active profile or the top-level config
	if cfg.ActiveProfile != "" {
		if plikrc.Profiles == nil {
			return fmt.Errorf("profile %q not found in config (no profiles defined)", cfg.ActiveProfile)
		}
		profile, ok := plikrc.Profiles[cfg.ActiveProfile]
		if !ok {
			return fmt.Errorf("profile %q not found in config", cfg.ActiveProfile)
		}
		profile.Token = token
		plikrc.Profiles[cfg.ActiveProfile] = profile
	} else {
		plikrc.CliConfig.Token = token
	}

	return saveConfig(path, &plikrc)
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return
	}
	// Best-effort — ignore errors
	_ = cmd.Start()
}
