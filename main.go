package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/gorilla/feeds"
	"github.com/joho/godotenv"
	"github.com/yuin/goldmark"
)

func reverse(s []*s3.Object) []*s3.Object {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return s
}

func createSlug(lastModified time.Time, title string) string {
	months := map[string]string{"January": "01", "February": "02", "March": "03", "April": "04", "May": "05", "June": "06", "July": "07", "August": "08", "Sepetember": "09", "October": "10", "November": "11", "December": "12"}
	year := lastModified.Year()
	month := lastModified.Month().String()
	return fmt.Sprintf("https://harrisonmalone.dev/%d/%s/%s", year, months[month], strings.TrimSuffix(title, ".txt"))
}

func createTitle(title string) string {
	titleWithoutTxt := strings.TrimSuffix(title, ".txt")
	titleSlice := strings.Split(titleWithoutTxt, "-")
	var titleSliceCapitalized []string
	for _, word := range titleSlice {
		capitalizedWord := strings.Title(word)
		titleSliceCapitalized = append(titleSliceCapitalized, capitalizedWord)
	}
	return strings.Join(titleSliceCapitalized[:], " ")
}

func createFeedItem(html string, item *s3.Object) *feeds.Item {
	return &feeds.Item{
		Title:   createTitle(*item.Key),
		Link:    &feeds.Link{Href: createSlug(*item.LastModified, *item.Key)},
		Author:  &feeds.Author{Name: "Harrison Malone", Email: "harrisonmalone@hey.com"},
		Created: *item.LastModified,
		Content: html,
	}
}

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

	sort.Slice(output.Contents, func(i, j int) bool {
		currentObj := output.Contents[i]
		nextObj := output.Contents[j]
		return currentObj.LastModified.Before(*nextObj.LastModified)
	})

	posts := reverse(output.Contents)

	now := time.Now()
	feed := &feeds.Feed{
		Title:       "harrisonmalone.dev blog",
		Link:        &feeds.Link{Href: "https://harrisonmalone.dev"},
		Description: "👋",
		Author:      &feeds.Author{Name: "Harrison Malone", Email: "harrisonmalone@hey.com"},
		Created:     now,
	}
	var feedItems []*feeds.Item

	for _, item := range posts {
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

		if err != nil {
			fmt.Println(err.Error())
			return
		}

		markdownStr := string(content)
		var buf bytes.Buffer
		if err := goldmark.Convert([]byte(markdownStr), &buf); err != nil {
			panic(err)
		}
		html := string(buf.Bytes())
		feedItems = append(feedItems, createFeedItem(html, item))
	}
	feed.Items = feedItems
	rss, err := feed.ToRss()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	os.WriteFile("./rss.xml", []byte(rss), 0666)
	cmd, err := exec.Command("./upload_rss_to_netlify").CombinedOutput()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println(string(cmd))
}
