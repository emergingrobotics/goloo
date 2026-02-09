package provider

import (
	"context"
	"time"

	"github.com/emergingrobotics/goloo/internal/config"
)

type VMProvider interface {
	Name() string
	Create(context context.Context, configuration *config.Config, cloudInitPath string) error
	Delete(context context.Context, configuration *config.Config) error
	Status(context context.Context, configuration *config.Config) (*VMStatus, error)
	List(context context.Context) ([]VMStatus, error)
	SSH(context context.Context, configuration *config.Config) error
	Stop(context context.Context, configuration *config.Config) error
	Start(context context.Context, configuration *config.Config) error
}

type VMStatus struct {
	Name      string
	State     string
	IP        string
	Provider  string
	CreatedAt time.Time
}
