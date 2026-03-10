package yinghua

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	browser "github.com/EDDYCJY/fake-useragent"
	"github.com/aoaostar/mooc/pkg/config"
	"github.com/aoaostar/mooc/pkg/util"
	"github.com/aoaostar/mooc/pkg/yinghua/types"
	"github.com/go-resty/resty/v2"
	"github.com/sirupsen/logrus"
)

// Global registry of all YingHua instances for graceful shutdown.
var (
	registry   []*YingHua
	registryMu sync.Mutex
)

func init() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		logrus.Info("收到退出信号，正在结束所有活跃会话...")
		registryMu.Lock()
		for _, yh := range registry {
			yh.EndActiveSession()
		}
		registryMu.Unlock()
		os.Exit(0)
	}()
}

type YingHua struct {
	User            config.User
	UserID          int
	Courses         []types.CoursesList
	client          *resty.Client
	activeSessionId string
	sessionMu       sync.Mutex
}

func New(user config.User) *YingHua {

	var client = resty.New()
	client.SetBaseURL(user.BaseURL)
	client.SetRetryCount(3)
	client.SetHeader("user-agent", browser.Mobile())
	yh := &YingHua{
		User:   user,
		client: client,
	}

	registryMu.Lock()
	registry = append(registry, yh)
	registryMu.Unlock()

	return yh

}

func (i *YingHua) Login() error {

	resp := new(types.LoginResponse)
	loginRes, loginErr := i.client.R().SetQueryParams(map[string]string{
		"number":   i.User.Username,
		"password": i.User.Password,
		"schoolId": strconv.Itoa(i.User.SchoolID),
	}).SetResult(resp).Get("/api/user/login")
	if loginErr != nil {
		return loginErr
	}
	if resp.Code != 200 {
		return errors.New(resp.Msg)
	}

	i.client.SetCookies(loginRes.Cookies())
	i.client.SetHeader("Authorization", resp.Result)

	// Fetch student info to obtain the real userId
	infoResp := new(types.StudentInfoResponse)
	_, infoErr := i.client.R().SetResult(infoResp).Get("/api/user/yee_student_info")
	if infoErr != nil {
		return infoErr
	}
	if infoResp.Code != 200 {
		return errors.New(infoResp.Msg)
	}
	i.UserID = infoResp.Data.ID
	i.Output(fmt.Sprintf("登录成功: %s (userId=%d)", infoResp.Data.Name, i.UserID))
	return nil

}

func (i *YingHua) GetCourses() error {

	resp := new(types.CoursesResponse)
	_, err := i.client.R().SetQueryParams(map[string]string{
		"schoolId":  strconv.Itoa(i.User.SchoolID),
		"studentId": strconv.Itoa(i.UserID),
		"type":      "0",
		"pageSize":  "1000",
		"pageNum":   "1",
	}).
		SetResult(resp).
		Get("/api/user/yee_my_course_list")
	if err != nil {
		return err
	}

	if resp.Code != 200 {
		return errors.New(resp.Msg)
	}

	// Only keep online courses (offline != 0)
	for _, course := range resp.Result {
		if course.Offline != 0 {
			i.Courses = append(i.Courses, course)
		}
	}
	return nil
}

func (i *YingHua) GetChapters(course types.CoursesList) ([]types.ChapterItem, error) {

	resp := new(types.ChaptersResponse)
	_, err := i.client.R().
		SetResult(resp).
		SetFormData(map[string]string{
			"schoolId": strconv.Itoa(i.User.SchoolID),
			"courseId": strconv.Itoa(course.ID),
		}).
		Post("/api/user/yee_chapter_select")

	if err != nil {
		return nil, err
	}

	if resp.Code != 200 {
		return nil, errors.New(resp.Msg)
	}
	return resp.Data, nil
}

func (i *YingHua) GetNodes(chapterId int) ([]types.ChaptersNodeList, error) {

	resp := new(types.NodesResponse)
	_, err := i.client.R().
		SetQueryParams(map[string]string{
			"schoolId":  strconv.Itoa(i.User.SchoolID),
			"chapterId": strconv.Itoa(chapterId),
		}).
		SetResult(resp).
		Get("/api/user/yee_node_select")

	if err != nil {
		return nil, err
	}

	if resp.Code != 200 {
		return nil, errors.New(resp.Msg)
	}
	return resp.Data, nil
}

