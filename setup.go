package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/apache/iceberg-go"
	"github.com/apache/iceberg-go/catalog"
	"github.com/apache/iceberg-go/table"
	_ "github.com/apache/iceberg-go/catalog/rest"
	_ "github.com/apache/iceberg-go/io/gocloud"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func setupInfrastructure() error {
	ctx := context.Background()

	// 1. Create MinIO storage bucket
	fmt.Println("Creating MinIO bucket 'warehouse'...")
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("ap-southeast-4"),
		config.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(
				func(service, region string, options ...interface{}) (aws.Endpoint, error) {
					return aws.Endpoint{
						URL:               "http://minio:9000",
						HostnameImmutable: true,
					}, nil
				},
			),
		),
	)
	if err != nil {
		return err
	}

	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})
	_, err = s3Client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String("warehouse"),
	})
	if err != nil {
		fmt.Println(" -> Bucket might already exist (safe to ignore).")
	} else {
		fmt.Println(" -> Bucket created.")
	}

	fmt.Println("Connecting to Iceberg REST catalog...")
	cat, err := catalog.Load(ctx, "rest", iceberg.Properties{
		"uri": "http://catalog:8181",
		"warehouse": "s3://warehouse/",
		"s3.endpoint": "http://minio:9000",
		"s3.access-key-id": "minioadmin",
		"s3.secret-access-key": "minioadmin",
		"s3.region": "ap-southeast-4",
	})
	if err != nil {
		return fmt.Errorf("failed to load Iceberg catalog: %w", err)
	}

	namespace := table.Identifier{"logging_db"}
	err = cat.CreateNamespace(ctx, namespace, iceberg.Properties{})
	if err != nil {
		if errors.Is(err, catalog.ErrNamespaceAlreadyExists) {
			fmt.Println(" -> Namespace 'logging_db' already exists.")
		} else {
			return fmt.Errorf("failed to create namespace: %w", err)
		}
	} else {
		fmt.Println(" -> Namespace 'logging_db' created.")
	}

	schema := iceberg.NewSchema(
		1,
		iceberg.NestedField{ID: 1, Name: "ip_address", Type: iceberg.PrimitiveTypes.String, Required: true},
		iceberg.NestedField{ID: 2, Name: "timestamp", Type: iceberg.PrimitiveTypes.TimestampTz, Required: true},
		iceberg.NestedField{ID: 3, Name: "method", Type: iceberg.PrimitiveTypes.String, Required: true},
		iceberg.NestedField{ID: 4, Name: "path", Type: iceberg.PrimitiveTypes.String, Required: true},
	)

	ident := table.Identifier{"logging_db", "access_logs"}
	_, err = cat.CreateTable(ctx, ident, schema)
	if err != nil {
		if errors.Is(err, catalog.ErrTableAlreadyExists) {
			fmt.Println(" -> Table 'access_logs' already exists.")
		} else {
			return fmt.Errorf("failed to create table: %w", err)
		}
	} else {
		fmt.Println(" -> Table 'access_logs' created.")
	}

	return nil
}
