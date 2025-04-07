// 用于处理b站的二维码
package model

// b站登录API响应结构体
type QrcodeResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Ttl     int    `json:"ttl"`
	Data    struct {
		Url       string `json:"url"`
		QrcodeKey string `json:"qrcode_key"`
	} `json:"data"`
}
