package chooser

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

type ipChooser struct {
	blacklist      map[string]blacklistItem
	blacklistMutex sync.Mutex
	freezeDuration time.Duration
	shrinkInterval time.Duration
	shrunkAt       time.Time
}

// IPChooserOptions IP 选择器的选项
type IPChooserOptions struct {
	FreezeDuration time.Duration
	ShrinkInterval time.Duration
}

// NewIPChooser 创建 IP 选择器
func NewIPChooser(options *IPChooserOptions) Chooser {
	if options == nil {
		options = &IPChooserOptions{}
	}
	if options.FreezeDuration == 0 {
		options.FreezeDuration = 10 * time.Minute
	}
	if options.ShrinkInterval == 0 {
		options.ShrinkInterval = 10 * time.Minute
	}
	return &ipChooser{
		blacklist:      make(map[string]blacklistItem),
		freezeDuration: options.FreezeDuration,
		shrinkInterval: options.ShrinkInterval,
		shrunkAt:       time.Now(),
	}
}

func (chooser *ipChooser) Choose(_ context.Context, options *ChooseOptions) []net.IP {
	return chooser.isInBlacklist(options.Domain, options.IPs)
}

func (chooser *ipChooser) FeedbackGood(_ context.Context, options *FeedbackOptions) {
	chooser.deleteFromBlacklist(options.Domain, options.IPs)
}

func (chooser *ipChooser) FeedbackBad(_ context.Context, options *FeedbackOptions) {
	chooser.putIntoBlacklist(options.Domain, options.IPs)
}

func (chooser *ipChooser) isInBlacklist(domain string, ips []net.IP) []net.IP {
	chooser.blacklistMutex.Lock()
	defer chooser.blacklistMutex.Unlock()

	filtered := make([]net.IP, 0, len(ips))

	for _, ip := range ips {
		blocklistKey := chooser.makeBlocklistKey(domain, ip)
		if blacklistItem, ok := chooser.blacklist[blocklistKey]; ok {
			if time.Now().After(blacklistItem.expiredAt) {
				delete(chooser.blacklist, blocklistKey)
				filtered = append(filtered, ip)
			}
		} else {
			filtered = append(filtered, ip)
		}
	}

	go chooser.shrink()

	return filtered
}

func (chooser *ipChooser) putIntoBlacklist(domain string, ips []net.IP) {
	chooser.blacklistMutex.Lock()
	defer chooser.blacklistMutex.Unlock()

	for _, ip := range ips {
		blocklistKey := chooser.makeBlocklistKey(domain, ip)
		chooser.blacklist[blocklistKey] = blacklistItem{expiredAt: time.Now().Add(chooser.freezeDuration)}
	}

	go chooser.shrink()
}

func (chooser *ipChooser) deleteFromBlacklist(domain string, ips []net.IP) {
	chooser.blacklistMutex.Lock()
	defer chooser.blacklistMutex.Unlock()

	for _, ip := range ips {
		blocklistKey := chooser.makeBlocklistKey(domain, ip)
		delete(chooser.blacklist, blocklistKey)
	}

	go chooser.shrink()
}

func (chooser *ipChooser) makeBlocklistKey(domain string, ip net.IP) string {
	return fmt.Sprintf("%s_%s", ip, domain)
}

func (chooser *ipChooser) shrink() {
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
