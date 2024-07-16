package resumablerecorder

import "time"

type dummyResumableRecorder struct{}

// 创建假的可恢复记录仪
func NewDummyResumableRecorder() ResumableRecorder {
	return dummyResumableRecorder{}
}

func (dummyResumableRecorder) OpenForReading(*ResumableRecorderOpenArgs) ReadableResumableRecorderMedium {
	return nil
}

func (dummyResumableRecorder) OpenForAppending(*ResumableRecorderOpenArgs) WriteableResumableRecorderMedium {
	return nil
}

func (dummyResumableRecorder) OpenForCreatingNew(*ResumableRecorderOpenArgs) WriteableResumableRecorderMedium {
	return nil
}

func (dummyResumableRecorder) Delete(*ResumableRecorderOpenArgs) error {
	return nil
}

func (dummyResumableRecorder) ClearOutdated(createdBefore time.Duration) error {
	return nil
}
