package main

import (
	"flag"
	"strconv"
	"time"

	"github.com/araddon/dateparse"
	"github.com/joho/godotenv"
	"github.com/shibu1x/blog_support/model"
)

func init() {
	_ = godotenv.Load()
	model.LoadEnv()
}

func main() {
	number := flag.Int("N", 0, "Number")
	publish := flag.Bool("P", false, "Publish")
	flag.Parse()

	if *publish {
		// Get year from positional argument for publish mode
		var year int
		args := flag.Args()
		if len(args) > 0 {
			year, _ = strconv.Atoi(args[0])
		}
		model.PublishYearPosts(year)
		return
	}

	// Get date from positional argument for post creation
	var dateStr string
	args := flag.Args()
	if len(args) > 0 {
		dateStr = args[0]
	}

	date, err := dateparse.ParseAny(dateStr)
	if err != nil {
		date = time.Now()
	}

	post := model.CreateNewPost(date, *number)
	post.CreatePost()
}
