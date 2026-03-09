package types

type CoursesResponse struct {
	Code   int           `json:"code"`
	Msg    string        `json:"msg"`
	Result []CoursesList `json:"data"`
}
type CoursesList struct {
	ID            int         `json:"id"`
	Name          string      `json:"name"`
	Mode          int         `json:"mode"`
	CollegeID     int         `json:"collegeId"`
	CategoryID    string      `json:"categoryId"`
	Lecturers     string      `json:"lecturers"`
	StartDate     string      `json:"startDate"`
	EndDate       string      `json:"endDate"`
	Cover         string      `json:"cover"`
	Credit        float32     `json:"credit"`
	Intro         string      `json:"intro"`
	Code          string      `json:"code"`
	StuCount      int         `json:"stuCount"`
	Proclamation  interface{} `json:"proclamation"`
	ClusterID     int         `json:"clusterId"`
	PeriodName    string      `json:"periodName"`
	AddTime       string      `json:"addTime"`
	CreateID      int         `json:"createId"`
	SchoolID      int         `json:"schoolId"`
	CateBid       int         `json:"cateBid"`
	CateMid       int         `json:"cateMid"`
	SignStartTime string      `json:"signStartTime"`
	SignEndTime   string      `json:"signEndTime"`
	SignScope     int         `json:"signScope"`
	SignClass     string      `json:"signClass"`
	LecturerName  string      `json:"lecturerName"`
	Offline       int         `json:"offline"`
	Mission       int         `json:"mission"`
	SignLimit     int         `json:"signLimit"`
	LineLock      int         `json:"lineLock"`
	AddDate       string      `json:"addDate"`
	TplId         int         `json:"tplId"`
	TemplateId    int         `json:"templateId"`
}
