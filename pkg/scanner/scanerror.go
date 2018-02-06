package scanner

import (
	"fmt"
)

type ErrorType int

const (
	ErrorTypeUnableToPullDockerImage = iota
	ErrorTypeFailedToRunJavaScanner  = iota
)

func (et ErrorType) String() string {
	switch et {
	case ErrorTypeUnableToPullDockerImage:
		return "unable to pull docker image"
	case ErrorTypeFailedToRunJavaScanner:
		return "failed to run java scanner"
	}
	panic(fmt.Errorf("invalid ErrorType value: %d", et))
}

type ScanError struct {
	Code      ErrorType
	RootCause error
}

func (se *ScanError) String() string {
	return fmt.Sprintf("%s: %s", se.Code.String(), se.RootCause.Error())
}

func (se ScanError) Error() string {
	return se.String()
}
