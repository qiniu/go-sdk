package chooser

import (
	"container/heap"
	"context"
	"net"
	"sync"
	"time"
)

type (
	subnetChooser struct {
		blackheap      blackheap
		blackheapMutex sync.Mutex
		freezeDuration time.Duration
	}

	// SubnetChooserConfig 子网选择器的选项
	SubnetChooserConfig IPChooserConfig
)

// NewSubnetChooser 创建子网选择器
func NewSubnetChooser(options *SubnetChooserConfig) Chooser {
	if options == nil {
		options = &SubnetChooserConfig{}
	}
	freezeDuration := options.FreezeDuration
	if freezeDuration == 0 {
		freezeDuration = 10 * time.Minute
	}
	return &subnetChooser{
		blackheap: blackheap{
			m:     make(map[string]*blackItem, 1024),
			items: make([]*blackItem, 0, 1024),
		},
		freezeDuration: freezeDuration,
	}
}

func (chooser *subnetChooser) Choose(_ context.Context, ips []net.IP, options *ChooseOptions) []net.IP {
	if len(ips) == 0 {
		return nil
	}
	if options == nil {
		options = &ChooseOptions{}
	}

	chooser.blackheapMutex.Lock()
	defer chooser.blackheapMutex.Unlock()

	var chosenSubnet net.IP
	chosenIPs := make([]net.IP, 0, chooser.blackheap.Len())
	for _, ip := range ips {
		subnetIP := makeSubnet(ip)
		if len(chosenIPs) == 0 {
			if item := chooser.blackheap.FindByDomainAndIp(options.Domain, subnetIP); item == nil {
				chosenIPs = append(chosenIPs, ip)
				chosenSubnet = subnetIP
			}
		} else if chosenSubnet.Equal(subnetIP) {
			chosenIPs = append(chosenIPs, ip)
		}
	}
	if len(chosenIPs) > 0 || options.FailFast {
		return chosenIPs
	}

	var chosenExpiredAt time.Time
	toFind := makeSet(ips, options.Domain, func(domain string, ip net.IP) string {
		return makeMapKey(domain, makeSubnet(ip))
	})
	backups := make([]*blackItem, 0, chooser.blackheap.Len())
	for chooser.blackheap.Len() > 0 {
		firstChosen := heap.Pop(&chooser.blackheap).(*blackItem)
		backups = append(backups, firstChosen)
		firstChosenSubnetIP := makeSubnet(firstChosen.ip)
		if _, ok := toFind[makeMapKey(firstChosen.domain, firstChosenSubnetIP)]; ok {
			chosenExpiredAt = firstChosen.expiredAt
			chosenSubnet = firstChosenSubnetIP
			break
		}
	}
	if chosenExpiredAt.IsZero() {
		panic("chosenExpiredAt should not be empty")
	}
	for _, ip := range ips {
		if chosenSubnet.Equal(makeSubnet(ip)) {
			chosenIPs = append(chosenIPs, ip)
		}
	}
	for _, backup := range backups {
		if backup.expiredAt.After(time.Now()) {
			heap.Push(&chooser.blackheap, backup)
		}
	}
	return chosenIPs
}

func (chooser *subnetChooser) FeedbackGood(_ context.Context, ips []net.IP, options *FeedbackOptions) {
	if len(ips) == 0 {
		return
	}
	if options == nil {
		options = &FeedbackOptions{}
	}

	chooser.blackheapMutex.Lock()
	defer chooser.blackheapMutex.Unlock()

	haveFeedback := make(map[string]struct{})
	for _, ip := range ips {
		subnetIP := makeSubnet(ip)
		subnetIPString := subnetIP.String()
		if _, ok := haveFeedback[subnetIPString]; ok {
			continue
		} else {
			haveFeedback[subnetIPString] = struct{}{}
		}
		if item := chooser.blackheap.FindByDomainAndIp(options.Domain, subnetIP); item != nil {
			heap.Remove(&chooser.blackheap, item.index)
		}
	}
}

func (chooser *subnetChooser) FeedbackBad(_ context.Context, ips []net.IP, options *FeedbackOptions) {
	if len(ips) == 0 {
		return
	}
	if options == nil {
		options = &FeedbackOptions{}
	}

	chooser.blackheapMutex.Lock()
	defer chooser.blackheapMutex.Unlock()

	haveFeedback := make(map[string]struct{})
	newExpiredAt := time.Now().Add(chooser.freezeDuration)
	for _, ip := range ips {
		subnetIP := makeSubnet(ip)
		subnetIPString := subnetIP.String()
		if _, ok := haveFeedback[subnetIPString]; ok {
			continue
		} else {
			haveFeedback[subnetIPString] = struct{}{}
		}
		if item := chooser.blackheap.FindByDomainAndIp(options.Domain, subnetIP); item != nil {
			if chooser.blackheap.items[item.index].expiredAt.Equal(newExpiredAt) {
				continue
			}
			chooser.blackheap.items[item.index].expiredAt = newExpiredAt
			heap.Fix(&chooser.blackheap, item.index)
		} else {
			heap.Push(&chooser.blackheap, &blackItem{
				domain:    options.Domain,
				ip:        subnetIP,
				expiredAt: newExpiredAt,
			})
		}
	}
}

func makeSubnet(ip net.IP) net.IP {
	if ipv4 := ip.To4(); ipv4 != nil {
		return ipv4.Mask(ipv4.DefaultMask())
	}
	return ip.Mask(net.CIDRMask(64, 128))
}
