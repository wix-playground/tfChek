package storer

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/spf13/viper"
	"github.com/wix-system/tfChek/misc"
	"log"
	"os"
	"path/filepath"
)

const providerName = "tfChek_custom_AWS_provider"

func S3UploadTask(bucket string, id int, suffix *string) error {
	return S3UploadTaskWithSuffix(bucket, id, nil)
}

func S3UploadTaskWithSuffix(bucket string, id int, suffix *string) error {
	dir := viper.GetString(misc.OutDirKey)
	awsRegion := viper.GetString(misc.AWSRegion)
	filename := getTaskPath(dir, id)
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file %s for upload to S3 bucket. Error: %w", filename, err)
	}
	defer file.Close()

	credentialsProvider, err := getCredentialsProvider()
	if err != nil {
		misc.Debugf("could not obtain AWS credentials provider. Error: %s", err)
		return err
	}
	misc.Debug("obtained AWS credentials provider")
	creds := credentials.NewCredentials(credentialsProvider)
	conf := aws.Config{
		Region:      aws.String(awsRegion),
		Credentials: creds,
	}
	sess := session.New(&conf)
	svc := s3manager.NewUploader(sess)
	if viper.GetBool(misc.DebugKey) {
		fmt.Println("Uploading file to S3")
	}
	key := filepath.Base(filename)
	if suffix != nil {
		key = fmt.Sprintf("%s-%s", key, *suffix)
	}
	result, err := svc.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   file,
	})
	if err != nil {
		if viper.GetBool(misc.DebugKey) {
			log.Printf("Cannot upload to S3. Error: %s", err)
		}
		return err
	}
	if viper.GetBool(misc.DebugKey) {
		log.Printf("Successfully uploaded %s to %s\n", filename, result.Location)
	}
	return nil
}

func S3DownloadTask(bucket string, id int) error {
	return S3DownloadTaskWithSuffix(bucket, id, nil)
}

func S3DownloadTaskWithSuffix(bucket string, id int, suffix *string) error {

	filename, file, err := s3PrepareDir(id)
	if err != nil {
		return fmt.Errorf("failed to prepare output directory. Error: %w", err)
	}
	key := s3PrepareKey(filename, suffix)
	downloader, err := s3PrepareDownloader()
	if err != nil {
		return fmt.Errorf("failed to prepare S3 downloader. Error: %w", err)
	}

	input := &s3.GetObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)}
	n, err := downloader.Download(file, input)
	if err != nil {
		misc.Debugf("cannot download from S3. Error: %s", err)
		misc.Debugf("removing file %s after unsuccessful download", filename)
		ferr := os.Remove(filename)
		if ferr != nil {
			misc.Debugf("failed to remove %s Error: %s", filename, err)
		}
		return fmt.Errorf("cannot download from S3. Error: %w", err)
	}
	misc.Debugf("successfully downloaded %d bytes to %s\n", n, filename)
	return nil
}
