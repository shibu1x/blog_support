package main

import (
	"flag"
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
	dateStr := flag.String("d", "", "Date in format MM/DD or YYYY/MM/DD")
	number := flag.Int("n", 0, "Number")
	publish := flag.Bool("p", false, "Publish")
	year := flag.Int("y", 0, "Year in format YYYY")
	flag.Parse()

	if *publish {
		model.PublishYearPosts(*year)
		return
	}

	date, err := dateparse.ParseAny(*dateStr)
	if err != nil {
		date = time.Now()
	}

	post := model.CreateNewPost(date, *number)
	post.CreatePost()
}