func (i *YingHua) GetStudyProgress(course types.CoursesList) (map[int]types.NodeProgress, error) {

	resp := new(types.StudyProgressResponse)
	_, err := i.client.R().
		SetQueryParams(map[string]string{
			"schoolId": strconv.Itoa(i.User.SchoolID),
			"userId":   strconv.Itoa(i.UserID),
			"courseId": strconv.Itoa(course.ID),
		}).
		SetResult(resp).
		Get("/api/user/get_study_progress")
	if err != nil {
		return nil, err
	}
	if resp.Code != 200 {
		return nil, errors.New(resp.Msg)
	}
	m := make(map[int]types.NodeProgress, len(resp.Data.NodeProgressList))
	for _, np := range resp.Data.NodeProgressList {
		m[np.NodeID] = np
	}
	i.Output(fmt.Sprintf("课程进度: 已完成 %d/%d 节 (完成率 %.1f%%)",
		resp.Data.Summary.CompletedNodes, resp.Data.Summary.TotalNodes, resp.Data.Summary.CompletionRate))
	return m, nil
}

func (i *YingHua) StudyCourse(course types.CoursesList) error {
	progressMap, err := i.GetStudyProgress(course)
	if err != nil {
		i.OutputWith(fmt.Sprintf("获取课程进度失败: %s，将逐节检查进度", err.Error()), logrus.Warnf)
		progressMap = nil
	}

	chapters, err := i.GetChapters(course)
	if err != nil {
		return err
	}
	for _, chapter := range chapters {
		i.StudyChapter(chapter, course, progressMap)
	}

	return nil
}

func (i *YingHua) StudyChapter(chapter types.ChapterItem, course types.CoursesList, progressMap map[int]types.NodeProgress) {

	i.Output(fmt.Sprintf("当前章节: [%s][chapterId=%d]", chapter.Name, chapter.ID))
	nodes, err := i.GetNodes(chapter.ID)
	if err != nil {
		i.OutputWith(fmt.Sprintf("获取节点列表失败: %s", err.Error()), logrus.Errorf)
		return
	}
	for _, node := range nodes {
		if node.TabVideo == 1 {
			i.StudyNode(node, course, progressMap)
		}
	}

}

