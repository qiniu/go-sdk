package chooser

import (
	"context"
	"net"
)

type (
	// ChooseOptions 选择器的选项
	ChooseOptions struct {
		// IPs 是待选择的 IP 地址列表
		IPs []net.IP
		// Domain 是待使用的域名
		Domain string
	}

	// FeedbackOptions 反馈的选项
	FeedbackOptions struct {
		// IPs 是待反馈的 IP 地址列表
		IPs []net.IP
		// Domain 是待使用的域名
		Domain string
	}

	// Chooser 选择器接口
	Chooser interface {
		// Choose 从给定的 IP 地址列表中选择一批 IP 地址用于发送请求
		Choose(context.Context, *ChooseOptions) []net.IP

		// FeedbackGood 反馈一批 IP 地址请求成功
		FeedbackGood(context.Context, *FeedbackOptions)

		// FeedbackBad 反馈一批 IP 地址请求失败
		FeedbackBad(context.Context, *FeedbackOptions)
	}
)

type directChooser struct {
}

func NewDirectChooser() Chooser {
	return &directChooser{}
}

func (chooser *directChooser) Choose(_ context.Context, options *ChooseOptions) []net.IP {
	return options.IPs
}

func (chooser *directChooser) FeedbackGood(_ context.Context, _ *FeedbackOptions) {
	// do nothing
}

func (chooser *directChooser) FeedbackBad(_ context.Context, _ *FeedbackOptions) {
	// do nothing
}

func (options *ChooseOptions) makeSet(makeKey func(string, net.IP) string) map[string]struct{} {
	m := make(map[string]struct{}, len(options.IPs))
	for _, ip := range options.IPs {
		m[makeKey(options.Domain, ip)] = struct{}{}
	}
	return m
}
