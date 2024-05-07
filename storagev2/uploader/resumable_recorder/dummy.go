package resumablerecorder

type dummyResumableRecorder struct{}

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
