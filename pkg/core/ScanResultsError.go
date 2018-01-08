package core

type ScanResultsError struct {
	message string
}

func (sre *ScanResultsError) Error() string {
	return sre.message
}
