package model

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

func LoadEnv() {
	postDir = getEnv("POST_DIR", "")
	awsRegion = getEnv("AWS_REGION", "ap-northeast-1")
	s3BucketName = getEnv("S3_BUCKET_NAME", "")
	s3KeyPrefix = getEnv("S3_KEY_PREFIX", "")
	remoteImgBaseURL = getEnv("REMOTE_IMG_BASE_URL", "")
}

var (
	postDir          string
	awsRegion        string
	s3BucketName     string
	s3KeyPrefix      string
	remoteImgBaseURL string
)

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

type Post struct {
	date    time.Time
	number  int
	dir     string
	dirPath string
}

func CreateNewPost(date time.Time, number int) Post {
	if date.Year() == 0 {
		currentYear := time.Now().Year()
		date = date.AddDate(currentYear, 0, 0)
	}

	suffix := ""
	if number > 0 {
		suffix = "_" + fmt.Sprintf("%d", number)
	}

	dir := filepath.Join(fmt.Sprintf("%d", date.Year()), fmt.Sprintf("%02d", date.Month()), fmt.Sprintf("%02d%s", date.Day(), suffix))

	return Post{date: date, number: number, dir: dir, dirPath: filepath.Join(postDir, dir)}
}

func (p Post) createDirectory() error {
	err := os.MkdirAll(p.dirPath, os.ModePerm)
	if err != nil {
		return fmt.Errorf("error creating directory: %v", err)
	}

	// Create additional directories
	dirs := []string{"img", "img_src"}
	for _, dir := range dirs {
		dirPath := fmt.Sprintf("%s/%s", p.dirPath, dir)
		err = os.MkdirAll(dirPath, os.ModePerm)
		if err != nil {
			return fmt.Errorf("error creating %s directory: %v", dir, err)
		}
		fmt.Printf("Directory created: %s\n", dirPath)
	}

	return nil
}

func (p Post) createIndexFile() error {
	indexFilePath := fmt.Sprintf("%s/index.md", p.dirPath)
	if _, err := os.Stat(indexFilePath); os.IsNotExist(err) {
		file, err := os.Create(indexFilePath)
		if err != nil {
			return fmt.Errorf("error creating index.md file: %v", err)
		}
		defer file.Close()

		slug := p.date.Format("20060102") + fmt.Sprintf("%d", p.number)

		contentStr := fmt.Sprintf(`---
title: 
slug: %s
date: %s
image: img/cover.jpg
categories:
- テスト
tags:
---

## きっかけ

`, slug, p.date.Format("2006-01-02"))
		_, err = file.WriteString(contentStr)
		if err != nil {
			return fmt.Errorf("error writing to index.md file: %v", err)
		}
		fmt.Printf("index.md file created: %s\n", indexFilePath)
	}
	return nil
}

func (p Post) resizeImages() error {
	imgSrcDir := filepath.Join(p.dirPath, "img_src")
	imgDir := filepath.Join(p.dirPath, "img")

	files, err := os.ReadDir(imgSrcDir)
	if err != nil {
		return fmt.Errorf("error reading img_src directory: %v", err)
	}

	allowedExtensions := map[string]bool{
		".heic": true,
		".webp": true,
		".avif": true,
		".jpg":  true,
		".jpeg": true,
		".png":  true,
	}

	var imgFileNames []string

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		ext := strings.ToLower(filepath.Ext(file.Name()))
		if !allowedExtensions[ext] {
			continue
		}

		imgFileName := file.Name()
		if strings.HasPrefix(imgFileName, "IMG_") {
			imgFileName = strings.Replace(imgFileName, "IMG_", "i", 1)
		}
		if ext != ".png" {
			imgFileName = imgFileName[:len(imgFileName)-len(ext)] + ".jpg"
		}

		imgFileName = strings.ToLower(imgFileName)
		srcPath := filepath.Join(imgSrcDir, file.Name())
		destPath := filepath.Join(imgDir, imgFileName)
		cmd := exec.Command("convert", srcPath, "-resize", "1024x1024", destPath)
		err := cmd.Run()
		if err != nil {
			return fmt.Errorf("error resizing image: %v", err)
		}

		imgFileNames = append(imgFileNames, imgFileName)

		err = os.Remove(srcPath)
		if err != nil {
			return fmt.Errorf("error removing source image: %v", err)
		}
	}

	err = p.writeImageNamesToIndex(imgFileNames)
	if err != nil {
		return fmt.Errorf("error writing dest names to index.md: %v", err)
	}

	return nil
}

