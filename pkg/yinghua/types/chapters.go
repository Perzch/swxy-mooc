package types

// ChaptersResponse - GET /api/user/yee_chapter_select
type ChaptersResponse struct {
	Code int           `json:"code"`
	Msg  string        `json:"msg"`
	Data []ChapterItem `json:"data"`
}

// ChapterItem is a single chapter (flat structure, no nested nodes)
type ChapterItem struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	CourseID int    `json:"courseId"`
	Sort     int    `json:"sort"`
	SchoolID int    `json:"schoolId"`
}

// NodesResponse - GET /api/user/yee_node_select
type NodesResponse struct {
	Code  int                `json:"code"`
	Msg   string             `json:"msg"`
	Data  []ChaptersNodeList `json:"data"`
	Count int                `json:"count"`
}

// ChaptersNodeList is a single lesson node
type ChaptersNodeList struct {
	ID            int         `json:"id"`
	Name          string      `json:"name"`
	Type          interface{} `json:"type"`
	ChapterID     int         `json:"chapterId"`
	CourseID      int         `json:"courseId"`
	VideoFile     interface{} `json:"videoFile"`
	VideoDuration int         `json:"videoDuration"`
	VotingPath    interface{} `json:"votingPath"`
	TabVideo      int         `json:"tabVideo"`
	TabFile       int         `json:"tabFile"`
	TabVote       int         `json:"tabVote"`
	TabWork       int         `json:"tabWork"`
	TabExam       int         `json:"tabExam"`
	Sort          int         `json:"sort"`
	VideoMode     int         `json:"videoMode"`
	LocalFile     string      `json:"localFile"`
	SchoolID      int         `json:"schoolId"`
	Lock          int         `json:"lock"`
	UnlockTime    int         `json:"unlockTime"`
}

// LastProgressResponse - GET /api/user/last_progress
// Data is a percentage string: "0".."100"
type LastProgressResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data string `json:"data"`
}

// StudySessionStartResponse - POST /api/user/study_session_start
// Data contains the sessionId
type StudySessionStartResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data string `json:"data"`
}

// StudySessionResponse - generic response for heartbeat / end
type StudySessionResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data string `json:"data"`
}

// StudyProgressResponse - GET /api/user/get_study_progress
type StudyProgressResponse struct {
	Code int               `json:"code"`
	Msg  string            `json:"msg"`
	Data StudyProgressData `json:"data"`
}
type StudyProgressData struct {
	Summary          StudyProgressSummary `json:"summary"`
	NodeProgressList []NodeProgress       `json:"nodeProgressList"`
}
type StudyProgressSummary struct {
	TotalVideoDuration string  `json:"totalVideoDuration"`
	CompletedNodes     int     `json:"completedNodes"`
	TotalWatchDuration string  `json:"totalWatchDuration"`
	CompletionRate     float64 `json:"completionRate"`
	TotalNodes         int     `json:"totalNodes"`
}
type NodeProgress struct {
	NodeID          int     `json:"nodeId"`
	NodeName        string  `json:"nodeName"`
	ProgressPercent float64 `json:"progressPercent"`
	State           int     `json:"state"`
	StatusText      string  `json:"statusText"`
}
