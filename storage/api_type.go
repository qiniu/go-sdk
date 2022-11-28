package storage

const (
	actionTypeNone           = 0
	actionTypeUploadForm     = uploadMethodForm
	actionTypeUploadResumeV1 = uploadMethodResumeV1
	actionTypeUploadResumeV2 = uploadMethodResumeV2
)

func isApisSupportAction(apis []string, actionType int) bool {
	if len(apis) == 0 || actionType == actionTypeNone {
		return false
	}

	actionApis := getActionApis(actionType)

	support := true
	for _, actionApi := range actionApis {

		contain := false
		for _, api := range apis {
			if api == actionApi {
				contain = true
				break
			}
		}

		// 只要一个 api 不包含，整个 action 不支持
		if !contain {
			support = false
			break
		}
	}

	return support
}

func getActionApis(actionType int) (apis []string) {
	switch actionType {
	case uploadMethodForm:
		apis = []string{"up.formupload"}
		break
	case uploadMethodResumeV1:
		apis = []string{"up.mkblk", "up.bput", "up.mkfile"}
		break
	case uploadMethodResumeV2:
		apis = []string{"up.initparts", "up.uploadpart", "up.completeparts"}
		break
	default:
		break
	}
	return
}
