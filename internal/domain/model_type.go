package domain

import "fmt"

type ModelType string

const (
	ModelTypeFlux2Trainer          ModelType = "flux-2-trainer"
	ModelTypeFlux2Klein4BBase      ModelType = "flux-2-klein-4b-base-trainer"
	ModelTypeFlux2Klein9BBase      ModelType = "flux-2-klein-9b-base-trainer"
	ModelTypeQwenImage2512Trainer  ModelType = "qwen-image-2512-trainer"
)

var validModelTypes = map[ModelType]bool{
	ModelTypeFlux2Trainer:         true,
	ModelTypeFlux2Klein4BBase:     true,
	ModelTypeFlux2Klein9BBase:     true,
	ModelTypeQwenImage2512Trainer: true,
}

func ParseModelType(s string) (ModelType, error) {
	mt := ModelType(s)
	if !validModelTypes[mt] {
		return "", fmt.Errorf("%w: %q", ErrInvalidModelType, s)
	}
	return mt, nil
}

func (mt ModelType) String() string {
	return string(mt)
}

func (mt ModelType) IsFluxVariant() bool {
	return mt == ModelTypeFlux2Trainer || mt == ModelTypeFlux2Klein4BBase || mt == ModelTypeFlux2Klein9BBase
}

func (mt ModelType) FluxVariant() string {
	switch mt {
	case ModelTypeFlux2Trainer:
		return "full"
	case ModelTypeFlux2Klein4BBase:
		return "klein-4b"
	case ModelTypeFlux2Klein9BBase:
		return "klein-9b"
	default:
		return ""
	}
}
