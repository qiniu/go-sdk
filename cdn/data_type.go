package cdn

//go:generate stringer -type DataType -linecomment
type DataType int

const (
	// 静态cdn带宽(bps)
	DataTypeBandwidth DataType = iota + 1 // bandwidth
	// 302cdn带宽(bps)
	DataType302Bandwidth // 302bandwidth
	// 302MIX带宽(bps)
	DataType302mBandwidth // 302mbandwidth
	// 静态cdn流量(bytes)
	DataTypeFlow // flow
	// 302cdn流量(bytes)
	DataType302Flow // 302flow
	// 302MIX流量(bytes)
	DataType302mFlow // 302mflow
)

func DataTypeOf(datatype string) DataType {
	for d := DataTypeBandwidth; d <= DataType302mFlow; d++ {
		if d.String() == datatype {
			return d
		}
	}
	return -1
}

func (d DataType) Valid() bool {
	return DataTypeBandwidth <= d && d <= DataType302mFlow
}
