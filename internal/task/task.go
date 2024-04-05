package task

import (
	"bytes"
	"codereview/internal/gitlab"
	"codereview/pkg/export"
	"codereview/pkg/message"
	"context"
	"fmt"
	"time"

	"github.com/firstep/aries/config"
	"github.com/firstep/aries/log"
	"github.com/go-co-op/gocron"
)

var (
	scheduler *gocron.Scheduler
	cron      string
	subject   string
	maxPage   int
	recivers  []string
	location  *time.Location
)

func init() {
	maxPage = config.GetInt("gitlab.maxPage", 10)
	recivers = config.GetStringSlice("smtp.receivers")
	if len(recivers) == 0 {
		panic("[task]smtp receiver address is required")
	}
	subject = config.GetString("smtp.subject")
	if subject == "" {
		panic("[task]smtp subject is required")
	}
	cron = config.GetString("task.cron")
	if cron == "" {
		panic("[task]task cron is required")
	}
	location, _ = time.LoadLocation("Asia/Shanghai")
}

func Start(ctx context.Context) error {
	scheduler = gocron.NewScheduler(location)
	scheduler.WaitForScheduleAll()

	_, err := scheduler.CronWithSeconds(cron).Do(func() {
		FetchReviewData(nil)
	})

	if err != nil {
		return err
	}

	scheduler.StartAsync()
	log.Info("[task]task is running...")
	return nil
}

func Stop(ctx context.Context) error {
	if scheduler != nil {
		scheduler.Stop()
		log.Info("[task]task has stopped")
	}
	return nil
}

func FetchReviewData(timeAfter *time.Time) {
	start := time.Now()
	log.Info("[task]start fetching review data...")
	defer func() {
		if r := recover(); r != nil {
			log.Error("[task]failed to fetch review data", r)
		}
		log.Infof("[task]fetching review data took %v seconds", time.Since(start).Seconds())
	}()

	if timeAfter == nil {
		now := time.Now().In(location)
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location)
		timeAfter = &today
	}

	var mrs []gitlab.MergeRequest
	//MR列表接口一次最多能获取100条数据，此处分页处理
	for i := 1; i <= maxPage; i++ {
		pages, err := gitlab.FetchMergeRequests(i, timeAfter)
		if err != nil {
			log.Error("[task]failed to fetch MR list", err)
			break
		}
		if len(pages) == 0 {
			break
		}
		mrs = append(mrs, pages...)
	}

	log.Infof("[task]fetch MR size: %d", len(mrs))

	var comments []gitlab.Comment
	for _, mr := range mrs {
		if mr.CommontCount == 0 {
			continue
		}

		datas, err := gitlab.FetchMergeRequestComments(&mr)
		if err != nil {
			log.Error("[task]failed to fetch MR comment", err)
			continue
		}

		for _, data := range datas {
			data.UpdateAt = data.UpdateAt.In(location)
			if data.UpdateAt.After(*timeAfter) {
				comments = append(comments, data)
			}
		}
	}

	if len(comments) == 0 {
		log.Info("[task]the current task has not fetched any comments yet")
		return
	}

	announceData(comments)
}

func announceData(comments []gitlab.Comment) {
	var buf bytes.Buffer
	buf.Write([]byte{0xEF, 0xBB, 0xBF}) //BOM byte

	export.ExportCSV(&buf, comments)

	// gbkBuf := transform.NewReader(&buf, simplifiedchinese.GBK.NewEncoder())

	time := time.Now().In(location)

	attach := message.Attach{
		File:     &buf,
		FileName: fmt.Sprintf("codereview-%s.csv", time.Format("20060102150405")),
		MiniType: "text/plain",
	}

	count := len(comments)
	subject := fmt.Sprintf("%s-%s", subject, time.Format("2006-01-02"))
	msg := fmt.Sprintf("Total comments: %d, Date: %s", count, time.Format("2006-01-02 15:04:05"))

	err := message.SendEmailWithText(subject, recivers, msg, attach)
	if err != nil {
		log.Error("[task]failed to send email, ", err)
	}

	log.Info("[task]comments size:", count)
}
