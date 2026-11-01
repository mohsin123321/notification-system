package pure

import (
	"bytes"
	"embed"
	"fmt"
	"text/template"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

// Render executes the template for the given notification type and channel,
// substituting payload values into the template.
func Render(notifType, channel string, payload map[string]any) (string, error) {
	name := fmt.Sprintf("templates/%s.%s.tmpl", notifType, channel)
	data, err := templateFS.ReadFile(name)
	if err != nil {
		return "", fmt.Errorf("template not found for %s/%s: %w", notifType, channel, err)
	}

	tmpl, err := template.New(name).Parse(string(data))
	if err != nil {
		return "", fmt.Errorf("template parse error: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, payload); err != nil {
		return "", fmt.Errorf("template execute error: %w", err)
	}
	return buf.String(), nil
}
