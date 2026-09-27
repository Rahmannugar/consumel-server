package templates

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
)

//go:embed welcome.html
var welcomeHTML string

type Email struct {
	Subject string
	Text    string
	HTML    string
}

func Render(organizationName, projectName, projectURL string) (Email, error) {
	data := struct {
		OrganizationName string
		ProjectName      string
		ProjectURL       string
	}{
		OrganizationName: organizationName,
		ProjectName:      projectName,
		ProjectURL:       projectURL,
	}
	parsed, err := template.New("welcome").Parse(welcomeHTML)
	if err != nil {
		return Email{}, fmt.Errorf("parse welcome email: %w", err)
	}
	var rendered bytes.Buffer
	if err := parsed.Execute(&rendered, data); err != nil {
		return Email{}, fmt.Errorf("render welcome email: %w", err)
	}
	text := fmt.Sprintf(
		"Welcome to Consumel\n\n%s is ready in %s.\n\nOpen your project: %s\n",
		projectName, organizationName, projectURL,
	)
	return Email{Subject: "Welcome to Consumel", Text: text, HTML: rendered.String()}, nil
}
