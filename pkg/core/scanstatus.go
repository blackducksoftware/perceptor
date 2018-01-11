package core

type ScanStatus int

const (
	ScanStatusNotScanned ScanStatus = iota
	ScanStatusInProgress ScanStatus = iota
	ScanStatusComplete   ScanStatus = iota
	ScanStatusError      ScanStatus = iota
)
