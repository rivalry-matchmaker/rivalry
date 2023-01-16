package templates

import (
	_ "embed"
)

var (
	//go:embed frontend_service.yaml
	FrontendService []byte

	//go:embed frontend_deployment.yaml
	FrontendDeployment []byte

	//go:embed accumulator_deployment.yaml
	AccumulatorDeployment []byte

	//go:embed dispenser_deployment.yaml
	DispenserDeployment []byte
)
