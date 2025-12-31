package s3

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"time"

	envConfig "github.com/alibaba0010/postgres-api/internal/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	// "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"
)

type S3Service struct {
	s3Client *s3.Client
	bucket   string
}

func NewS3Service() (*S3Service, error) {
	appCfg := envConfig.LoadConfig()

	// Load AWS config with static credentials
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(appCfg.AWS_REGION),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			appCfg.AWS_ACCESS_KEY_ID,
			appCfg.AWS_SECRET_ACCESS_KEY,
			"",
		)),
	)
	if err != nil {
		return nil, err
	}

	return &S3Service{
		s3Client: s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.UsePathStyle = true
		}),
		bucket: appCfg.AWS_BUCKET_NAME,
	}, nil
}

func (s *S3Service) UploadFile(file multipart.File, fileHeader *multipart.FileHeader, folder string) (string, error) {
	size := fileHeader.Size
	buffer := make([]byte, size)
	_, err := file.Read(buffer)
	if err != nil {
		return "", err
	}

	// Reset file pointer
	file.Seek(0, 0)

	// Create a unique file name
	ext := filepath.Ext(fileHeader.Filename)
	filename := fmt.Sprintf("%s/%s_%s%s", folder, time.Now().Format("20060102150405"), uuid.New().String(), ext)

	// Detect content type
	contentType := http.DetectContentType(buffer)

	_, err = s.s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(filename),
		Body:          bytes.NewReader(buffer),
		ContentLength: aws.Int64(size),
		ContentType:   aws.String(contentType),
		// ACL:           types.ObjectCannedACLPublicRead,
	})

	if err != nil {
		return "", err
	}

	// Construct public URL
	appCfg := envConfig.LoadConfig()
	url := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucket, appCfg.AWS_REGION, filename)
	return url, nil
}
