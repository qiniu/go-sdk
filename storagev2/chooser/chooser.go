package chooser

import (
	"context"
	"net"
)

type (
	// ChooseOptions 选择器的选项
	ChooseOptions struct {
		// Domain 是待选择的域名
		Domain string

		// 如果找不到合适的域名就直接返回空
		FailFast bool
	}

	// FeedbackOptions 反馈的选项
	FeedbackOptions struct {
		// Domain 是待反馈的域名
		Domain string
	}

	// Chooser 选择器接口
	Chooser interface {
		// Choose 从给定的 IP 地址列表中选择一批 IP 地址用于发送请求
		Choose(context.Context, []net.IP, *ChooseOptions) []net.IP

		// FeedbackGood 反馈一批 IP 地址请求成功
		FeedbackGood(context.Context, []net.IP, *FeedbackOptions)

		// FeedbackBad 反馈一批 IP 地址请求失败
		FeedbackBad(context.Context, []net.IP, *FeedbackOptions)
	}

	directChooser struct{}
)

// NewDirectChooser 创建直接选择器
func NewDirectChooser() Chooser {
	return &directChooser{}
}

func (chooser *directChooser) Choose(_ context.Context, ips []net.IP, _ *ChooseOptions) []net.IP {
	return ips
}

func (chooser *directChooser) FeedbackGood(_ context.Context, _ []net.IP, _ *FeedbackOptions) {
	// do nothing
}

func (chooser *directChooser) FeedbackBad(_ context.Context, _ []net.IP, _ *FeedbackOptions) {
	// do nothing
}

func makeSet(ips []net.IP, domain string, makeKey func(string, net.IP) string) map[string]struct{} {
	m := make(map[string]struct{}, len(ips))
	for _, ip := range ips {
		m[makeKey(domain, ip)] = struct{}{}
	}
	return m
}
