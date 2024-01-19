package chooser

import (
	"context"
	"net"
)

type (
	smartIPChooser struct {
		ipChooser, subnetChooser Chooser
	}

	// SmartIPChooserOptions 智能 IP 选择器的选项
	SmartIPChooserOptions IPChooserOptions
)

// NewSmartIPChooser 创建智能 IP 选择器
func NewSmartIPChooser(options *SmartIPChooserOptions) Chooser {
	return &smartIPChooser{
		ipChooser:     NewIPChooser((*IPChooserOptions)(options)),
		subnetChooser: NewSubnetChooser((*SubnetChooserOptions)(options)),
	}
}

func (chooser *smartIPChooser) Choose(ctx context.Context, options *ChooseOptions) []net.IP {
	if chooser.allInSingleSubnet(options.IPs) {
		return chooser.ipChooser.Choose(ctx, options)
	} else {
		return chooser.subnetChooser.Choose(ctx, options)
	}
}

func (chooser *smartIPChooser) FeedbackGood(ctx context.Context, options *FeedbackOptions) {
	chooser.ipChooser.FeedbackGood(ctx, options)
	chooser.subnetChooser.FeedbackGood(ctx, options)
}

func (chooser *smartIPChooser) FeedbackBad(ctx context.Context, options *FeedbackOptions) {
	chooser.ipChooser.FeedbackBad(ctx, options)
	chooser.subnetChooser.FeedbackBad(ctx, options)
}

func (chooser *smartIPChooser) allInSingleSubnet(ips []net.IP) bool {
	var subnet net.IP
	for i, ip := range ips {
		if i == 0 {
			subnet = makeSubnet(ip)
		} else if !subnet.Equal(makeSubnet(ip)) {
			return false
		}
	}
	return true
}
