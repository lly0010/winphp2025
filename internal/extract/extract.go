package extract

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Zip 解压 zip 到 dest. 如果 rootInZip 非空, 把该子目录的内容直接放到 dest 下.
// (例如 rootInZip="nginx-1.27.3", 则 zip 内的 nginx-1.27.3/conf/... 变成 dest/conf/...)
func Zip(zipPath, dest, rootInZip string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	// 准备 dest (先清空, 全新解压)
	if err := os.RemoveAll(dest); err != nil {
		return err
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}

	// 如果未指定 rootInZip, 自动探测: zip 内顶层是否只有一个目录, 是则用它做 root
	if rootInZip == "" {
		topLevels := map[string]bool{}
		for _, f := range r.File {
			parts := strings.SplitN(strings.ReplaceAll(f.Name, "\\", "/"), "/", 2)
			if parts[0] != "" {
				topLevels[parts[0]] = true
			}
		}
		if len(topLevels) == 1 {
			for k := range topLevels {
				// 仅当包内顶级是目录时才认作 root
				for _, f := range r.File {
					name := strings.ReplaceAll(f.Name, "\\", "/")
					if name == k+"/" || strings.HasPrefix(name, k+"/") {
						rootInZip = k
						break
					}
				}
			}
		}
	}

	root := strings.ReplaceAll(rootInZip, "\\", "/")
	if root != "" && !strings.HasSuffix(root, "/") {
		root += "/"
	}

	for _, f := range r.File {
		name := strings.ReplaceAll(f.Name, "\\", "/")
		if root != "" {
			if !strings.HasPrefix(name, root) {
				continue
			}
			name = strings.TrimPrefix(name, root)
		}
		if name == "" {
			continue
		}
		// 防 zip slip
		outPath := filepath.Join(dest, filepath.FromSlash(name))
		if !strings.HasPrefix(outPath, filepath.Clean(dest)+string(os.PathSeparator)) && outPath != dest {
			return fmt.Errorf("非法路径: %s", f.Name)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(outPath, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return err
		}
		fr, err := f.Open()
		if err != nil {
			return err
		}
		fw, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
		if err != nil {
			fr.Close()
			return err
		}
		_, cerr := io.Copy(fw, fr)
		fr.Close()
		fw.Close()
		if cerr != nil {
			return cerr
		}
	}
	return nil
}
