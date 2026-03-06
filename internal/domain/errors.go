package domain

import "errors"

var (
	ErrInvalidModelType    = errors.New("invalid model type")
	ErrInvalidRequest      = errors.New("invalid training request")
	ErrJobNotFound         = errors.New("job not found")
	ErrInvalidTransition   = errors.New("invalid state transition")
	ErrJobAlreadyCancelled = errors.New("job already cancelled")
	ErrJobAlreadyComplete  = errors.New("job already completed")
	ErrProviderUnavailable = errors.New("no GPU provider available")
	ErrUploadFailed        = errors.New("upload failed")
	ErrTrainingFailed      = errors.New("training failed")
	ErrProvisionTimeout    = errors.New("provisioning timed out")
	ErrTrainingTimeout     = errors.New("training timed out")
)