func (i *YingHua) StudyNode(node types.ChaptersNodeList, course types.CoursesList, progressMap map[int]types.NodeProgress) {
	i.Output(fmt.Sprintf("当前节点: [%s][nodeId=%d][时长=%ds]", node.Name, node.ID, node.VideoDuration))

	// -- 1. Determine start percentage --
	startPct := 0.0
	if progressMap != nil {
		if np, ok := progressMap[node.ID]; ok {
			if np.ProgressPercent >= 100 {
				i.Output(fmt.Sprintf("%s[nodeId=%d] 已完成，跳过", node.Name, node.ID))
				return
			}
			startPct = np.ProgressPercent
		} else {
			// Node not in bulk progress map, fall back to last_progress
			startPct = i.fetchLastProgress(node)
			if startPct < 0 {
				return
			}
		}
	} else {
		// No bulk map available, fall back to last_progress
		startPct = i.fetchLastProgress(node)
		if startPct < 0 {
			return
		}
	}

	// -- 2. Start study session --
	startResp := new(types.StudySessionStartResponse)
	_, err := i.client.R().
		SetHeader("content-type", "application/json").
		SetBody(map[string]interface{}{
			"schoolId": i.User.SchoolID,
			"userId":   i.UserID,
			"nodeId":   strconv.Itoa(node.ID),
			"courseId": strconv.Itoa(course.ID),
			"terminal": "web",
		}).
		SetResult(startResp).
		Post("/api/user/study_session_start")
	if err != nil {
		i.OutputWith(fmt.Sprintf("%s[nodeId=%d] 开始会话失败: %s", node.Name, node.ID, err.Error()), logrus.Errorf)
		return
	}
	if startResp.Code != 200 {
		i.OutputWith(fmt.Sprintf("%s[nodeId=%d] 开始会话失败: %s", node.Name, node.ID, startResp.Msg), logrus.Errorf)
		return
	}
	sessionId := startResp.Data
	i.sessionMu.Lock()
	i.activeSessionId = sessionId
	i.sessionMu.Unlock()
	i.Output(fmt.Sprintf("%s[nodeId=%d] 会话已开始: %s, 从 %.0f%% 继续", node.Name, node.ID, sessionId, startPct))

	// -- 3. Heartbeat loop (every 30s, progress is percentage 0~100) --
	// Use 90% of actual elapsed to avoid triggering "progress rollback" limit.
	sessionStart := time.Now()
	currentPct := startPct
	for currentPct < 100.0 {
		time.Sleep(time.Second * 30)
		if node.VideoDuration > 0 {
			elapsed := time.Since(sessionStart).Seconds()
			currentPct = startPct + (elapsed/float64(node.VideoDuration)*100.0) - 1
		}
		if currentPct > 100.0 {
			currentPct = 100.0
		}
		progressStr := fmt.Sprintf("%.0f", currentPct)
		hbResp := new(types.StudySessionResponse)
		_, hbErr := i.client.R().
			SetHeader("content-type", "application/json").
			SetBody(map[string]string{
				"sessionId": sessionId,
				"progress":  progressStr,
			}).
			SetResult(hbResp).
			Post("/api/user/study_session_heartbeat")
		if hbErr != nil {
			i.OutputWith(fmt.Sprintf("%s[nodeId=%d] 心跳失败: %s", node.Name, node.ID, hbErr.Error()), logrus.Errorf)
			break
		}
		i.Output(fmt.Sprintf("%s[nodeId=%d] %s, 当前进度: %s%%", node.Name, node.ID, hbResp.Data, progressStr))
	}

	// -- 4. End session --
	i.sessionMu.Lock()
	i.activeSessionId = ""
	i.sessionMu.Unlock()
	endResp := new(types.StudySessionResponse)
	_, endErr := i.client.R().
		SetHeader("content-type", "application/json").
		SetBody(map[string]string{
			"sessionId": sessionId,
		}).
		SetResult(endResp).
		Post("/api/user/study_session_end")
	if endErr != nil {
		i.OutputWith(fmt.Sprintf("%s[nodeId=%d] 结束会话失败: %s", node.Name, node.ID, endErr.Error()), logrus.Errorf)
		return
	}
	i.Output(fmt.Sprintf("%s[nodeId=%d] %s", node.Name, node.ID, endResp.Data))
}
// fetchLastProgress calls /api/user/last_progress for a single node.
// Returns the progress percentage (0.0~100.0) on success,
// or -1 if the request fails or the node is already complete.
func (i *YingHua) fetchLastProgress(node types.ChaptersNodeList) float64 {
	progressResp := new(types.LastProgressResponse)
	_, err := i.client.R().
		SetQueryParams(map[string]string{
			"nodeId":   strconv.Itoa(node.ID),
			"userId":   strconv.Itoa(i.UserID),
			"schoolId": strconv.Itoa(i.User.SchoolID),
		}).
		SetResult(progressResp).
		Get("/api/user/last_progress")
	if err != nil {
		i.OutputWith(fmt.Sprintf("%s[nodeId=%d] 获取进度失败: %s", node.Name, node.ID, err.Error()), logrus.Errorf)
		return -1
	}
	if progressResp.Code != 200 {
		i.OutputWith(fmt.Sprintf("%s[nodeId=%d] 获取进度失败: %s", node.Name, node.ID, progressResp.Msg), logrus.Errorf)
		return -1
	}
	pct, parseErr := strconv.ParseFloat(progressResp.Data, 64)
	if parseErr != nil {
		pct = 0
	}
	if pct >= 100.0 {
		i.Output(fmt.Sprintf("%s[nodeId=%d] 已完成，跳过", node.Name, node.ID))
		return -1
	}
	return pct
}

func (i *YingHua) EndActiveSession() {
	i.sessionMu.Lock()
	sessionId := i.activeSessionId
	i.activeSessionId = ""
	i.sessionMu.Unlock()

	if sessionId == "" {
		return
	}
	i.Output(fmt.Sprintf("正在结束会话: %s", sessionId))
	endResp := new(types.StudySessionResponse)
	_, err := i.client.R().
		SetHeader("content-type", "application/json").
		SetBody(map[string]string{
			"sessionId": sessionId,
		}).
		SetResult(endResp).
		Post("/api/user/study_session_end")
	if err != nil {
		i.OutputWith(fmt.Sprintf("结束会话失败: %s", err.Error()), logrus.Errorf)
		return
	}
	i.Output(fmt.Sprintf("会话已结束: %s", endResp.Data))
}

func (i *YingHua) Output(message string) {
	i.OutputWith(message, logrus.Infof)
}

func (i *YingHua) OutputWith(message string, writer func(format string, args ...interface{})) {
	writer("[协程ID=%d][%s] %s", util.GetGid(), i.User.Username, message)
}
