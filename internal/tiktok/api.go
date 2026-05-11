package tt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

var (
	URL              string        = "https://tikwm.com/api"
	Timeout          time.Duration = time.Second + time.Millisecond*100
	MaxUserFeedCount int           = 33
	Debug                          = false
	requestSync      *sync.Mutex   = &sync.Mutex{}
)

func Raw(method string, query map[string]string) ([]byte, error) {
	if Timeout != 0 {
		requestSync.Lock()
		defer unlock()
	}

	url := fmt.Sprintf("%s/%s", URL, method)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	for key, val := range query {
		q.Add(key, val)
	}
	req.URL.RawQuery = q.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	buffer, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if Debug {
		log.Print(string(buffer))
	}

	return buffer, nil
}

func RawParsed[T any](method string, query map[string]string) (*T, error) {
	data, err := Raw(method, query)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Code          int     `json:"code"`
		Msg           string  `json:"msg"`
		ProcessedTime float64 `json:"processed_time"`
		Data          *T      `json:"data"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		queryStr := "???"
		if buf, err := json.Marshal(query); err == nil {
			queryStr = string(buf)
		}
		return nil, fmt.Errorf("tikwm error: %s (%d) [%s, query: %s]", resp.Msg, resp.Code, method, queryStr)
	}

	return resp.Data, nil
}

// TaskSubmitResponse represents the response from task submission
type TaskSubmitResponse struct {
	TaskId string `json:"task_id"`
	Detail struct {
		Id        string `json:"id"`
		Vid       string `json:"vid"`
		PlayUrl   string `json:"play_url"`
		Size      int64  `json:"size"`
	} `json:"detail"`
	Status int `json:"status"`
}

// TaskResultResponse represents the response from task result polling
type TaskResultResponse struct {
	TaskId string `json:"task_id"`
	Detail struct {
		Id          string `json:"id"`
		Vid         string `json:"vid"`
		PlayUrl     string `json:"play_url"`
		DownloadUrl string `json:"download_url"`
		Size        int64  `json:"size"`
	} `json:"detail"`
	Status int `json:"status"`
}

// RawPost makes a POST request to tikwm API (similar to Raw but for POST)
func RawPost(method string, payload map[string]string) ([]byte, error) {
	if Timeout != 0 {
		requestSync.Lock()
		defer unlock()
	}

	apiURL := fmt.Sprintf("https://tikwm.com/api/%s", method)
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	buffer, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if Debug {
		log.Print(string(buffer))
	}

	return buffer, nil
}

// submitTask submits a video URL to the task API and returns the task ID
func submitTask(url string) (string, error) {
	data, err := RawPost("video/task/submit", map[string]string{"url": url})
	if err != nil {
		return "", err
	}

	var apiResp struct {
		Code int                  `json:"code"`
		Msg  string               `json:"msg"`
		Data *TaskSubmitResponse `json:"data"`
	}
	if err := json.Unmarshal(data, &apiResp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}
	if apiResp.Code != 0 {
		return "", fmt.Errorf("tikwm error: %s (%d)", apiResp.Msg, apiResp.Code)
	}

	return apiResp.Data.TaskId, nil
}

// getTaskResult polls for task result and returns the original quality URL
func getTaskResult(taskId string) (string, int64, error) {
	data, err := Raw("video/task/result", map[string]string{"task_id": taskId})
	if err != nil {
		return "", 0, err
	}

	var apiResp struct {
		Code int                `json:"code"`
		Msg  string             `json:"msg"`
		Data *TaskResultResponse `json:"data"`
	}
	if err := json.Unmarshal(data, &apiResp); err != nil {
		return "", 0, fmt.Errorf("unmarshal response: %w", err)
	}
	if apiResp.Code != 0 {
		return "", 0, fmt.Errorf("tikwm error: %s (%d)", apiResp.Msg, apiResp.Code)
	}

	// Status 2 means completed, 0 means pending, 1 means processing
	if apiResp.Data.Status == 2 && apiResp.Data.Detail.PlayUrl != "" {
		return apiResp.Data.Detail.PlayUrl, apiResp.Data.Detail.Size, nil
	}

	return "", 0, fmt.Errorf("task not ready yet, status: %d", apiResp.Data.Status)
}

// GetPostOriginal gets a post with original quality using the task API
func GetPostOriginal(url string) (*Post, error) {
	// First get the regular post info
	post, err := GetPost(url, true)
	if err != nil {
		return nil, err
	}

	// Submit task for original quality
	taskId, err := submitTask(url)
	if err != nil {
		// If task submission fails, return the regular post (fallback)
		if Debug {
			log.Printf("Failed to submit task for original quality: %v", err)
		}
		return post, nil
	}

	// Poll for result (with retries)
	maxRetries := 10
	retryDelay := time.Second * 2
	for i := 0; i < maxRetries; i++ {
		originalUrl, size, err := getTaskResult(taskId)
		if err == nil && originalUrl != "" {
			post.Original = originalUrl
			post.OriginalSize = size
			return post, nil
		}

		// If task is still processing, wait and retry
		if i < maxRetries-1 {
			time.Sleep(retryDelay)
		}
	}

	if Debug {
		log.Printf("Could not get original quality after %d retries, using HD", maxRetries)
	}
	return post, nil
}

// GetPost (hd default: true, original: false)
func GetPost(url string, hd ...bool) (*Post, error) {
	query := map[string]string{"url": url}
	if len(hd) == 0 || hd[0] {
		query["hd"] = "1"
	}
	return RawParsed[Post]("", query)
}

// GetUserFeedRaw is almost unuseful by itself, check wrappers around it -- GetUserFeed/GetUserFeedAwait.
func GetUserFeedRaw(uniqueID string, count int, cursor string) (*UserFeed, error) {
	query := map[string]string{"unique_id": uniqueID, "count": strconv.Itoa(count), "cursor": cursor}
	if _, err := strconv.ParseInt(uniqueID, 10, 64); err == nil {
		query = map[string]string{"user_id": uniqueID, "count": strconv.Itoa(count), "cursor": cursor}
	}
	return RawParsed[UserFeed]("user/posts", query)
}

func GetUserDetail(uniqueID string) (*UserDetail, error) {
	query := map[string]string{"unique_id": uniqueID}
	return RawParsed[UserDetail]("user/info", query)
}

func unlock() {
	time.Sleep(Timeout)
	requestSync.Unlock()
}
