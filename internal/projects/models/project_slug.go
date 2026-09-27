package models

import (
	"strings"

	"github.com/google/uuid"
)

const MaximumProjectSlugLength = 120

func NewProjectSlug(name string, projectID uuid.UUID) string {
	var slug strings.Builder
	separatorPending := false
	for _, character := range strings.ToLower(strings.TrimSpace(name)) {
		if character >= 'a' && character <= 'z' || character >= '0' && character <= '9' {
			if separatorPending && slug.Len() > 0 {
				slug.WriteByte('-')
			}
			slug.WriteRune(character)
			separatorPending = false
			continue
		}
		separatorPending = slug.Len() > 0
	}

	value := slug.String()
	if value == "" {
		value = "project-" + strings.ReplaceAll(projectID.String(), "-", "")[:8]
	}
	if len(value) > MaximumProjectSlugLength {
		value = strings.TrimRight(value[:MaximumProjectSlugLength], "-")
	}
	return value
}

func ProjectSlugWithIDSuffix(slug string, projectID uuid.UUID) string {
	suffix := "-" + strings.ReplaceAll(projectID.String(), "-", "")[:8]
	baseLength := MaximumProjectSlugLength - len(suffix)
	base := strings.TrimRight(slug[:min(len(slug), baseLength)], "-")
	return base + suffix
}
