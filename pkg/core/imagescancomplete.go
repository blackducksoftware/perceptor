package core

// TODO may want to get rid of this ... is it really necessary?
type ImageScanComplete struct {
	AffectedPods []Pod
	Image        string // TODO or common.Image ?
	ScanResults  ScanResults
}
