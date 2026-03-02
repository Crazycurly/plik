package s3

import (
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/root-gg/utils"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/data"
)

// Ensure S3 Data Backend implements data.Backend interface
var _ data.Backend = (*Backend)(nil)

// Config describes configuration for S3 data backend
type Config struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	Location        string
	Prefix          string
	PartSize        uint64
	UseSSL          bool
	SendContentMd5  bool
	SSE             string
}

// NewConfig instantiate a new default configuration
// and override it with configuration passed as argument
func NewConfig(params map[string]any) (config *Config) {
	config = new(Config)
	config.Bucket = "plik"
	config.Location = "us-east-1"
	config.PartSize = 16 * 1024 * 1024 // 16MiB
	utils.Assign(config, params)
	return
}

// Validate check config parameters
func (config *Config) Validate() error {
	if config.Endpoint == "" {
		return fmt.Errorf("missing endpoint")
	}
	if config.AccessKeyID == "" {
		return fmt.Errorf("missing access key ID")
	}
	if config.SecretAccessKey == "" {
		return fmt.Errorf("missing secret access key")
	}
	if config.Bucket == "" {
		return fmt.Errorf("missing bucket name")
	}
	if config.Location == "" {
		return fmt.Errorf("missing location")
	}
	if config.PartSize < 5*1024*1024 {
		return fmt.Errorf("invalid part size")
	}
	return nil
}

// BackendDetails additional backend metadata
type BackendDetails struct {
	SSEKey string
}

// Backend object
type Backend struct {
	config *Config
	client *minio.Client
}

// NewBackend instantiate a new S3 Data Backend
// from configuration passed as argument
func NewBackend(config *Config) (b *Backend, err error) {
	b = new(Backend)
	b.config = config

	err = b.config.Validate()
	if err != nil {
		return nil, fmt.Errorf("invalid s3 data backend config : %s", err)
	}

	b.client, err = minio.New(config.Endpoint, &minio.Options{
		Creds: credentials.NewStaticV4(config.AccessKeyID, config.SecretAccessKey, ""),
		//Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
		Secure: config.UseSSL,
	})
	if err != nil {
		return nil, err
	}

	// Check if bucket exists
	exists, err := b.client.BucketExists(context.TODO(), config.Bucket)
	if err != nil {
		return nil, fmt.Errorf("unable to check if bucket %s exists : %s", config.Bucket, err)
	}

	if !exists {
		// Create bucket
		err = b.client.MakeBucket(context.TODO(), config.Bucket, minio.MakeBucketOptions{Region: config.Location})
		if err != nil {
			return nil, fmt.Errorf("unable to create bucket %s : %s", config.Bucket, err)
		}
	}

	return b, nil
}

// GetFile implementation for S3 Data Backend
func (b *Backend) GetFile(file *common.File) (reader io.ReadSeekCloser, err error) {
	getOpts := minio.GetObjectOptions{}

	// Configure server side encryption
	getOpts.ServerSideEncryption, err = b.getServerSideEncryption(file)
	if err != nil {
		return nil, err
	}

	// Try new object name format first ({uploadID}.{fileID})
	obj, err := b.client.GetObject(context.TODO(), b.config.Bucket, b.getObjectName(file.UploadID, file.ID), getOpts)
	if err != nil {
		return nil, fmt.Errorf("unable to get s3 object : %s", err)
	}

	// Peek to check if the object exists (GetObject does only basic checking)
	_, statErr := obj.Stat()
	if statErr == nil {
		return obj, nil
	}
	_ = obj.Close()

	// Check if it's a "not found" error before falling back
	errResponse := minio.ToErrorResponse(statErr)
	if errResponse.Code != "NoSuchKey" {
		return nil, fmt.Errorf("unable to get s3 object : %s", statErr)
	}

	// Fall back to legacy object name format ({fileID}) for backward compatibility
	return b.client.GetObject(context.TODO(), b.config.Bucket, b.getObjectNameLegacy(file.ID), getOpts)
}

// AddFile implementation for S3 Data Backend
func (b *Backend) AddFile(file *common.File, fileReader io.Reader) (err error) {
	putOpts := b.newPutObjectOptions(file.Type)

	// Configure server side encryption
	putOpts.ServerSideEncryption, err = b.getServerSideEncryption(file)
	if err != nil {
		return err
	}

	if file.Size > 0 {
		_, err = b.client.PutObject(context.TODO(), b.config.Bucket, b.getObjectName(file.UploadID, file.ID), fileReader, file.Size, putOpts)
	} else {
		// https://github.com/minio/minio-go/issues/989
		// Minio defaults to 128MiB chunks and has to actually allocate a buffer of this size before uploading the chunk
		// This can lead to very high memory usage when uploading a lot of small files in parallel
		// We default to 16MiB which allow to store files up to 156GiB ( 10000 chunks of 16MiB ), feel free to adjust this parameter to your needs.
		putOpts.PartSize = b.config.PartSize

		_, err = b.client.PutObject(context.TODO(), b.config.Bucket, b.getObjectName(file.UploadID, file.ID), fileReader, -1, putOpts)
	}
	return err
}

func (b *Backend) newPutObjectOptions(contentType string) minio.PutObjectOptions {
	return minio.PutObjectOptions{
		ContentType:    contentType,
		SendContentMd5: b.config.SendContentMd5,
	}
}

// RemoveFile implementation for S3 Data Backend
func (b *Backend) RemoveFile(file *common.File) (err error) {
	// Try removing the new object name format first ({uploadID}.{fileID})
	objectName := b.getObjectName(file.UploadID, file.ID)
	err = b.client.RemoveObject(context.TODO(), b.config.Bucket, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		errResponse := minio.ToErrorResponse(err)
		if errResponse.Code != "NoSuchKey" {
			return fmt.Errorf("unable to remove s3 object %s : %s", objectName, err)
		}

		// Fall back to legacy object name format ({fileID}) for backward compatibility
		legacyName := b.getObjectNameLegacy(file.ID)
		err = b.client.RemoveObject(context.TODO(), b.config.Bucket, legacyName, minio.RemoveObjectOptions{})
		if err != nil {
			errResponse = minio.ToErrorResponse(err)
			if errResponse.Code == "NoSuchKey" {
				return nil
			}
			return fmt.Errorf("unable to remove s3 object %s : %s", legacyName, err)
		}
	}

	return nil
}

func (b *Backend) getObjectName(uploadID, fileID string) string {
	name := fmt.Sprintf("%s.%s", uploadID, fileID)
	if b.config.Prefix != "" {
		return fmt.Sprintf("%s/%s", b.config.Prefix, name)
	}
	return name
}

// getObjectNameLegacy returns the legacy object name format for backward compatibility
// with objects stored before the {uploadID}.{fileID} naming convention.
func (b *Backend) getObjectNameLegacy(fileID string) string {
	if b.config.Prefix != "" {
		return fmt.Sprintf("%s/%s", b.config.Prefix, fileID)
	}
	return fileID
}
