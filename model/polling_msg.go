package model

import "net/http"

// 轮询操作的结构体
type LoginInfo struct {
	Url          string `json:"url"`
	RefreshToken string `json:"refresh_token"`
	Timestamp    int64  `json:"timestamp"`
	Code         int    `json:"code"`
	Message      string `json:"message"`
	Cookies      []*http.Cookie
}
