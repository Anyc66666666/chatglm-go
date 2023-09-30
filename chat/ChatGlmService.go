package chat

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Anyc66666666/chatglm-go/entity"
)

type ChatService struct {
	// ... 可能的依赖或其他属性
	authorization string
	cookie        string
}

type tpe string

const (
	baseUrl = "https://chatglm.cn/chatglm/backend-api/v1"
	chat    = "/stream?context_id="
	task    = "/conversation"
	context = "/stream_context"
	refresh = "/user/refresh"

	cookie        = "cookie"
	Authorization = "Authorization"
	accept        = "accept"

	chatType    tpe = "chat"
	taskType    tpe = "task"
	contextType tpe = "context"
	refreshType tpe = "refresh"

	streamAccept = "text/event-stream"
	jsonAccept   = "application/json, text/plain, */*"
)

var client = &http.Client{
	Timeout: time.Minute * 3,
}

var (
	headerCookie = map[string][]string{
		"authority":        {"chatglm.cn"},
		"accept":           {""},
		"accept-language":  {"zh-CN,zh;q=0.9"},
		"cache-control":    {"no-cache"},
		"cookie":           {""},
		"referer":          {"https://chatglm.cn/detail"},
		"sec-ch-ua":        {"\"Not.A/Brand\";v=\"8\", \"Chromium\";v=\"114\", \"Google Chrome\";v=\"114\""},
		"sec-ch-ua-mobile": {"?0"},
		"sec-fetch-dest":   {"empty"},
		"sec-fetch-mode":   {"cors"},
		"sec-fetch-site":   {"same-origin"},
		"user-agent":       {"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36"},
	}

	headerAuth = map[string][]string{
		"Accept":        {"application/json, text/plain, */*"},
		"Content-Type":  {"application/json;charset=utf-8"},
		"Authorization": {""},
	}

	UnauthorizedError = fmt.Errorf("401 token/cookie 已过期")
)

func NewChatService(authorization string, cookie string) *ChatService {
	return &ChatService{
		// ... 初始化依赖或其他属性
		authorization: authorization,
		cookie:        cookie,
	}
}

func (s *ChatService) GetChatStream(contextId string) (bufio.Scanner, error) {
	scanner := bufio.NewScanner(nil)
	url := baseUrl + chat + contextId
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		fmt.Println(err)
		return *scanner, err
	}

	req.Header = s.getHeaderByType(chatType)

	res, err1 := client.Do(req)
	if res != nil {
		defer res.Body.Close()
	}
	if err1 != nil {
		fmt.Println(err1)
		return *scanner, err1
	}
	if res.StatusCode == http.StatusUnauthorized {
		return bufio.Scanner{}, UnauthorizedError
	}

	scanner = bufio.NewScanner(res.Body)
	return *scanner, nil
}

func (s *ChatService) GetChat(contextId string) (string, error) {
	scanner := bufio.NewScanner(nil)
	url := baseUrl + chat + contextId

	req, err := http.NewRequest(http.MethodGet, url, nil)

	if err != nil {
		fmt.Println(err)
		return "", err
	}
	req.Header = s.getHeaderByType(chatType)
	res, err1 := client.Do(req)
	if res != nil {
		defer res.Body.Close()
	}
	if err1 != nil {
		fmt.Println(err1)
		return "", err1
	}
	if res.StatusCode == http.StatusUnauthorized {
		return "", UnauthorizedError
	}

	scanner = bufio.NewScanner(res.Body)
	var ok bool
	var answer string
	for scanner.Scan() {
		str := scanner.Text()
		if ok {
			answer = answer + str + "\n"
			continue
		}
		if str == "event:finish" {
			ok = true
		}
	}
	answer = strings.ReplaceAll(answer, "data:", "")
	return answer, nil
}

func (s *ChatService) GetTaskId(prompt string) (*entity.TaskResponse, error) {
	url := baseUrl + task
	payload := strings.NewReader(`{"prompt":"` + prompt + `"}`)
	req, err := http.NewRequest(http.MethodPost, url, payload)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	req.Header = s.getHeaderByType(taskType)
	res, err1 := client.Do(req)
	if res != nil {
		defer res.Body.Close()
	}
	if err1 != nil {
		fmt.Println(err1)
		return nil, err1
	}
	if res.StatusCode == http.StatusUnauthorized {
		return nil, UnauthorizedError
	}

	body, err2 := io.ReadAll(res.Body)
	if err2 != nil {
		fmt.Println(err2)
		return nil, err2
	}
	var response *entity.TaskResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		fmt.Println(string(body), err)
		return nil, err
	}

	return response, nil
}

func (s *ChatService) GetContextId(prompt string, taskid string) (*entity.ContextResponse, error) {
	url := baseUrl + context
	str := `{"prompt":"` + prompt + `","seed":93549,"max_tokens":512,"conversation_task_id":"` +
		taskid + `","retry":false,"retry_history_task_id":null}`
	payload := strings.NewReader(str)
	req, err := http.NewRequest(http.MethodPost, url, payload)

	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	req.Header = s.getHeaderByType(contextType)

	res, err1 := client.Do(req)
	if res != nil {
		defer res.Body.Close()
	}
	if err1 != nil {
		fmt.Println(err1)
		return nil, err1
	}
	if res.StatusCode == http.StatusUnauthorized {
		return nil, UnauthorizedError
	}

	body, err2 := io.ReadAll(res.Body)
	if err2 != nil {
		fmt.Println(err2)
		return nil, err2
	}
	var response *entity.ContextResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		fmt.Println(string(body), err)
		return nil, err
	}

	return response, nil
}

func (s *ChatService) getHeaderByType(tpe tpe) map[string][]string {
	header := headerCookie
	switch tpe {
	case chatType:
		header[cookie][0] = s.cookie
		header[accept][0] = streamAccept
	case taskType:
		header = headerAuth
		header[Authorization][0] = s.authorization
	case contextType:
		header = headerAuth
		header[Authorization][0] = s.authorization
	case refreshType:
		header = headerAuth
		header[Authorization][0] = s.authorization
	default:

	}
	return header
}

// RefreshToken get new token and update auth
func (s *ChatService) RefreshToken() (string, error) {
	url := baseUrl + refresh
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		fmt.Println(err)
		return "", err
	}

	req.Header = s.getHeaderByType(refreshType)
	res, err1 := client.Do(req)
	if res != nil {
		defer res.Body.Close()
	}
	if err1 != nil {
		fmt.Println(err1)
		return "", err1
	}
	if res.StatusCode == http.StatusUnauthorized {
		return "", UnauthorizedError
	}

	body, err2 := io.ReadAll(res.Body)
	if err2 != nil {
		fmt.Println(err2)
		return "", err
	}
	var response *entity.RefreshResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		fmt.Println(string(body), err)
		return "", err
	}
	token := response.Result.AccessToken
	s.updateAuth(token)
	return token, nil
}

func (s *ChatService) updateAuth(token string) {
	s.authorization = token
}
