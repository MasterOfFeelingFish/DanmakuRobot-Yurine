package tools

import (
	"DanmakuRobot_Moon/model"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/skip2/go-qrcode"
)

const (
	PollInterval = 2 * time.Second
	PollTimeout  = 180 * time.Second
)

// 登录方法：
// 参考https://github.com/SocialSisterYi/bilibili-API-collect/blob/master/docs/login/login_action/QR.md
// 根据API参考实现，获取qrcode以获取cookie并保存

func GetLoginURL() (*model.QrcodeResponse, error) {
	resp, err := http.Get("https://passport.bilibili.com/x/passport-login/web/qrcode/generate")
	if err != nil {
		return nil, fmt.Errorf("API请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("非200状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("响应体读取失败: %w", err)
	}

	var qrResp model.QrcodeResponse
	if err := json.Unmarshal(body, &qrResp); err != nil {
		return nil, fmt.Errorf("JSON解析失败: %w", err)
	}

	if qrResp.Code != 0 {
		return nil, fmt.Errorf("API返回错误: [%d]%s", qrResp.Code, qrResp.Message)
	}

	return &qrResp, nil
}

// 轮询操作
func PollLoginStatus(qrcodeKey string) (*model.LoginInfo, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	url := fmt.Sprintf("https://passport.bilibili.com/x/passport-login/web/qrcode/poll?qrcode_key=%s", qrcodeKey)

	startTime := time.Now()
	for time.Since(startTime) < PollTimeout {
		resp, err := client.Get(url)
		if err != nil {
			return nil, fmt.Errorf("轮询请求失败: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		// 新增：解析Cookies
		cookies := parseResponseCookies(resp.Header) // 关键修改点

		// 调试输出
		fmt.Printf("原始响应数据:\n%s\n", string(body))
		fmt.Printf("获取Cookies: %+v\n", cookies) // 调试信息
		fmt.Println("----------------------------------------")

		var statusResp struct {
			Code    int             `json:"code"`
			Message string          `json:"message"`
			Data    model.LoginInfo `json:"data"`
		}

		if err := json.Unmarshal(body, &statusResp); err != nil {
			continue
		}

		// 将Cookies存入Data
		statusResp.Data.Cookies = cookies // 关键赋值

		switch statusResp.Code {
		case 0:
			if statusResp.Data.Code == 0 { // 登录成功
				return &statusResp.Data, nil
			}
		case 86038: // 二维码过期
			return nil, fmt.Errorf("二维码已过期")
		case 86090: // 已扫码未确认
			// 继续轮询
		case 86101: // 未扫码
			// 继续轮询
		}

		time.Sleep(PollInterval)
	}

	return nil, fmt.Errorf("轮询超时")
}

func parseResponseCookies(header http.Header) []*http.Cookie {
	// 使用标准库方法解析Set-Cookie头
	cookies := header.Values("set-cookie")
	var parsedCookies []*http.Cookie

	for _, c := range cookies {
		// 解析单个Cookie字符串
		cookie, err := http.ParseCookie(c)
		if err != nil {
			continue // 忽略解析失败的Cookie
		}
		parsedCookies = append(parsedCookies, cookie...)
	}
	return parsedCookies
}

// 登录方法
// 该方法直接生成一个二维码，并且轮询检测登录状态并且返回cookie
func Login_In() error {
	// 获取二维码
	qrResp, err := GetLoginURL()
	if err != nil {
		panic(err)
	}

	// 生成二维码图片（需要二维码生成库）
	validURL := strings.ReplaceAll(qrResp.Data.Url, `\u0026`, "&")
	qrcode.WriteFile(validURL, qrcode.Medium, 256, "qrcode.png")

	fmt.Printf("二维码已生成")

	// 开始轮询
	loginInfo, err := PollLoginStatus(qrResp.Data.QrcodeKey)
	if err != nil {
		panic(err)
	}

	// 获取到登录凭证和Cookies
	fmt.Printf("Refresh Token: %s\n", loginInfo.RefreshToken)
	fmt.Println("重要Cookies:")
	for _, c := range loginInfo.Cookies {
		fmt.Printf("%s=%s (Domain:%s Path:%s Expires:%v)\n",
			c.Name, c.Value, c.Domain, c.Path, c.Expires)
	}
	fmt.Println("cookies:", loginInfo.Cookies)
	// 保存Cookies到文件
	if err := saveCookiesToFile(loginInfo.Cookies); err != nil {
		fmt.Println("保存Cookies到文件失败:", err)
	}

	return nil
}

func saveCookiesToFile(cookies []*http.Cookie) error {
	type Cookie struct {
		Name     string
		Value    string
		Domain   string
		Path     string
		Expires  time.Time
		Secure   bool
		HttpOnly bool
	}

	// 转换结构体以便序列化
	var saveCookies []Cookie
	for _, c := range cookies {
		saveCookies = append(saveCookies, Cookie{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Expires:  c.Expires,
			Secure:   c.Secure,
			HttpOnly: c.HttpOnly,
		})
	}

	data, err := json.MarshalIndent(saveCookies, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile("cookies.json", data, 0644)
}