func (p Post) writeImageNamesToIndex(imgFileNames []string) error {
	if len(imgFileNames) == 0 {
		return nil
	}

	indexFilePath := filepath.Join(p.dirPath, "index.md")
	file, err := os.OpenFile(indexFilePath, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("error opening index.md file: %v", err)
	}
	defer file.Close()

	for _, name := range imgFileNames {
		_, err := file.WriteString(fmt.Sprintf("![](img/%s)\n\n", name))

		if err != nil {
			return fmt.Errorf("error writing to index.md file: %v", err)
		}
	}

	return nil
}

func (p Post) removeUnusedImages() error {
	imgDir := filepath.Join(p.dirPath, "img")
	indexFilePath := filepath.Join(p.dirPath, "index.md")

	indexContent, err := os.ReadFile(indexFilePath)
	if err != nil {
		return fmt.Errorf("error reading index.md file: %v", err)
	}

	files, err := os.ReadDir(imgDir)
	if err != nil {
		return fmt.Errorf("error reading img directory: %v", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// Skip files that start with 'cover.'
		if strings.HasPrefix(file.Name(), "cover.") {
			continue
		}

		if !strings.Contains(string(indexContent), file.Name()) {
			err := os.Remove(filepath.Join(imgDir, file.Name()))
			if err != nil {
				return fmt.Errorf("error removing unused image: %v", err)
			}
			fmt.Printf("Removed unused image: %s\n", file.Name())
		}
	}

	return nil
}

func (p Post) uploadImagesToS3() error {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(awsRegion),
	})
	if err != nil {
		return fmt.Errorf("error creating AWS session: %v", err)
	}

	svc := s3.New(sess)

	imgDir := filepath.Join(p.dirPath, "img")
	files, err := os.ReadDir(imgDir)
	if err != nil {
		return fmt.Errorf("error reading img directory: %v", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := filepath.Join(imgDir, file.Name())
		file, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("error opening file: %v", err)
		}
		defer file.Close()

		_, err = svc.PutObject(&s3.PutObjectInput{
			Bucket: aws.String(s3BucketName),
			Key:    aws.String(fmt.Sprintf("%s/%s/img/%s", s3KeyPrefix, p.dir, filepath.Base(file.Name()))),
			Body:   file,
		})
		if err != nil {
			return fmt.Errorf("error uploading file to S3: %v", err)
		}
		fmt.Printf("Uploaded %s to S3\n", file.Name())
	}

	return nil
}

func (p Post) removeImageDirectories() error {
	imgDir := filepath.Join(p.dirPath, "img")
	imgSrcDir := filepath.Join(p.dirPath, "img_src")

	err := os.RemoveAll(imgDir)
	if err != nil {
		return fmt.Errorf("error removing img directory: %v", err)
	}

	err = os.RemoveAll(imgSrcDir)
	if err != nil {
		return fmt.Errorf("error removing img_src directory: %v", err)
	}

	fmt.Printf("Removed directories: %s, %s\n", imgDir, imgSrcDir)
	return nil
}

