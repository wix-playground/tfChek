package storer

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/spf13/viper"
	"io"
	"log"
	"os"
	"path/filepath"
	"tfChek/misc"
)

//func Save2FileFromReader(id int, in io.Reader) error {
//	dir := viper.GetString("outdir")
//	file, err := os.Create(fmt.Sprintf("%s/task-%d", dir, id))
//	if err != nil {
//		log.Printf("Cannot create file task-%d Error %s", id, err)
//		return err
//	}
//	defer file.Close()
//	fInfo, err := file.Stat()
//	if err != nil {
//		log.Printf("Cannot get file task-%d info. Error: %s", id, err)
//		return err
//	}
//	buf := make([]byte, 4096)
//	bin := bufio.NewReader(in)
//	for {
//		n, err := bin.Read(buf)
//		if err != nil {
//			if err == io.EOF {
//				file.Write(buf[:n])
//				break
//			} else {
//				log.Printf("Cannot create file task-%d Error %s", id, err)
//				return err
//			}
//		}
//		file.Write(buf)
//	}
//	log.Printf("Task %d output has been stored to file %s", id, fInfo.Name())
//	return nil
//}

func getTaskPath(dir string, id int) string {
	return fmt.Sprintf("%s/task-%d", dir, id)
}

func GetTaskFileWriteCloser(id int) (io.WriteCloser, error) {
	dir := viper.GetString(misc.OutDirKey)
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			log.Printf("Cannot create directory %s Error %s", dir, err)
		}
	}
	file, err := os.Create(getTaskPath(dir, id))
	if err != nil {
		log.Printf("Cannot create file task-%d Error %s", id, err)
		return nil, err

	}
	return file, nil
}

func S3UploadTask(bucket string, id int) {
	dir := viper.GetString(misc.OutDirKey)
	awsRegion := viper.GetString(misc.AWSRegion)
	filename := getTaskPath(dir, id)
	file, err := os.Open(filename)
	if err != nil {
		fmt.Println("Failed to open file for upload to S3 bucket", filename, err)
		os.Exit(1)
	}
	defer file.Close()

	//TODO fix credentials
	conf := aws.Config{
		Region:      aws.String(awsRegion),
		Credentials: credentials.NewSharedCredentials("", "production_42"),
	}
	sess := session.New(&conf)
	svc := s3manager.NewUploader(sess)
	if viper.GetBool(misc.DebugKey) {
		fmt.Println("Uploading file to S3")
	}
	result, err := svc.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(filepath.Base(filename)),
		Body:   file,
	})
	if err != nil {
		log.Println("error", err)
	}

	log.Printf("Successfully uploaded %s to %s\n", filename, result.Location)
}
