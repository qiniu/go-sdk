package uploader

import "github.com/qiniu/go-sdk/v7/storagev2/region"

type ReinitializeOptions struct {
	keepOriginal    bool
	regionsProvider region.RegionsProvider
}

func KeepOriginalRegions() *ReinitializeOptions {
	return &ReinitializeOptions{keepOriginal: true}
}

func RefreshRegions() *ReinitializeOptions {
	return &ReinitializeOptions{keepOriginal: false}
}

func SetRegions(regionsProvider region.RegionsProvider) *ReinitializeOptions {
	return &ReinitializeOptions{regionsProvider: regionsProvider}
}

func (ro *ReinitializeOptions) KeepOriginalRegions() bool {
	if ro.regionsProvider != nil {
		return false
	}
	return ro.keepOriginal
}

func (ro *ReinitializeOptions) RefreshRegions() bool {
	if ro.regionsProvider != nil {
		return false
	}
	return !ro.keepOriginal
}

func (ro *ReinitializeOptions) SetRegions() region.RegionsProvider {
	return ro.regionsProvider
}
