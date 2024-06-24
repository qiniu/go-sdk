package resumablerecorder

type dummyResumableRecorder struct{}

// 创建假的可恢复记录仪
func NewDummyResumableRecorder() ResumableRecorder {
	return dummyResumableRecorder{}
}

func (dummyResumableRecorder) OpenForReading(*ResumableRecorderOpenOptions) ReadableResumableRecorderMedium {
	return nil
}

func (dummyResumableRecorder) OpenForAppending(*ResumableRecorderOpenOptions) WriteableResumableRecorderMedium {
	return nil
}

func (dummyResumableRecorder) OpenForCreatingNew(*ResumableRecorderOpenOptions) WriteableResumableRecorderMedium {
	return nil
}

func (dummyResumableRecorder) Delete(*ResumableRecorderOpenOptions) error {
	return nil
}

func (dummyResumableRecorder) ClearExpired() error {
	return nil
}
