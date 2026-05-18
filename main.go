package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/apache/iceberg-go"
	"github.com/apache/iceberg-go/catalog"
	"github.com/apache/iceberg-go/table"
	_ "github.com/apache/iceberg-go/catalog/rest"
	_ "github.com/apache/iceberg-go/io/gocloud"
)

func main() {
	// Control Plane: setup phase
	fmt.Println("--- STARTING BUILD UP STAGE ---")
	if err := setupInfrastructure(); err != nil {
		log.Fatalf("Setup failed: %v", err)
	}

	// Data Plane: run the execution phase
	fmt.Println("--- STARTING EXECUTION PHASE ---")
	ctx := context.Background()

	cat, err := catalog.Load(ctx, "rest", iceberg.Properties{
		"uri": "http://catalog:8181",
		"warehouse": "s3://warehouse/",
		"s3.endpoint": "http://minio:9000",
		"s3.access-key-id": "minioadmin",
		"s3.secret-access-key": "minioadmin",
		"s3.region": "ap-southeast-4",
	})
	if err != nil {
		log.Fatalf("Failed to connect to Iceberg REST catalog: %v", err)
	}

	ident := table.Identifier{"logging_db", "access_logs"}
	tbl, err := cat.LoadTable(ctx, ident)
	if err != nil {
		log.Fatalf("Failed to load table: %v", err)
	}

	fmt.Printf("Successfully loaded table: %s\n", strings.Join(tbl.Identifier(), "."))
	fmt.Printf("Current schema: %s\n", tbl.Schema().String())

	fmt.Println("Generating fake access logs...")
	logs := generateFakeAccessLogs(10)

	fmt.Printf("Generated %d fake logs\n", len(logs))

	mem := memory.NewGoAllocator()
	schema := arrow.NewSchema(
		[]arrow.Field{
			{Name: "ip_address", Type: arrow.BinaryTypes.String, Nullable: false},
			{Name: "timestamp", Type: &arrow.TimestampType{Unit: arrow.Microsecond, TimeZone: "UTC"}, Nullable: false},
			{Name: "method", Type: arrow.BinaryTypes.String, Nullable: false},
			{Name: "path", Type: arrow.BinaryTypes.String, Nullable: false},
		},
		nil,
	)

	b := array.NewRecordBuilder(mem, schema)
	defer b.Release()

	ipBuilder := b.Field(0).(*array.StringBuilder)
	tsBuilder := b.Field(1).(*array.TimestampBuilder)
	methodBuilder := b.Field(2).(*array.StringBuilder)
	pathBuilder := b.Field(3).(*array.StringBuilder)

	for _, l := range logs {
		ipBuilder.Append(l.IPAddress)
		tsBuilder.Append(arrow.Timestamp(l.Timestamp.UnixMicro()))
		methodBuilder.Append(l.Method)
		pathBuilder.Append(l.Path)
	}

	rec := b.NewRecord()
	defer rec.Release()

	arrowTbl := array.NewTableFromRecords(schema, []arrow.Record{rec})
	defer arrowTbl.Release()

	fmt.Println("Appending data to Iceberg table...")
	_, err = tbl.AppendTable(ctx, arrowTbl, 0, nil)
	if err != nil {
		log.Fatalf("Failed to append data: %v", err)
	}

	fmt.Println("Data appended successfully!")
	fmt.Println("Sandbox test completed successfully.")
}

// Simple struct to represent our data
type AccessLog struct {
	IPAddress string
	Timestamp time.Time
	Method    string
	Path      string
}

func generateFakeAccessLogs(count int) []AccessLog {
	var logs []AccessLog
	now := time.Now()
	for i := 0; i < count; i++ {
		logs = append(logs, AccessLog{
			IPAddress: fmt.Sprintf("192.168.1.%d", i%255),
			Timestamp: now.Add(time.Duration(i) * time.Second),
			Method:    "GET",
			Path:      "/api/v1/data",
		})
	}
	return logs
}