func (p Post) replaceImageLinks() error {
	remoteImgDir := fmt.Sprintf("%s/%s/", remoteImgBaseURL, p.dir)

	indexFilePath := filepath.Join(p.dirPath, "index.md")
	indexContent, err := os.ReadFile(indexFilePath)
	if err != nil {
		return fmt.Errorf("error reading index.md file: %v", err)
	}

	re := regexp.MustCompile(`\]\((img/[^)]+)\)`)
	replacedContent := re.ReplaceAllString(string(indexContent), fmt.Sprintf("](%s$1?d=300x300)", remoteImgDir))

	coverJpgPath := filepath.Join(p.dirPath, "img", "cover.jpg")
	coverPngPath := filepath.Join(p.dirPath, "img", "cover.png")

	if _, err := os.Stat(coverJpgPath); os.IsNotExist(err) {
		if _, err := os.Stat(coverPngPath); err == nil {
			re = regexp.MustCompile(`image: img/cover\..{3}`)
			replacedContent = re.ReplaceAllString(replacedContent, "image: img/cover.png")
		} else {
			re = regexp.MustCompile(`(?m)^image: img/cover\..{3}\n?`)
			replacedContent = re.ReplaceAllString(replacedContent, "")
		}
	}

	re = regexp.MustCompile(`image: (img/cover\..{3})`)
	replacedContent = re.ReplaceAllString(replacedContent, fmt.Sprintf("image: %s$1?d=300x300", remoteImgDir))

	err = os.WriteFile(indexFilePath, []byte(replacedContent), 0644)
	if err != nil {
		return fmt.Errorf("error writing to index.md file: %v", err)
	}

	fmt.Printf("Replaced image links in: %s\n", indexFilePath)
	return nil
}

func (p Post) CreatePost() error {
	var err error

	// Create directory structure
	if err := p.createDirectory(); err != nil {
		return fmt.Errorf("error creating directory: %v", err)
	}

	if err = p.createIndexFile(); err != nil {
		return err
	}

	if err = p.resizeImages(); err != nil {
		return fmt.Errorf("error resizing images: %v", err)
	}
	return nil
}

func (p Post) publishPost() error {

	imgDir := filepath.Join(p.dirPath, "img")
	if _, err := os.Stat(imgDir); os.IsNotExist(err) {
		return nil
	}

	// Remove unused images
	err := p.removeUnusedImages()
	if err != nil {
		return fmt.Errorf("error removing unused images: %v", err)
	}

	// Upload images to S3
	err = p.uploadImagesToS3()
	if err != nil {
		return fmt.Errorf("error uploading images to S3: %v", err)
	}

	// Replace image links
	err = p.replaceImageLinks()
	if err != nil {
		return fmt.Errorf("error replacing image links: %v", err)
	}

	// Remove image directories
	err = p.removeImageDirectories()
	if err != nil {
		return fmt.Errorf("error removing image directories: %v", err)
	}

	return nil
}

func PublishYearPosts(year int) error {
	posts, err := scanPostDirectories(year)
	if err != nil {
		return fmt.Errorf("error scanning directories: %v", err)
	}

	for _, post := range posts {
		err := post.publishPost()
		if err != nil {
			return fmt.Errorf("error publishing post: %v", err)
		}
	}

	return nil
}

// ScanDirectories scans the content/post directory and calls NewPost for each subdirectory
func scanPostDirectories(year int) ([]Post, error) {
	if year == 0 {
		year = time.Now().Year()
	}

	var posts []Post
	err := filepath.Walk(filepath.Join(postDir, fmt.Sprintf("%d", year)), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() || filepath.Base(path) != "img" {
			return nil
		}

		path = strings.Replace(path, postDir+"/", "", 1)

		dirParts := strings.Split(path, string(os.PathSeparator))
		year, _ := strconv.Atoi(dirParts[0])
		month, _ := strconv.Atoi(dirParts[1])
		day, _ := strconv.Atoi(dirParts[2])
		date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
		number := 0
		if strings.Contains(dirParts[2], "_") {
			dayParts := strings.Split(dirParts[2], "_")
			day, _ := strconv.Atoi(dayParts[0])
			date = time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
			number, _ = strconv.Atoi(dayParts[1])
		}

		post := CreateNewPost(date, number)

		posts = append(posts, post)

		return nil
	})
	if err != nil {
		return nil, err
	}
	return posts, nil
}
