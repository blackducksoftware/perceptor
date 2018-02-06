package docker

import "fmt"

type ErrorType int

const (
	ErrorTypeUnableToCreateImage       ErrorType = iota
	ErrorTypeUnableToGetImage          ErrorType = iota
	ErrorTypeBadStatusCodeFromGetImage ErrorType = iota
	ErrorTypeUnableToCreateTarFile     ErrorType = iota
	ErrorTypeUnableToCopyTarFile       ErrorType = iota
	ErrorTypeUnableToGetFileStats      ErrorType = iota
)

func (et ErrorType) String() string {
	switch et {
	case ErrorTypeUnableToCreateImage:
		return "unable to create image in local docker"
	case ErrorTypeUnableToGetImage:
		return "unable to get image"
	case ErrorTypeBadStatusCodeFromGetImage:
		return "bad status code from GET image"
	case ErrorTypeUnableToCreateTarFile:
		return "Error opening file"
	case ErrorTypeUnableToCopyTarFile:
		return "Error copying file"
	case ErrorTypeUnableToGetFileStats:
		return "Error getting file stats"
	}
	panic(fmt.Errorf("invalid ErrorType value: %d", et))
}

type ImagePullError struct {
	Code      ErrorType
	RootCause error
}

func (ipe *ImagePullError) String() string {
	return fmt.Sprintf("%s: %s", ipe.Code.String(), ipe.RootCause.Error())
}

func (ipe ImagePullError) Error() string {
	return ipe.String()
}
