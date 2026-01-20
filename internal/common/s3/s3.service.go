package s3

import (
	"context"
	"fmt"
	"io"
	"time"

	envConfig "github.com/alibaba0010/postgres-api/internal/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"
)

type S3Service struct {
	s3Client *s3.Client
	bucket   string
}

// NewS3Service creates a new S3 service instance.
func NewS3Service(ctx context.Context) (*S3Service, error) {
	appCfg := envConfig.LoadConfig()

	// Load AWS config with static credentials
	cfg, err := config.LoadDefaultConfig(ctx,
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
			// Origin Access Control (OAC) usually doesn't require PathStyle if using CloudFront,
			// but we'll leave it or remove it based on regional S3 requirements.
			// Modern S3 usually works better without PathStyle.
			o.UsePathStyle = false
		}),
		bucket: appCfg.AWS_BUCKET_NAME,
	}, nil
}

// GenerateUploadURL generates a short-lived presigned URL for uploading a file directly to S3.
// This follows the AWS best practice of using presigned URLs for scalability and security.
func (s *S3Service) GenerateUploadURL(ctx context.Context, key, contentType string) (string, error) {
	ps := s3.NewPresignClient(s.s3Client)

	req, err := ps.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(5*time.Minute))

	if err != nil {
		return "", err
	}

	return req.URL, nil
}
// GetMenuImageKey generates the S3 key for a menu image according to the recommended structure:
// menus/images/menu/{userId}/avatar.webp (or other extensions)
func (s *S3Service) GetMenuImageKey(userId string, filename string) string {
	return fmt.Sprintf("menus/images/menu/%s/%s", userId, filename)
}

// GetVideoUploadKey generates the S3 key for a video upload according to the recommended structure:
// menus/videos/uploads/{uuidv7}.mp4
// Using UUID V7 for better database indexing and temporal ordering.
func (s *S3Service) GetVideoUploadKey(ext string) string {
	if ext == "" {
		ext = ".mp4"
	}
	id, err := uuid.NewV7()
	if err != nil {
		// Fallback to V4 if V7 fails for some reason
		return fmt.Sprintf("menus/videos/uploads/%s%s", uuid.New().String(), ext)
	}
	return fmt.Sprintf("menus/videos/uploads/%s%s", id.String(), ext)
}

// InitiateMultipartUpload starts a multipart upload and returns the UploadId
func (s *S3Service) InitiateMultipartUpload(ctx context.Context, key, contentType string) (string, error) {
	resp, err := s.s3Client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", err
	}
	return *resp.UploadId, nil
}

// GeneratePresignPartURL generates a presigned URL for a specific part of a multipart upload
func (s *S3Service) GetPresignedPartURL(ctx context.Context, key, uploadID string, partNumber int32) (string, error) {
	ps := s3.NewPresignClient(s.s3Client)
	req, err := ps.PresignUploadPart(ctx, &s3.UploadPartInput{
		Bucket:     aws.String(s.bucket),
		Key:        aws.String(key),
		UploadId:   aws.String(uploadID),
		PartNumber: aws.Int32(partNumber),
	}, s3.WithPresignExpires(15*time.Minute))
	if err != nil {
		return "", err
	}
	return req.URL, nil
}

// CompleteMultipartUpload completes a multipart upload by assembling the parts
func (s *S3Service) CompleteMultipartUpload(ctx context.Context, key, uploadID string, parts []types.CompletedPart) (string, error) {
	_, err := s.s3Client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(s.bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: parts,
		},
	})
	if err != nil {
		return "", err
	}
	return s.GetCloudFrontURL(key), nil
}

// DirectUpload uploads a file directly from an io.Reader to S3.
// Useful for smaller files or when client-side direct upload is not preferred.
func (s *S3Service) DirectUpload(ctx context.Context, key string, body io.Reader, contentType string) error {
	// Use a 10-minute timeout for the upload. We don't use the request context directly as the primary
	// because intermediate proxies (like Next.js rewrites) might time out the request context at 30s.
	uploadCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	_, err := s.s3Client.PutObject(uploadCtx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
	})
	return err
}





// AbortMultipartUpload aborts a multipart upload
func (s *S3Service) AbortMultipartUpload(ctx context.Context, key, uploadID string) error {
	_, err := s.s3Client.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
		Bucket:   aws.String(s.bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
	})
	return err
}


// GetCloudFrontURL returns the public URL for a given S3 key via CloudFront.
func (s *S3Service) GetCloudFrontURL(key string) string {
	appCfg := envConfig.LoadConfig()
	if appCfg.AWS_CLOUDFRONT_DOMAIN != "" {
		return fmt.Sprintf("https://%s/%s", appCfg.AWS_CLOUDFRONT_DOMAIN, key)
	}
	// Fallback to S3 URL if CloudFront is not configured
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucket, appCfg.AWS_REGION, key)
}

/*
Video Processing (Production Grade) - Implementation Guidance:
To implement the recommended stack:
1. Bucket Event: Configure S3 to send an event notification to AWS Lambda or SNS when an object is created in `menus/videos/uploads/`.
2. Lambda/MediaConvert:
   - Use AWS Elemental MediaConvert for managed transcoding (recommended for production).
   - Alternatively, use a Lambda function with an FFmpeg layer for smaller tasks.
3. Output Storage: Store the processed HLS/Dash output in a separate folder (e.g., `menus/videos/processed/`) and serve via CloudFront.
*/
