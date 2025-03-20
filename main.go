package main

import (
	"fmt"
	"os"

	"github.com/shibu1x/blog_support/pkg"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: main.go <date>")
		return
	}

	postModel, err := pkg.NewPostModel(os.Args[1])
	if err != nil {
		fmt.Printf("Error creating PostModel: %v\n", err)
		return
	}

	// Create directory based on the date
	err = postModel.CreateDirectory()
	if err != nil {
		fmt.Printf("Error creating directory: %v\n", err)
		return
	}
	fmt.Printf("Directory created successfully for date: %s\n", os.Args[1])
}
