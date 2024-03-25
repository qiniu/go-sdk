package chooser

import (
	"context"
	"net"
)

type (
	smartIPChooser struct {
		ipChooser, subnetChooser Chooser
	}

	// SmartIPChooserConfig 智能 IP 选择器的选项
	SmartIPChooserConfig IPChooserConfig
)

// NewSmartIPChooser 创建智能 IP 选择器
func NewSmartIPChooser(options *SmartIPChooserConfig) Chooser {
	return &smartIPChooser{
		ipChooser:     NewIPChooser((*IPChooserConfig)(options)),
		subnetChooser: NewSubnetChooser((*SubnetChooserConfig)(options)),
	}
}

func (chooser *smartIPChooser) Choose(ctx context.Context, ips []net.IP, options *ChooseOptions) []net.IP {
	if chooser.allInSingleSubnet(ips) {
		return chooser.ipChooser.Choose(ctx, ips, options)
	} else {
		return chooser.subnetChooser.Choose(ctx, ips, options)
	}
}

func (chooser *smartIPChooser) FeedbackGood(ctx context.Context, ips []net.IP, options *FeedbackOptions) {
	chooser.ipChooser.FeedbackGood(ctx, ips, options)
	chooser.subnetChooser.FeedbackGood(ctx, ips, options)
}

func (chooser *smartIPChooser) FeedbackBad(ctx context.Context, ips []net.IP, options *FeedbackOptions) {
	chooser.ipChooser.FeedbackBad(ctx, ips, options)
	chooser.subnetChooser.FeedbackBad(ctx, ips, options)
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
