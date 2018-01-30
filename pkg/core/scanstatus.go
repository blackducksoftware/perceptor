package core

import "fmt"

// ScanStatus describes the state of an image -- have we checked the hub for it?
// Have we scanned it?  Are we scanning it?
type ScanStatus int

// Allowed transitions:
//  - Unknown -> InHubCheckQueue
//  - InHubCheckQueue -> CheckingHub
//  - CheckingHub -> InQueue
//  - CheckingHub -> Complete
//  - InQueue -> RunningScanClient
//  - RunningScanClient -> Error
//  - RunningScanClient -> RunningHubScan
//  - RunningHubScan -> Error
//  - RunningHubScan -> Complete
//  - Error -> ??? throw it back into the queue?
const (
	ScanStatusUnknown           ScanStatus = iota
	ScanStatusInHubCheckQueue   ScanStatus = iota
	ScanStatusCheckingHub       ScanStatus = iota
	ScanStatusInQueue           ScanStatus = iota
	ScanStatusRunningScanClient ScanStatus = iota
	ScanStatusRunningHubScan    ScanStatus = iota
	ScanStatusComplete          ScanStatus = iota
	ScanStatusError             ScanStatus = iota
)

func (status ScanStatus) String() string {
	switch status {
	case ScanStatusUnknown:
		return "ScanStatusUnknown"
	case ScanStatusInHubCheckQueue:
		return "ScanStatusInHubCheckQueue"
	case ScanStatusCheckingHub:
		return "ScanStatusCheckingHub"
	case ScanStatusInQueue:
		return "ScanStatusInQueue"
	case ScanStatusRunningScanClient:
		return "ScanStatusRunningScanClient"
	case ScanStatusRunningHubScan:
		return "ScanStatusRunningHubScan"
	case ScanStatusComplete:
		return "ScanStatusComplete"
	case ScanStatusError:
		return "ScanStatusError"
	}
	panic(fmt.Errorf("invalid ScanStatus value: %d", status))
}
