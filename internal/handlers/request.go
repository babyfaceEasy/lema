package handlers

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type monitorRepositoryRequest struct {
	RepositoryName string    `json:"repo_name"`
	OwnerName      string    `json:"owner_name"`
	StartTimeStr   string    `json:"start_time"`
	StartTime      time.Time `json:"-"`
}

func (r monitorRepositoryRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.RepositoryName, validation.Required),
		validation.Field(&r.OwnerName, validation.Required),
	)
}

type resetCollectionRequest struct {
	RepositoryName string    `json:"repo_name"`
	OwnerName      string    `json:"owner_name"`
	StartTimeStr   string    `json:"start_time"`
	StartTime      time.Time `json:"-"`
}

func (r resetCollectionRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.RepositoryName, validation.Required),
		validation.Field(&r.OwnerName, validation.Required),
	)
}
