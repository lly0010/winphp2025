package download

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/lly0010/winphp2025/internal/logger"
)

const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

// Progress callback: bytesDownloaded, totalBytes (-1 if unknown)
type ProgressFn func(downloaded, total int64)

func client() *http.Client {
	return &http.Client{
		Timeout: 0, // 我们用 context 控制
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
			Proxy:           http.ProxyFromEnvironment,
		},
	}
}

// Download 下载单个 URL 到 outFile, 支持进度回调和取消.
func Download(ctx context.Context, url, outFile string, prog ProgressFn) error {
	logger.Info("下载: %s", url)

	if err := os.MkdirAll(filepath.Dir(outFile), 0o755); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9,zh-CN;q=0.8")

	cl := client()
	resp, err := cl.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d %s", resp.StatusCode, resp.Status)
	}

	total := resp.ContentLength

	f, err := os.Create(outFile)
	if err != nil {
		return err
	}
	defer f.Close()

	buf := make([]byte, 64*1024)
	var read int64
	last := time.Now()
	for {
		select {
		case <-ctx.Done():
			os.Remove(outFile)
			return ctx.Err()
		default:
		}
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				return werr
			}
			read += int64(n)
			if prog != nil && time.Since(last) > 150*time.Millisecond {
				prog(read, total)
				last = time.Now()
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return rerr
		}
	}
	if prog != nil {
		prog(read, read)
	}
	logger.Info("下载完成: %s (%.2f MB)", outFile, float64(read)/1024/1024)
	return nil
}

// DownloadWithRetry 多 URL 多重试. 任一 URL 成功即返回.
func DownloadWithRetry(ctx context.Context, urls []string, outFile string, prog ProgressFn, maxRetry int) error {
	if maxRetry < 1 {
		maxRetry = 3
	}
	var last error
	for i, u := range urls {
		for attempt := 1; attempt <= maxRetry; attempt++ {
			logger.Info("尝试源 %d/%d, 重试 %d/%d: %s", i+1, len(urls), attempt, maxRetry, u)
			err := Download(ctx, u, outFile, prog)
			if err == nil {
				return nil
			}
			last = err
			logger.Warn("失败: %v", err)
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if attempt < maxRetry {
				time.Sleep(time.Duration(attempt) * 2 * time.Second)
			}
		}
	}
	return fmt.Errorf("所有 %d 个 URL 均失败. 最后错误: %v", len(urls), last)
}
