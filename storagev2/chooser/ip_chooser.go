package chooser

import (
	"bytes"
	"container/heap"
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

type (
	blackItem struct {
		index     int
		domain    string
		ip        net.IP
		expiredAt time.Time
	}

	blackheap struct {
		m     map[string]*blackItem
		items []*blackItem
	}

	ipChooser struct {
		blackheap      blackheap
		blackheapMutex sync.Mutex
		freezeDuration time.Duration
	}

	// IPChooserConfig IP 选择器的选项
	IPChooserConfig struct {
		// FreezeDuration IP 冻结时长（默认：600s）
		FreezeDuration time.Duration
	}
)

// NewIPChooser 创建 IP 选择器
func NewIPChooser(options *IPChooserConfig) Chooser {
	if options == nil {
		options = &IPChooserConfig{}
	}
	freezeDuration := options.FreezeDuration
	if freezeDuration == 0 {
		freezeDuration = 10 * time.Minute
	}
	return &ipChooser{
		blackheap: blackheap{
			m:     make(map[string]*blackItem, 1024),
			items: make([]*blackItem, 0, 1024),
		},
		freezeDuration: freezeDuration,
	}
}

func (chooser *ipChooser) Choose(_ context.Context, ips []net.IP, options *ChooseOptions) []net.IP {
	if len(ips) == 0 {
		return nil
	}
	if options == nil {
		options = &ChooseOptions{}
	}

	chooser.blackheapMutex.Lock()
	defer chooser.blackheapMutex.Unlock()

	chosenIPs := make([]net.IP, 0, chooser.blackheap.Len())
	for _, ip := range ips {
		if item := chooser.blackheap.FindByDomainAndIp(options.Domain, ip); item == nil {
			chosenIPs = append(chosenIPs, ip)
		}
	}
	if len(chosenIPs) > 0 || options.FailFast {
		return chosenIPs
	}

	var chosenExpiredAt time.Time
	toFind := makeSet(ips, options.Domain, makeMapKey)
	backups := make([]*blackItem, 0, chooser.blackheap.Len())
	for chooser.blackheap.Len() > 0 {
		firstChosen := heap.Pop(&chooser.blackheap).(*blackItem)
		backups = append(backups, firstChosen)
		key := makeMapKey(firstChosen.domain, firstChosen.ip)
		if _, ok := toFind[key]; ok {
			chosenExpiredAt = firstChosen.expiredAt
			chosenIPs = append(chosenIPs, firstChosen.ip)
			delete(toFind, key)
			break
		}
	}
	if chosenExpiredAt.IsZero() {
		panic("chosenExpiredAt should not be empty")
	}
	for chooser.blackheap.Len() > 0 {
		item := heap.Pop(&chooser.blackheap).(*blackItem)
		backups = append(backups, item)
		if chosenExpiredAt.Equal(item.expiredAt) {
			key := makeMapKey(item.domain, item.ip)
			if _, ok := toFind[key]; ok {
				chosenIPs = append(chosenIPs, item.ip)
				delete(toFind, key)
			}
		} else {
			break
		}
	}
	for _, backup := range backups {
		if backup.expiredAt.After(time.Now()) {
			heap.Push(&chooser.blackheap, backup)
		}
	}
	return chosenIPs
}

func (chooser *ipChooser) FeedbackGood(_ context.Context, ips []net.IP, options *FeedbackOptions) {
	if len(ips) == 0 {
		return
	}
	if options == nil {
		options = &FeedbackOptions{}
	}

	chooser.blackheapMutex.Lock()
	defer chooser.blackheapMutex.Unlock()

	for _, ip := range ips {
		if item := chooser.blackheap.FindByDomainAndIp(options.Domain, ip); item != nil {
			heap.Remove(&chooser.blackheap, item.index)
		}
	}
}

func (chooser *ipChooser) FeedbackBad(_ context.Context, ips []net.IP, options *FeedbackOptions) {
	if len(ips) == 0 {
		return
	}
	if options == nil {
		options = &FeedbackOptions{}
	}

	chooser.blackheapMutex.Lock()
	defer chooser.blackheapMutex.Unlock()

	newExpiredAt := time.Now().Add(chooser.freezeDuration)
	for _, ip := range ips {
		if item := chooser.blackheap.FindByDomainAndIp(options.Domain, ip); item != nil {
			chooser.blackheap.items[item.index].expiredAt = newExpiredAt
			heap.Fix(&chooser.blackheap, item.index)
		} else {
			heap.Push(&chooser.blackheap, &blackItem{
				domain:    options.Domain,
				ip:        ip,
				expiredAt: newExpiredAt,
			})
		}
	}
}

func (h *blackheap) Len() int {
	return len(h.items)
}

func (h *blackheap) Less(i, j int) bool {
	return h.items[i].expiredAt.Before(h.items[j].expiredAt)
}

func (h *blackheap) Swap(i, j int) {
	if i == j {
		return
	}
	h.items[i].domain, h.items[j].domain = h.items[j].domain, h.items[i].domain
	h.items[i].ip, h.items[j].ip = h.items[j].ip, h.items[i].ip
	h.items[i].expiredAt, h.items[j].expiredAt = h.items[j].expiredAt, h.items[i].expiredAt
	h.m[makeMapKey(h.items[i].domain, h.items[i].ip)] = h.items[i]
	h.m[makeMapKey(h.items[j].domain, h.items[j].ip)] = h.items[j]
}

func (h *blackheap) Push(x interface{}) {
	item := x.(*blackItem)
	item.index = len(h.items)
	h.items = append(h.items, item)
	h.m[makeMapKey(item.domain, item.ip)] = item
}

func (h *blackheap) Pop() interface{} {
	n := len(h.items)
	last := h.items[n-1]
	h.items = h.items[0 : n-1]
	delete(h.m, makeMapKey(last.domain, last.ip))
	return last
}

func (h *blackheap) FindByDomainAndIp(domain string, ip net.IP) *blackItem {
	key := makeMapKey(domain, ip)
	if item, ok := h.m[key]; ok {
		if item.expiredAt.Before(time.Now()) {
			heap.Remove(h, item.index)
			return nil
		}
		return item
	}
	return nil
}

func (h *blackheap) String() string {
	var buf bytes.Buffer

	buf.WriteString("&blackheap{ items: [")
	for _, item := range h.items {
		buf.WriteString(item.String())
		buf.WriteString(", ")
	}
	buf.WriteString("], m: {")
	for k, v := range h.m {
		buf.WriteString(k)
		buf.WriteString(": ")
		buf.WriteString(v.String())
		buf.WriteString(", ")
	}
	buf.WriteString("}")

	return buf.String()
}

func (item *blackItem) String() string {
	var buf bytes.Buffer

	buf.WriteString("&blackItem{ index: ")
	buf.WriteString(fmt.Sprintf("%d", item.index))
	buf.WriteString(", domain: ")
	buf.WriteString(item.domain)
	buf.WriteString(", ip: ")
	buf.WriteString(item.ip.String())
	buf.WriteString(", expiredAt: ")
	buf.WriteString(item.expiredAt.String())
	buf.WriteString("}")

	return buf.String()
}

func makeMapKey(domain string, ip net.IP) string {
	return ip.String() + "|" + domain
}
