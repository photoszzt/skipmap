module github.com/zhangyunhao116/skipmap/benchmark

go 1.18

require (
	github.com/cornelk/hashmap v1.0.4
	github.com/google/btree v1.1.2
	github.com/orcaman/concurrent-map/v2 v2.0.0
	github.com/zhangyunhao116/fastrand v0.3.0
	github.com/zhangyunhao116/skipmap v0.0.0
)

require github.com/cespare/xxhash v1.1.0 // indirect

replace github.com/zhangyunhao116/skipmap v0.0.0 => ../
