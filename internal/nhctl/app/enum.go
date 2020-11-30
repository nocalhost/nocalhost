package app

type SvcType string

const (
	Deployment SvcType = "deployment"

	DevImageFlagAnnotationKey   = "nhctl.dev.image.revision"
	DevImageFlagAnnotationValue = "first"
)
