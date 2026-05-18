package main

import (
	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"time"
)

func main() {
	mem := memory.NewGoAllocator()
	schema := arrow.NewSchema(
		[]arrow.Field{
			{Name: "ip_address", Type: arrow.BinaryTypes.String, Nullable: false},
			{Name: "timestamp", Type: arrow.FixedWidthTypes.Timestamp_us, Nullable: false},
			{Name: "method", Type: arrow.BinaryTypes.String, Nullable: false},
			{Name: "path", Type: arrow.BinaryTypes.String, Nullable: false},
		},
		nil,
	)
	
	b := array.NewRecordBuilder(mem, schema)
	defer b.Release()

	ipBuilder := b.Field(0).(*array.StringBuilder)
	tsBuilder := b.Field(1).(*array.TimestampBuilder)
	
	ipBuilder.Append("127.0.0.1")
	tsBuilder.Append(arrow.Timestamp(time.Now().UnixMicro()))

	rec := b.NewRecord()
	defer rec.Release()
	
	arrowTbl := array.NewTableFromRecords(schema, []arrow.Record{rec})
	defer arrowTbl.Release()
}
