package scanner

import (
	"fmt"
)

type ErrorType int

const (
	ErrorTypeUnableToPullDockerImage = iota
	ErrorTypeFailedToRunJavaScanner  = iota
)

type ScanError struct {
	Code      ErrorType
	RootCause error
}

func (se ScanError) Error() string {
	switch se.Code {
	case ErrorTypeUnableToPullDockerImage:
		return fmt.Sprintf("unable to pull docker image: %s", se.RootCause.Error())
	case ErrorTypeFailedToRunJavaScanner:
		return fmt.Sprintf("failed to run java scanner: %s", se.RootCause.Error())
	}
	panic(fmt.Errorf("invalid ErrorType value: %d", se.Code))
}
