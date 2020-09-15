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
	"os"
	"path/filepath"
)

func s3PrepareDir(id int) (string, *os.File, error) {
	dir := viper.GetString(misc.OutDirKey)
	ds, err := os.Stat(dir)
	if os.IsNotExist(err) {
		err := os.MkdirAll(dir, 0775)
		if err != nil {
			misc.Debugf("cannot create directory to store task output files. Error: %s", err)
			return "", nil, fmt.Errorf("cannot create directory to store task output files. Error: %w", err)
		}
	}
	if !ds.IsDir() {
		return "", nil, fmt.Errorf("%s is not a directory. Cannot save task %d output", dir, id)
	}
	filename := getTaskPath(dir, id)
	file, err := os.Create(filename)
	if err != nil {
		return "", nil, fmt.Errorf("failed to open file %s for upload to S3 bucket. Error: %w", filename, err)
	}
	defer file.Close()
	return filename, file, nil
}

func s3PrepareKey(filename string, suffix *string) string {
	key := filepath.Base(filename)
	if suffix != nil {
		key = fmt.Sprintf("%s-%s", key, *suffix)
	}
	return key
}

func s3PrepareDownloader() (*s3manager.Downloader, error) {
	credentialsProvider, err := getCredentialsProvider()
	if err != nil {
		misc.Debugf("could not obtain AWS credentials provider. Error: %s", err)
		return nil, fmt.Errorf("could not obtain AWS credentials provider. Error: %w", err)
	}
	misc.Debug("obtained AWS credentials provider")
	creds := credentials.NewCredentials(credentialsProvider)
	awsRegion := viper.GetString(misc.AWSRegion)
	conf := aws.Config{
		Region:      aws.String(awsRegion),
		Credentials: creds,
	}
	client, err := session.NewSession(&conf)
	if err != nil {
		misc.Debugf("failed to obtain AWS client. Error: %s", err)
		return nil, fmt.Errorf("failed to obtain AWS client. Error: %w", err)
	}
	downloader := s3manager.NewDownloader(client)
	return downloader, nil
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
