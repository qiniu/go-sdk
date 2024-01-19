package chooser

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

type subnetChooser struct {
	blacklist      map[string]blacklistItem
	blacklistMutex sync.Mutex
	freezeDuration time.Duration
	shrinkInterval time.Duration
	shrunkAt       time.Time
}

// SubnetChooserOptions 子网选择器的选项
type SubnetChooserOptions IPChooserOptions

// NewSubnetChooser 创建子网选择器
func NewSubnetChooser(options *SubnetChooserOptions) Chooser {
	if options == nil {
		options = &SubnetChooserOptions{}
	}
	if options.FreezeDuration == 0 {
		options.FreezeDuration = 10 * time.Minute
	}
	if options.ShrinkInterval == 0 {
		options.ShrinkInterval = 10 * time.Minute
	}
	return &subnetChooser{
		blacklist:      make(map[string]blacklistItem),
		freezeDuration: options.FreezeDuration,
		shrinkInterval: options.ShrinkInterval,
		shrunkAt:       time.Now(),
	}
}
func (chooser *subnetChooser) Choose(_ context.Context, options *ChooseOptions) []net.IP {
	return chooser.isInBlacklistAndOneSubnet(options.Domain, options.IPs)
}

func (chooser *subnetChooser) FeedbackGood(_ context.Context, options *FeedbackOptions) {
	chooser.deleteFromBlacklist(options.Domain, options.IPs)
}

func (chooser *subnetChooser) FeedbackBad(_ context.Context, options *FeedbackOptions) {
	chooser.putIntoBlacklist(options.Domain, options.IPs)
}

func (chooser *subnetChooser) isInBlacklistAndOneSubnet(domain string, ips []net.IP) []net.IP {
	chooser.blacklistMutex.Lock()
	defer chooser.blacklistMutex.Unlock()

	var (
		chosenSubnet net.IP
		filtered     = make([]net.IP, 0, len(ips))
	)
	for _, ip := range ips {
		blocklistKey := chooser.makeBlocklistKey(domain, ip)
		if blacklistItem, ok := chooser.blacklist[blocklistKey]; ok {
			if time.Now().After(blacklistItem.expiredAt) {
				delete(chooser.blacklist, blocklistKey)
			} else {
				continue
			}
		}
		if len(filtered) == 0 {
			chosenSubnet = makeSubnet(ip)
		} else if !chosenSubnet.Equal(makeSubnet(ip)) {
			continue
		}
		filtered = append(filtered, ip)
	}

	go chooser.shrink()

	return filtered
}

func (chooser *subnetChooser) putIntoBlacklist(domain string, ips []net.IP) {
	chooser.blacklistMutex.Lock()
	defer chooser.blacklistMutex.Unlock()

	for _, ip := range ips {
		blocklistKey := chooser.makeBlocklistKey(domain, ip)
		chooser.blacklist[blocklistKey] = blacklistItem{expiredAt: time.Now().Add(chooser.freezeDuration)}
	}

	go chooser.shrink()
}

func (chooser *subnetChooser) deleteFromBlacklist(domain string, ips []net.IP) {
	chooser.blacklistMutex.Lock()
	defer chooser.blacklistMutex.Unlock()

	for _, ip := range ips {
		blocklistKey := chooser.makeBlocklistKey(domain, ip)
		delete(chooser.blacklist, blocklistKey)
	}

	go chooser.shrink()
}

func (chooser *subnetChooser) makeBlocklistKey(domain string, ip net.IP) string {
	return fmt.Sprintf("%s_%s", makeSubnet(ip), domain)
}

func (chooser *subnetChooser) shrink() {
	chooser.blacklistMutex.Lock()
	defer chooser.blacklistMutex.Unlock()

	if time.Now().After(chooser.shrunkAt.Add(chooser.shrinkInterval)) {
		shrinkKeys := make([]string, 0, len(chooser.blacklist))
		for key, blacklistItem := range chooser.blacklist {
			if time.Now().After(blacklistItem.expiredAt) {
				shrinkKeys = append(shrinkKeys, key)
			}
		}
		for _, key := range shrinkKeys {
			delete(chooser.blacklist, key)
		}
		chooser.shrunkAt = time.Now()
	}
}

func makeSubnet(ip net.IP) net.IP {
	if ipv4 := ip.To4(); ipv4 != nil {
		return ipv4.Mask(ipv4.DefaultMask())
	}
	return ip.Mask(net.CIDRMask(64, 128))
}
