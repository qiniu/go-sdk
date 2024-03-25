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

func (chooser *shuffleChooser) Choose(ctx context.Context, ips []net.IP, options *ChooseOptions) []net.IP {
	chosen_ips := chooser.chooser.Choose(ctx, ips, options)
	rand.Shuffle(len(chosen_ips), func(i, j int) {
		chosen_ips[i], chosen_ips[j] = chosen_ips[j], chosen_ips[i]
	})
	return chosen_ips
}

func (chooser *shuffleChooser) FeedbackGood(ctx context.Context, ips []net.IP, options *FeedbackOptions) {
	chooser.chooser.FeedbackGood(ctx, ips, options)
}

func (chooser *shuffleChooser) FeedbackBad(ctx context.Context, ips []net.IP, options *FeedbackOptions) {
	chooser.chooser.FeedbackBad(ctx, ips, options)
}
