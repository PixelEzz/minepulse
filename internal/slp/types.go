package slp

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// StatusResponse represents the modern JSON structure returned by Minecraft 1.7+ servers.
type StatusResponse struct {
	Version struct {
		Name     string `json:"name"`
		Protocol int    `json:"protocol"`
	} `json:"version"`
	Players struct {
		Max    int `json:"max"`
		Online int `json:"online"`
		Sample []struct {
			Name string `json:"name"`
			ID   string `json:"id"`
		} `json:"sample,omitempty"`
	} `json:"players"`
	Description json.RawMessage `json:"description"`
	Favicon     string          `json:"favicon,omitempty"`
	EnforcesSecureChat bool     `json:"enforcesSecureChat,omitempty"`
	PreviewsChat       bool     `json:"previewsChat,omitempty"`

	// Parsed clean MOTD
	CleanMOTD string `json:"clean_motd"`
}

var mcColorCodeRegex = regexp.MustCompile(`(?i)§[0-9a-fk-or]`)

// CleanMinecraftFormatting removes § color & format codes from a string.
func CleanMinecraftFormatting(s string) string {
	cleaned := mcColorCodeRegex.ReplaceAllString(s, "")
	return strings.TrimSpace(cleaned)
}

// ParseMOTD parses the polymorphic description field (either string or JSON chat component).
func ParseMOTD(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	// Case 1: Plain string (e.g. "A Minecraft Server")
	var simple string
	if err := json.Unmarshal(raw, &simple); err == nil {
		return CleanMinecraftFormatting(simple)
	}

	// Case 2: Chat component object
	var component struct {
		Text  string `json:"text"`
		Extra []struct {
			Text string `json:"text"`
		} `json:"extra,omitempty"`
	}
	if err := json.Unmarshal(raw, &component); err == nil {
		var sb strings.Builder
		sb.WriteString(component.Text)
		for _, extra := range component.Extra {
			sb.WriteString(extra.Text)
		}
		return CleanMinecraftFormatting(sb.String())
	}

	return CleanMinecraftFormatting(string(raw))
}

// ParseStatusResponse decodes the raw JSON status string.
func ParseStatusResponse(data []byte) (*StatusResponse, error) {
	var resp StatusResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse status JSON: %w", err)
	}
	resp.CleanMOTD = ParseMOTD(resp.Description)
	return &resp, nil
}
