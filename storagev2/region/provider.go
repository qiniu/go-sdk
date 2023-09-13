package region

import "context"

// 区域提供者
type RegionsProvider interface {
	GetRegions(context.Context) ([]*Region, error)
}
