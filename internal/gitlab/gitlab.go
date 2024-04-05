package gitlab

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/firstep/aries/config"
	"github.com/firstep/aries/log"
)

type MergeRequest struct {
	Id           float64
	ProjectId    float64
	Title        string
	Author       string
	CommontCount float64
}

type Comment struct {
	Catetory string    `col:"问题分类"`
	Level    string    `col:"问题等级"`
	Problem  string    `col:"问题确认"`
	Commitor string    `col:"提交人"`
	Author   string    `col:"检视人"`
	Content  string    `col:"检视意见"`
	Filepath string    `col:"代码路径"`
	LineNo   float64   `col:"代码行号"`
	UpdateAt time.Time `col:"检视时间"`
}

var (
	endpoint    string
	token       string
	groupId     int
	timePattern string
	client      *http.Client
)

func init() {
	endpoint = config.GetString("gitlab.endpoint")
	if endpoint == "" {
		panic("gitlab endpoint is required")
	}
	endpoint = strings.TrimSuffix(endpoint, "/")

	token = config.GetString("gitlab.token")
	if token == "" {
		panic("gitlab token is required")
	}
	groupId = config.GetInt("gitlab.groupId", 0)
	if groupId == 0 {
		panic("gitlab group id is required")
	}
	timePattern = config.GetString("gitlab.timePattern", "2006-01-02T15:04:05Z")

	client = &http.Client{
		Timeout: 15 * time.Second,
	}
}

func FetchMergeRequests(page int, timeAfter *time.Time) ([]MergeRequest, error) {
	if timeAfter == nil {
		return nil, nil
	}
	timeAfterStr := timeAfter.UTC().Format("2006-01-02T15:04:05Z")

	url := fmt.Sprintf("%s/groups/%d/merge_requests?state=all&page=%d&per_page=100&updated_after=%s",
		endpoint, groupId, max(page, 1), timeAfterStr)

	bodyBytes, err := exchange(url)
	if err != nil {
		return nil, err
	}

	if log.IsDebugEnabled() {
		log.Debugf("[gitlab]MR list data: %s", string(bodyBytes))
	}

	var apiResult []map[string]any
	err = json.Unmarshal(bodyBytes, &apiResult)
	if err != nil {
		return nil, errors.New("parse MR response failed")
	}

	var results []MergeRequest
	for _, item := range apiResult {
		if item["state"].(string) == "closed" {
			continue
		}
		mr := MergeRequest{
			Id:           item["iid"].(float64),
			ProjectId:    item["project_id"].(float64),
			Title:        item["title"].(string),
			CommontCount: item["user_notes_count"].(float64),
		}
		author := item["author"].(map[string]any)
		mr.Author = author["name"].(string)
		results = append(results, mr)
	}

	return results, nil
}

func FetchMergeRequestComments(mr *MergeRequest) ([]Comment, error) {
	url := fmt.Sprintf("%s/projects/%v/merge_requests/%v/discussions", endpoint, mr.ProjectId, mr.Id)

	bodyBytes, err := exchange(url)
	if err != nil {
		return nil, err
	}

	if log.IsDebugEnabled() {
		log.Debugf("[gitlab]MR[%d] %s comments: %s", mr.Id, mr.Title, string(bodyBytes))
	}

	var apiResult []map[string]any
	err = json.Unmarshal(bodyBytes, &apiResult)
	if err != nil {
		return nil, errors.New("[gitlab]unable to parse MR comment response")
	}

	var results []Comment
	for _, item := range apiResult {
		if item["individual_note"].(bool) {
			continue
		}

		notes := item["notes"].([]any)
		for _, _item := range notes {
			note := _item.(map[string]any)
			body := note["body"].(string)

			cate, level, problem, content := parseNote(body)
			if cate == "" || level == "" {
				continue
			}

			commont := Comment{
				Catetory: cate,
				Level:    level,
				Content:  content,
				Commitor: mr.Author,
			}
			if problem != "非问题" {
				commont.Problem = "是"
			} else {
				commont.Problem = "否"
			}

			author := note["author"].(map[string]any)
			commont.Author = author["name"].(string)

			position := note["position"].(map[string]any)
			commont.Filepath = position["new_path"].(string)
			if position["new_line"] == nil {
				commont.LineNo = position["old_line"].(float64)
			} else {
				commont.LineNo = position["new_line"].(float64)
			}

			updateAt := note["updated_at"].(string)
			if updateAt != "" {
				t, err := time.Parse(timePattern, updateAt)
				if err == nil {
					commont.UpdateAt = t.In(time.UTC)
				}
			}

			results = append(results, commont)
		}

	}

	return results, nil
}

func parseNote(body string) (cate, level, problem, content string) {
	content = body
	cate, content = parseTag(content)
	if cate == "" {
		return
	}

	level, content = parseTag(content)
	if level == "" {
		return
	}

	problem, content = parseTag(content)
	return
}

func parseTag(text string) (tag, newText string) {
	startPos := strings.Index(text, "【")
	if startPos == -1 {
		return "", text
	}

	endPos := strings.Index(text, "】")
	if endPos == -1 {
		return "", text
	}

	if startPos > endPos {
		return "", text
	}

	tag = text[startPos+3 : endPos]

	newText = text[endPos+3:]

	return
}

func exchange(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("PRIVATE-TOKEN", token)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 300 {
		log.Error("[gitlab]failed to fetch data, url:%s, responseStatus:%d, responseBody: %s", url, resp.StatusCode, string(bodyBytes))
		return nil, errors.New("unable to exchange data over http")
	}

	return bodyBytes, nil

}
