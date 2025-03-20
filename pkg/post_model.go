package pkg

import (
	"fmt"
	"os"
	"time"

	"github.com/araddon/dateparse"
)

type PostModel struct {
	Date time.Time
}

func NewPostModel(dateStr string) (PostModel, error) {
	var date time.Time
	var err error
	date, err = dateparse.ParseAny(dateStr)
	if err != nil {
		return PostModel{}, fmt.Errorf("error parsing date: %v", err)
	}
	if date.Year() == 0 {
		currentYear := time.Now().Year()
		date = date.AddDate(currentYear, 0, 0)
	}
	return PostModel{Date: date}, nil
}

func (p PostModel) CreateDirectory() error {
	// Create a directory structure based on the date
	year := fmt.Sprintf("%d", p.Date.Year())
	month := fmt.Sprintf("%02d", p.Date.Month())
	day := fmt.Sprintf("%02d", p.Date.Day())
	dirPath := fmt.Sprintf("content/post/%s/%s/%s", year, month, day)
	err := os.MkdirAll(dirPath, os.ModePerm)
	if err != nil {
		return fmt.Errorf("error creating directory: %v", err)
	}
	fmt.Printf("Directory created: %s\n", dirPath)
	return nil
}
