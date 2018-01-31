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

type ImagePullError struct {
	Code      ErrorType
	RootCause error
}

func (ipe *ImagePullError) String() string {
	switch ipe.Code {
	case ErrorTypeUnableToCreateImage:
		return fmt.Sprintf("unable to create image in local docker: %s", ipe.RootCause.Error())
	case ErrorTypeUnableToGetImage:
		return fmt.Sprintf("unable to get image: %s", ipe.RootCause.Error())
	case ErrorTypeBadStatusCodeFromGetImage:
		return fmt.Sprintf("bad status code from GET image: %s", ipe.RootCause.Error())
	case ErrorTypeUnableToCreateTarFile:
		return fmt.Sprintf("Error opening file: %s", ipe.RootCause.Error())
	case ErrorTypeUnableToCopyTarFile:
		return fmt.Sprintf("Error copying file: %s", ipe.RootCause.Error())
	case ErrorTypeUnableToGetFileStats:
		return fmt.Sprintf("Error getting file stats: %s", ipe.RootCause.Error())
	}
	panic(fmt.Errorf("invalid ErrorType value: %d", ipe.Code))
}

func (ipe ImagePullError) Error() string {
	return ipe.String()
}
