package openapi

import onboardingmodels "github.com/Rahmannugar/consumel-server/internal/onboarding/models"

func onboardingOperations() []operation {
	return []operation{{
		Method: "post", Path: "/onboarding", Summary: "Create the signed-in owner's organization and first project.",
		Request: "OnboardingSetupRequest", SuccessCode: "201", Success: "OnboardingSetup",
		SuccessDescription: "The organization, owner access, first project, Sandbox, and Live environment were created, and the welcome email was queued.",
		AlternateSuccess:   map[string]string{"200": "A repeated request returned the existing first project without creating duplicates or queueing another welcome email."},
		Protected:          true, Errors: map[string]string{"400": "OnboardingInvalid", "409": "OnboardingConflict", "500": "OnboardingFailed"},
	}}
}

func onboardingSchemas() map[string]any {
	return map[string]any{
		"OnboardingSetupRequest": onboardingmodels.SetupRequestOpenAPISchema(),
		"OnboardingSetup": object([]string{"organization", "project"}, map[string]any{
			"organization": schemaReference("OnboardingOrganization"), "project": schemaReference("Project"),
		}),
		"OnboardingOrganization": object([]string{"id", "name"}, map[string]any{
			"id": map[string]any{"type": "string"}, "name": map[string]any{"type": "string"},
		}),
	}
}

func onboardingErrorResponses() map[string]any {
	return map[string]any{
		"OnboardingConflict": errorResponseWithMessage("The account already has organization access that cannot be changed by onboarding.", "organization_already_exists", "This account already belongs to an organization that needs attention."),
		"OnboardingFailed":   errorResponseWithMessage("The onboarding transaction could not be completed.", "onboarding_failed", "Consumel could not create your project. Try again shortly."),
		"OnboardingInvalid":  errorResponseWithMessage("The organization name or project name is invalid.", "invalid_request", "Enter an organization name and project name between 1 and 120 characters."),
	}
}

func onboardingExample(name string) (map[string]any, bool) {
	switch name {
	case "OnboardingSetupRequest":
		return map[string]any{"organizationName": "Acme", "projectName": "Acme API"}, true
	case "OnboardingSetup":
		project := projectExample()
		return map[string]any{
			"organization": map[string]any{"id": project["organizationId"], "name": "Acme"},
			"project":      project,
		}, true
	default:
		return nil, false
	}
}
