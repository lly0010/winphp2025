// Package nettest 探测下载源是否可达 (HEAD / Range 0-0), 不真下载文件.
// 用于前端 "测试连通性" 按钮: 用户安装组件前能看到每个候选 URL 是否能访问.

package nettest

import (
	"context"
	"crypto/tls"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Result struct {
	URL       string `json:"url"`
	OK        bool   `json:"ok"`        // 状态码 2xx/3xx 视为可达
	Status    int    `json:"status"`    // HTTP 状态码
	ElapsedMs int64  `json:"elapsedMs"` // 总耗时 (毫秒)
	Error     string `json:"error,omitempty"`
}

const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 WinPHP/2.0"

func newClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig:       &tls.Config{MinVersion: tls.VersionTLS12},
			Proxy:                 http.ProxyFromEnvironment,
			DisableKeepAlives:     true,
			ResponseHeaderTimeout: timeout,
		},
	}
}

// Test 探测单个 URL. 先 HEAD, 若 405/501 等不支持就降级 GET Range: bytes=0-0.
func Test(ctx context.Context, url string) Result {
	start := time.Now()
	r := Result{URL: url}

	c, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cli := newClient(10 * time.Second)

	headReq, err := http.NewRequestWithContext(c, "HEAD", url, nil)
	if err != nil {
		r.Error = "URL 格式错误: " + err.Error()
		r.ElapsedMs = time.Since(start).Milliseconds()
		return r
	}
	headReq.Header.Set("User-Agent", userAgent)
	headReq.Header.Set("Accept", "*/*")

	resp, err := cli.Do(headReq)
	if err == nil {
		defer resp.Body.Close()
		// 4xx (尤其 405) 时 HEAD 不支持, 用 GET Range 兜底
		if resp.StatusCode == 405 || resp.StatusCode == 501 || resp.StatusCode == 400 {
			return tryRange(c, url, start)
		}
		r.Status = resp.StatusCode
		r.OK = resp.StatusCode >= 200 && resp.StatusCode < 400
		if !r.OK {
			r.Error = httpReason(resp.StatusCode)
		}
		r.ElapsedMs = time.Since(start).Milliseconds()
		return r
	}

	// HEAD 完全失败 (网络层): 直接试 Range
	r2 := tryRange(c, url, start)
	if r2.OK {
		return r2
	}
	r.Error = friendlyErr(err)
	r.ElapsedMs = time.Since(start).Milliseconds()
	return r
}

func tryRange(ctx context.Context, url string, start time.Time) Result {
	r := Result{URL: url}
	cli := newClient(10 * time.Second)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		r.Error = err.Error()
		r.ElapsedMs = time.Since(start).Milliseconds()
		return r
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Range", "bytes=0-0")
	req.Header.Set("Accept", "*/*")
	resp, err := cli.Do(req)
	if err != nil {
		r.Error = friendlyErr(err)
		r.ElapsedMs = time.Since(start).Milliseconds()
		return r
	}
	defer resp.Body.Close()
	r.Status = resp.StatusCode
	r.OK = resp.StatusCode >= 200 && resp.StatusCode < 400
	if !r.OK {
		r.Error = httpReason(resp.StatusCode)
	}
	r.ElapsedMs = time.Since(start).Milliseconds()
	return r
}

// TestMany 并发探测多个 URL, 返回顺序与输入一致.
func TestMany(ctx context.Context, urls []string) []Result {
	out := make([]Result, len(urls))
	var wg sync.WaitGroup
	for i, u := range urls {
		wg.Add(1)
		go func(idx int, url string) {
			defer wg.Done()
			out[idx] = Test(ctx, url)
		}(i, u)
	}
	wg.Wait()
	return out
}

func httpReason(code int) string {
	t := http.StatusText(code)
	if t == "" {
		return ""
	}
	return t
}

func friendlyErr(err error) string {
	s := err.Error()
	low := strings.ToLower(s)
	switch {
	case strings.Contains(low, "no such host"):
		return "DNS 解析失败 (无法解析域名)"
	case strings.Contains(low, "timeout") || strings.Contains(low, "deadline"):
		return "连接超时"
	case strings.Contains(low, "refused"):
		return "连接被拒绝"
	case strings.Contains(low, "x509") || strings.Contains(low, "certificate"):
		return "SSL 证书错误"
	case strings.Contains(low, "network is unreachable"):
		return "网络不可达"
	}
	return s
}
