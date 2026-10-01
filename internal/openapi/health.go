package openapi

// @Summary Report whether the API process is alive.
// @Tags Health
// @Success 200 {object} Health "Completed successfully."
// @Failure 500 {object} ServerError "The request could not be completed."
// @Router /health/live [get]
func GetHealthLive() {}

// @Summary Report whether PostgreSQL is reachable.
// @Tags Health
// @Success 200 {object} Health "Completed successfully."
// @Failure 500 {object} ServerError "The request could not be completed."
// @Failure 503 {object} Health "PostgreSQL is unavailable."
// @Router /health/ready [get]
func GetHealthReady() {}

type Health struct {
	Status string `json:"status" validate:"required" example:"ok"`
}
