package main

import (
	"codereview/internal/task"
	_ "codereview/pkg/log"
	"flag"
	"fmt"
	"time"

	"github.com/firstep/aries"
	"github.com/firstep/aries/log"
)

func main() {
	if startImmediately() {
		return
	}

	startScheduling()
}

func startImmediately() bool {
	timeStr := flag.String("t", "", "Execute once and set MR(updated) after time, pattern: yyyyMMddHHmmss")
	flag.Parse()

	if *timeStr != "" {
		location, _ := time.LoadLocation("Asia/Shanghai")
		time, err := time.ParseInLocation("20060102150405", *timeStr, location)
		if err != nil {
			fmt.Println("Invalid time format, pattern: yyyyMMddHHmmss")
			return true
		}
		task.FetchReviewData(&time)
		fmt.Println("Done!")
		return true
	}

	return false
}

func startScheduling() {
	app := aries.NewApp(
		aries.WithName("CodeReview"),
		aries.WithVersion("1.0.0"),
		aries.WithServer(
			aries.NewServerWraper(task.Start, task.Stop),
		),
	)

	err := app.Run()
	log.Flush()
	if err != nil {
		panic(err)
	}

}
