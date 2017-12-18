package scanner

type ScanClientInterface interface {
	Scan(job ScanJob)
	GetFinishedJobs() []ScanJob
}

type ScanJob struct {
	ProjectName string
	ImageName   string
}

func NewScanJob(projectName string, imageName string) *ScanJob {
	return &ScanJob{
		ProjectName: projectName,
		ImageName:   imageName,
	}
}
