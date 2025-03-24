package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/joho/godotenv"
	"github.com/shibu1x/blog_support/model"
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}
	model.LoadEnv()
}

func main() {
	dateStr := flag.String("d", "", "Date in format MM/DD or YYYY/MM/DD")
	number := flag.Int("n", 0, "Number")
	publish := flag.Bool("p", false, "Publish")
	year := flag.Int("y", 0, "Year in format YYYY")
	flag.Parse()

	if *publish {
		model.PublishYearPosts(*year)
		return
	}

	post, err := model.CreateNewPost(*dateStr, *number)
	if err != nil {
		fmt.Printf("Error creating PostModel: %v\n", err)
		return
	}

	post.CreatePost()
}
