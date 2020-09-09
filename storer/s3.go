package storer

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
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
		fmt.Println("Failed to open file for upload to S3 bucket", filename, err)
		os.Exit(1)
	}
	defer file.Close()

	credentialsProvider, err := getCredentialsProvider()
	if err != nil {
		if viper.GetBool(misc.DebugKey) {
			log.Printf("Could not obtain AWS credentials provider. Error: %s", err)
		}
		return err
	}
	if viper.GetBool(misc.DebugKey) {
		log.Printf("obtained AWS credentials provider")
	}
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

func getCredentialsProvider() (credentials.Provider, error) {

	//NOTE: Prehaps it worth here to falback to shared credentials provider
	ak := viper.GetString(misc.AWSAccessKey)
	if len(ak) == 0 {
		return nil, errors.New("AWS access key was not configured")
	}
	sk := viper.GetString(misc.AWSSecretKey)
	if len(sk) == 0 {
		return nil, errors.New("AWS secret key was not configured")
	}

	secretValue := credentials.Value{AccessKeyID: ak, SecretAccessKey: sk, ProviderName: providerName}
	provider := credentials.StaticProvider{secretValue}
	return &provider, nil
}
