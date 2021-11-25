package jobs

type Job struct {
	Spec	string
	Task	func()
}
