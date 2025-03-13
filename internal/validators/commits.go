package validators

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type SaveCommitsRequest struct {
	RepositoryName string `json:"repository_name"`
	StartDate      string `json:"start_date"` // Only store the date as string (YYYY-MM-DD)
}

func (req SaveCommitsRequest) Validate() error {
	return validation.ValidateStruct(&req,
		validation.Field(&req.RepositoryName, validation.Required.Error("repository_name is required")),
		validation.Field(&req.StartDate, validation.Required.Error("start_date is required")),
	)
}
