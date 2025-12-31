package fixs3cors
package main

import (
	"context"
	"log"

	appConfig "github.com/alibaba0010/postgres-api/internal/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env from root if possible
	_ = godotenv.Load()

	cfg := appConfig.LoadConfig()

	log.Printf("Configuring S3 CORS for bucket: %s in region: %s", cfg.AWS_BUCKET_NAME, cfg.AWS_REGION)

	// Load AWS config
	awsCfg, err := awsConfig.LoadDefaultConfig(context.TODO(),
		awsConfig.WithRegion(cfg.AWS_REGION),
		awsConfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AWS_ACCESS_KEY_ID,
			cfg.AWS_SECRET_ACCESS_KEY,
			"",
		)),
	)
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	client := s3.NewFromConfig(awsCfg)

	input := &s3.PutBucketCorsInput{
		Bucket: aws.String(cfg.AWS_BUCKET_NAME),
		CORSConfiguration: &types.CORSConfiguration{
			CORSRules: []types.CORSRule{
				{
					AllowedHeaders: []string{"*"},
					AllowedMethods: []string{"PUT", "POST", "GET", "HEAD", "DELETE"},
					AllowedOrigins: []string{"http://localhost:3000", "http://localhost:8000", "http://localhost:8001", "*"},
					ExposeHeaders:  []string{"ETag", "x-amz-server-side-encryption", "x-amz-request-id", "x-amz-id-2"},
					MaxAgeSeconds:  aws.Int32(3000),
				},
			},
		},
	}

	_, err = client.PutBucketCors(context.TODO(), input)
	if err != nil {
		log.Fatalf("Failed to set CORS: %v", err)
	}

	log.Println("Successfully updated S3 CORS configuration! You should now be able to upload.")
}
