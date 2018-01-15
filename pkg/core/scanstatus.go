package core

type ScanStatus int

const (
	ScanStatusNotScanned        ScanStatus = iota
	ScanStatusInQueue           ScanStatus = iota
	ScanStatusRunningScanClient ScanStatus = iota
	ScanStatusRunningHubScan    ScanStatus = iota
	ScanStatusComplete          ScanStatus = iota
	ScanStatusError             ScanStatus = iota
)
