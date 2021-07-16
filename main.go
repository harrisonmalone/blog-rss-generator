package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	}
	bucket := aws.String(os.Getenv("S3_BUCKET"))

	s3Config := &aws.Config{
		Credentials: credentials.NewStaticCredentials(os.Getenv("ACCESS_KEY"), os.Getenv("SECRET_KEY"), ""),
		Region:      aws.String("ap-southeast-2"),
	}

	newSession := session.New(s3Config)

	s3Client := s3.New(newSession)

	output, err := s3Client.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: bucket,
	})

	if err != nil {
		fmt.Println(err.Error())
		fmt.Printf("Failed fetch items from bucket: %s", *bucket)
		return
	}

	downloader := s3manager.NewDownloader(newSession)

	for _, item := range *&output.Contents {
		objectFilePath := fmt.Sprintf("./posts/%s", *item.Key)

		file, err := os.Create(objectFilePath)

		if err != nil {
			fmt.Println(err.Error())
			return
		}

		_, err = downloader.Download(file,
			&s3.GetObjectInput{
				Bucket: bucket,
				Key:    aws.String(*item.Key),
			})

		if err != nil {
			fmt.Println(err.Error())
			return
		}

		content, err := ioutil.ReadFile(objectFilePath)
		fmt.Println("")
		fmt.Println(string(content))
		fmt.Println("")
	}
}

// next steps
// 1. sort objects by date into array
// 2. convert strings of content into html
// 3. create rss file
// 4. upload rss file to netlify, write deploy script
// 5. update frontend code to use rss data and not fetch for each file individually from s3
