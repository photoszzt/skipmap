module github.com/zhangyunhao116/skipmap/benchmark

go 1.18

require (
	github.com/google/btree v1.1.2
	github.com/orcaman/concurrent-map/v2 v2.0.0
	github.com/zhangyunhao116/fastrand v0.3.0
	github.com/zhangyunhao116/skipmap v0.0.0
)

require (
	github.com/SaveTheRbtz/generic-sync-map-go v0.0.0-20220414055132-a37292614db8 // indirect
	github.com/cespare/xxhash v1.1.0 // indirect
	github.com/cornelk/hashmap v1.0.4 // indirect
)

replace github.com/zhangyunhao116/skipmap v0.0.0 => github.com/photoszzt/skipmap v0.0.0-20220818193111-0136ec8de741
