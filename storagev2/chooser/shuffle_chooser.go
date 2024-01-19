package chooser

import (
	"context"
	"math/rand"
	"net"
)

type shuffleChooser struct {
	chooser Chooser
}

// NewShuffleChooser 创建随机混淆选择器
func NewShuffleChooser(chooser Chooser) Chooser {
	return &shuffleChooser{chooser: chooser}
}

func (chooser *shuffleChooser) Choose(ctx context.Context, options *ChooseOptions) []net.IP {
	ips := chooser.chooser.Choose(ctx, options)
	rand.Shuffle(len(ips), func(i, j int) {
		ips[i], ips[j] = ips[j], ips[i]
	})
	return ips
}

func (chooser *shuffleChooser) FeedbackGood(ctx context.Context, options *FeedbackOptions) {
	chooser.chooser.FeedbackGood(ctx, options)
}

func (chooser *shuffleChooser) FeedbackBad(ctx context.Context, options *FeedbackOptions) {
	chooser.chooser.FeedbackBad(ctx, options)
}
