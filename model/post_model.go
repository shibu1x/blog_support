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

const (
	imageResizeSize = "1024x1024"
	imgDirName      = "img"
	imgSrcDirName   = "img_src"
	indexFileName   = "index.md"
)

type Config struct {
	PostDir          string
	AwsRegion        string
	S3BucketName     string
	S3KeyPrefix      string
	RemoteImgBaseURL string
}

var config Config

// LoadEnv loads configuration from environment variables.
// Returns an error if required environment variables are not set.
func LoadEnv() error {
	config = Config{
		PostDir:          getEnv("POST_DIR", ""),
		AwsRegion:        getEnv("AWS_REGION", ""),
		S3BucketName:     getEnv("S3_BUCKET_NAME", ""),
		S3KeyPrefix:      getEnv("S3_KEY_PREFIX", ""),
		RemoteImgBaseURL: getEnv("REMOTE_IMG_BASE_URL", ""),
	}

	if config.PostDir == "" {
		return fmt.Errorf("POST_DIR environment variable is required")
	}
	if config.S3BucketName == "" {
		return fmt.Errorf("S3_BUCKET_NAME environment variable is required")
	}
	if config.RemoteImgBaseURL == "" {
		return fmt.Errorf("REMOTE_IMG_BASE_URL environment variable is required")
	}

	return nil
}

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

// CreateNewPost creates a new blog post.
// date: post date
// number: sequential number for multiple posts on the same day (0 means no sequence number)
// Returns: Post struct representing the created post
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

	return Post{date: date, number: number, dir: dir, dirPath: filepath.Join(config.PostDir, dir)}
}

func (p Post) createDirectory() error {
	err := os.MkdirAll(p.dirPath, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create directory %s: %w", p.dirPath, err)
	}

	dirs := []string{imgDirName, imgSrcDirName}
	for _, dir := range dirs {
		dirPath := filepath.Join(p.dirPath, dir)
		if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
			return fmt.Errorf("failed to create %s directory at %s: %w", dir, dirPath, err)
		}
	}

	return nil
}

func (p Post) createIndexFile() error {
	indexFilePath := filepath.Join(p.dirPath, indexFileName)
	if _, err := os.Stat(indexFilePath); os.IsNotExist(err) {
		file, err := os.Create(indexFilePath)
		if err != nil {
			return fmt.Errorf("failed to create index.md file at %s: %w", indexFilePath, err)
		}
		defer file.Close()

		slug := p.date.Format("20060102")
		if p.number > 0 {
			slug += fmt.Sprintf("%d", p.number)
		}

		contentStr := fmt.Sprintf(`---
title: 
slug: %s
date: %s
image: img/cover.jpg
categories:
- Test
tags:
---

## Background

`, slug, p.date.Format("2006-01-02"))

		if _, err = file.WriteString(contentStr); err != nil {
			return fmt.Errorf("failed to write to index.md file at %s: %w", indexFilePath, err)
		}
	}
	return nil
}

// resizeImages resizes images in the img_src directory,
// saves them to the img directory, and adds the resized file names
// to index.md.
func (p Post) resizeImages() error {
	imgSrcDir := filepath.Join(p.dirPath, imgSrcDirName)
	imgDir := filepath.Join(p.dirPath, imgDirName)

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
		cmd := exec.Command("convert", srcPath, "-resize", imageResizeSize, destPath)
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

	indexFilePath := filepath.Join(p.dirPath, indexFileName)
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

// removeUnusedImages removes image files that are not referenced
// in index.md. However, files starting with 'cover.' are not deleted.
func (p Post) removeUnusedImages() error {
	imgDir := filepath.Join(p.dirPath, imgDirName)
	indexFilePath := filepath.Join(p.dirPath, indexFileName)

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

// uploadImagesToS3 uploads image files to AWS S3.
func (p Post) uploadImagesToS3() error {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(config.AwsRegion),
	})
	if err != nil {
		return fmt.Errorf("failed to create AWS session: %w", err)
	}

	svc := s3.New(sess)
	imgDir := filepath.Join(p.dirPath, imgDirName)

	files, err := os.ReadDir(imgDir)
	if err != nil {
		return fmt.Errorf("failed to read img directory at %s: %w", imgDir, err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := filepath.Join(imgDir, file.Name())
		f, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("failed to open file %s: %w", filePath, err)
		}
		defer f.Close()

		s3Key := fmt.Sprintf("%s/%s/img/%s", config.S3KeyPrefix, p.dir, filepath.Base(filePath))
		_, err = svc.PutObject(&s3.PutObjectInput{
			Bucket: aws.String(config.S3BucketName),
			Key:    aws.String(s3Key),
			Body:   f,
		})
		if err != nil {
			return fmt.Errorf("failed to upload file %s to S3: %w", filePath, err)
		}
	}

	return nil
}

func (p Post) removeImageDirectories() error {
	imgDir := filepath.Join(p.dirPath, imgDirName)
	imgSrcDir := filepath.Join(p.dirPath, imgSrcDirName)

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

// replaceImageLinks replaces image links in index.md
// with remote URLs.
func (p Post) replaceImageLinks() error {
	remoteImgDir := fmt.Sprintf("%s/%s/", config.RemoteImgBaseURL, p.dir)

	indexFilePath := filepath.Join(p.dirPath, indexFileName)
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

// CreatePost creates a new post, sets up necessary directory structure and files.
// It also handles image resizing.
func (p Post) CreatePost() error {
	if err := p.createDirectory(); err != nil {
		return fmt.Errorf("failed to create directory structure: %w", err)
	}

	if err := p.createIndexFile(); err != nil {
		return fmt.Errorf("failed to create index file: %w", err)
	}

	if err := p.resizeImages(); err != nil {
		return fmt.Errorf("failed to resize images: %w", err)
	}

	return nil
}

// publishPost publishes the post by:
// - Removing unused images
// - Uploading images to S3
// - Replacing image links
// - Removing local image directories
func (p Post) publishPost() error {
	imgDir := filepath.Join(p.dirPath, imgDirName)
	if _, err := os.Stat(imgDir); os.IsNotExist(err) {
		return nil
	}

	if err := p.removeUnusedImages(); err != nil {
		return fmt.Errorf("failed to remove unused images: %w", err)
	}

	if err := p.uploadImagesToS3(); err != nil {
		return fmt.Errorf("failed to upload images to S3: %w", err)
	}

	if err := p.replaceImageLinks(); err != nil {
		return fmt.Errorf("failed to replace image links: %w", err)
	}

	if err := p.removeImageDirectories(); err != nil {
		return fmt.Errorf("failed to remove image directories: %w", err)
	}

	return nil
}

// PublishYearPosts publishes all posts for the specified year.
// year: year of posts to publish (0 means current year)
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

// scanPostDirectories scans directories for the specified year and
// returns a slice of Post structs corresponding to each post directory.
// year: year to scan (0 means current year)
func scanPostDirectories(year int) ([]Post, error) {
	if year == 0 {
		year = time.Now().Year()
	}

	var posts []Post
	err := filepath.Walk(filepath.Join(config.PostDir, fmt.Sprintf("%d", year)), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() || filepath.Base(path) != imgDirName {
			return nil
		}

		path = strings.Replace(path, config.PostDir+"/", "", 1)

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
