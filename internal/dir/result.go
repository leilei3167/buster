package dir

import (
	"bytes"
	"fmt"
	"net/http"
)

type Result struct {
	URL, Path                                      string
	Verbose, Expanded, NoStatus, HideLength, Found bool
	Header                                         http.Header
	StatusCode                                     int
	Size                                           int64
}

// ResulToString 实现result接口,将结果转换为字符串
func (r Result) ResulToString() (string, error) {
	buf := &bytes.Buffer{}

	// Prefix if we're in verbose mode
	if r.Verbose {
		if r.Found {
			if _, err := fmt.Fprintf(buf, "Found: "); err != nil {
				return "", err
			}
		} else {
			if _, err := fmt.Fprintf(buf, "Missed: "); err != nil {
				return "", err
			}
		}
	}

	//是否打印完整url 或者只是相对路径
	if r.Expanded {
		if _, err := fmt.Fprintf(buf, "%s", r.URL); err != nil {
			return "", err
		}
	} else {
		if _, err := fmt.Fprintf(buf, "/"); err != nil {
			return "", err
		}
	}
	if _, err := fmt.Fprintf(buf, "%-20s", r.Path); err != nil {
		return "", err
	}
	if !r.NoStatus {
		if _, err := fmt.Fprintf(buf, " (Status: %d)", r.StatusCode); err != nil {
			return "", err
		}
	}

	if !r.HideLength {
		if _, err := fmt.Fprintf(buf, " [Size: %d]", r.Size); err != nil {
			return "", err
		}
	}

	//location一般是301重定向时会被写入
	location := r.Header.Get("Location")
	if location != "" {
		if _, err := fmt.Fprintf(buf, "[--> %s]", location); err != nil {
			return "", err
		}
	}
	if _, err := fmt.Fprintf(buf, "\n"); err != nil {
		return "", err
	}

	s := buf.String()
	return s, nil
}
