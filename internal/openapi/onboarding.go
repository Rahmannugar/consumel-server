package openapi

// @Summary Create the signed-in owner's organization and first project.
// @Tags Onboarding
// @Param body body models.SetupRequest true "Organization and project names."
// @Success 200 {object} OnboardingSetup "A repeated request returned the existing first project without creating duplicates or queueing another welcome email."
// @Success 201 {object} OnboardingSetup "The organization, owner access, first project, Sandbox, and Live environment were created, and the welcome email was queued."
// @Failure 400 {object} OnboardingInvalid "The organization name or project name is invalid."
// @Failure 401 {object} NotAuthenticated "Authentication is required."
// @Failure 409 {object} OnboardingConflict "The account already has organization access that cannot be changed by onboarding."
// @Failure 500 {object} OnboardingFailed "The onboarding transaction could not be completed."
// @Security productionCookieSession
// @Security localCookieSession
// @Router /onboarding [post]
func PostOnboarding() {}

type OnboardingOrganization struct {
	ID   string `json:"id" validate:"required" example:"0199a416-d2c8-75ea-bdb4-1d13c627169b"`
	Name string `json:"name" validate:"required" example:"Acme"`
}

type OnboardingSetup struct {
	Organization OnboardingOrganization `json:"organization" validate:"required"`
	Project      Project                `json:"project" validate:"required"`
}

type OnboardingConflict struct {
	Error OnboardingConflictError `json:"error" validate:"required"`
}

type OnboardingConflictError struct {
	Code    string `json:"code" validate:"required" example:"organization_already_exists"`
	Message string `json:"message,omitempty" example:"This account already belongs to an organization that needs attention."`
}

type OnboardingFailed struct {
	Error OnboardingFailedError `json:"error" validate:"required"`
}

type OnboardingFailedError struct {
	Code    string `json:"code" validate:"required" example:"onboarding_failed"`
	Message string `json:"message,omitempty" example:"Consumel could not create your project. Try again shortly."`
}

type OnboardingInvalid struct {
	Error OnboardingInvalidError `json:"error" validate:"required"`
}

type OnboardingInvalidError struct {
	Code    string `json:"code" validate:"required" example:"invalid_request"`
	Message string `json:"message,omitempty" example:"Enter an organization name and project name between 1 and 120 characters."`
}
