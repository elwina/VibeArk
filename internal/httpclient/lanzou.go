package httpclient

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"vibeark/internal/i18n"
)

var (
	lanzouIframeRE     = regexp.MustCompile(`<iframe[^>]+src="(/fn\?[^"]+)"`)
	lanzouFileIDRE     = regexp.MustCompile(`/q/jb/\?f=(\d+)`)
	lanzouAjaxFileIDRE = regexp.MustCompile(`/ajaxm\.php\?file=(\d+)`)
	lanzouWpSignRE     = regexp.MustCompile(`var\s+wp_sign\s*=\s*'([^']+)'`)
	lanzouAjaxDataRE   = regexp.MustCompile(`var\s+ajaxdata\s*=\s*'([^']+)'`)
	lanzouKdnsRE       = regexp.MustCompile(`var\s+kdns\s*=\s*(\d+)`)
)

const browserUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

// IsLanzouURL checks if URL is a Lanzou cloud download link.
func IsLanzouURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	return strings.Contains(u.Host, "lanzou")
}

// ResolveLanzou resolves a Lanzou cloud short/redirect URL to the actual download URL.
func (c *Client) ResolveLanzou(lanzouURL string) (string, error) {
	u, err := url.Parse(lanzouURL)
	if err != nil {
		return "", fmt.Errorf("%s: %v", i18n.T("lz_parse_url_fail"), err)
	}
	domain := fmt.Sprintf("%s://%s", u.Scheme, u.Host)

	pageBody, err := c.getWithUA(lanzouURL, lanzouURL)
	if err != nil {
		return "", fmt.Errorf("%s: %v", i18n.T("lz_fetch_page_fail"), err)
	}

	iframeMatch := lanzouIframeRE.FindStringSubmatch(pageBody)
	if len(iframeMatch) < 2 {
		return "", fmt.Errorf("%s", i18n.T("lz_no_iframe"))
	}
	iframeURL := domain + iframeMatch[1]

	fileID := ""
	if m := lanzouFileIDRE.FindStringSubmatch(pageBody); len(m) >= 2 {
		fileID = m[1]
	}

	iframeBody, err := c.getWithUA(iframeURL, lanzouURL)
	if err != nil {
		return "", fmt.Errorf("%s: %v", i18n.T("lz_fetch_iframe_fail"), err)
	}

	wpSign := ""
	if m := lanzouWpSignRE.FindStringSubmatch(iframeBody); len(m) >= 2 {
		wpSign = m[1]
	}
	ajaxData := ""
	if m := lanzouAjaxDataRE.FindStringSubmatch(iframeBody); len(m) >= 2 {
		ajaxData = m[1]
	}
	kdns := "1"
	if m := lanzouKdnsRE.FindStringSubmatch(iframeBody); len(m) >= 2 {
		kdns = m[1]
	}

	if wpSign == "" {
		return "", fmt.Errorf("%s", i18n.T("lz_no_sign"))
	}

	if fileID == "" {
		if m := lanzouAjaxFileIDRE.FindStringSubmatch(iframeBody); len(m) >= 2 {
			fileID = m[1]
		}
	}

	formData := fmt.Sprintf("action=downprocess&sign=%s&signs=%s&websignkey=%s&websign=&kd=%s&ves=1",
		wpSign, ajaxData, ajaxData, kdns)

	ajaxURL := domain + "/ajaxm.php"
	if fileID != "" {
		ajaxURL += "?file=" + fileID
	}

	req, err := http.NewRequest("POST", ajaxURL, strings.NewReader(formData))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", iframeURL)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("User-Agent", browserUA)
	req.Header.Set("Origin", domain)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("%s: %v", i18n.T("lz_request_fail"), err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result struct {
		Zt  interface{} `json:"zt"`
		Dom string      `json:"dom"`
		URL string      `json:"url"`
		Inf string      `json:"inf"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("%s: %s", i18n.T("lz_parse_result_fail"), string(respBody))
	}

	ztStr := fmt.Sprintf("%v", result.Zt)
	if ztStr != "1" {
		info := result.Inf
		if info == "" {
			info = i18n.T("lz_unknown_error")
		}
		return "", fmt.Errorf("%s: %s", i18n.T("lz_error_return"), info)
	}

	downDomain := result.Dom
	if kdns == "0" {
		downDomain = "https://slssctm.dmpdmp.com"
	}

	if result.URL != "" {
		if downDomain != "" && !strings.HasPrefix(result.URL, "http") {
			return downDomain + "/file/" + result.URL, nil
		}
		if !strings.HasPrefix(result.URL, "http") {
			if strings.HasPrefix(result.URL, "/") {
				return domain + result.URL, nil
			}
			return domain + "/" + result.URL, nil
		}
		return result.URL, nil
	}

	return "", fmt.Errorf("%s: %s", i18n.T("lz_no_download_url"), string(respBody))
}

func (c *Client) getWithUA(urlStr, referer string) (string, error) {
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", browserUA)
	if referer != "" {
		req.Header.Set("Referer", referer)
	}
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
