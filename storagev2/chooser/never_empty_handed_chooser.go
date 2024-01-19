package chooser

import (
	"context"
	"math/rand"
	"net"
	"time"

	"github.com/alex-ant/gomath/rational"
)

type neverEmptyHandedChooser struct {
	chooser           Chooser
	randomChooseRatio rational.Rational
}

func NewNeverEmptyHandedChooser(chooser Chooser, randomChooseRatio rational.Rational) Chooser {
	return &neverEmptyHandedChooser{chooser: chooser, randomChooseRatio: randomChooseRatio}
}

func (chooser *neverEmptyHandedChooser) Choose(ctx context.Context, options *ChooseOptions) []net.IP {
	ips := chooser.chooser.Choose(ctx, options)
	if len(ips) == 0 {
		return chooser.chooseMultiple(append(make([]net.IP, 0, len(options.IPs)), options.IPs...), chooser.randomChooseRatio)
	}
	return ips
}

func (chooser *neverEmptyHandedChooser) chooseMultiple(ips []net.IP, ratio rational.Rational) []net.IP {
	value := chooser.randomChooseRatio
	x, y := value.MultiplyByNum(int64(len(ips))).Get()
	willChoose := (x + y - 1) / y
	chosenIPs := make([]net.IP, 0, willChoose)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := int64(0); i < willChoose; i++ {
		chosenIndex := r.Intn(len(ips))
		chosenIPs = append(chosenIPs, ips[chosenIndex])
		ips[chosenIndex] = ips[len(ips)-1]
		ips = ips[:len(ips)-1]
	}
	return chosenIPs
}

func (chooser *neverEmptyHandedChooser) FeedbackGood(ctx context.Context, options *FeedbackOptions) {
	chooser.chooser.FeedbackGood(ctx, options)
}

func (chooser *neverEmptyHandedChooser) FeedbackBad(ctx context.Context, options *FeedbackOptions) {
	chooser.chooser.FeedbackBad(ctx, options)
}
