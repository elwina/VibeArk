package i18n

import (
	"fmt"
	"os"
	"syscall"
)

type Lang string

const (
	ZH Lang = "zh"
	EN Lang = "en"
)

var current Lang = ZH

func T(key string) string {
	if m, ok := translations[key]; ok {
		if s, ok := m[current]; ok && s != "" {
			return s
		}
		if s, ok := m[ZH]; ok && s != "" {
			return s
		}
	}
	return key
}

func Tf(key string, args ...interface{}) string {
	return fmt.Sprintf(T(key), args...)
}

func SetLang(l Lang) { current = l }
func CurrentLang() Lang { return current }

func ToggleLang() Lang {
	if current == ZH {
		current = EN
	} else {
		current = ZH
	}
	return current
}

func DetectLang() Lang {
	if os.Getenv("VIBEARK_LANG") == "zh" {
		return ZH
	}
	if os.Getenv("VIBEARK_LANG") == "en" {
		return EN
	}

	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	proc := kernel32.NewProc("GetUserDefaultUILanguage")
	langID, _, _ := proc.Call()
	switch uint16(langID) {
	case 0x0804:
		return ZH
	}
	return EN
}

var translations = map[string]map[Lang]string{
	"title":              {ZH: "便携应用管理器", EN: "Portable App Manager"},
	"check_done":         {ZH: "检查完成", EN: "Check complete"},
	"install_success":    {ZH: "安装成功", EN: "Installed"},
	"download_cancelled": {ZH: "下载已取消", EN: "Download cancelled"},
	"already_latest":     {ZH: "已是最新", EN: "Already latest"},
	"proxy_on":           {ZH: "代理已开启", EN: "Proxy enabled"},
	"proxy_off":          {ZH: "代理已关闭", EN: "Proxy disabled"},
	"ghproxy_on":         {ZH: "GitHub加速已开启", EN: "GitHub proxy enabled"},
	"ghproxy_off":        {ZH: "GitHub加速已关闭", EN: "GitHub proxy disabled"},
	"refreshed":          {ZH: "已刷新", EN: "Refreshed"},
	"settings_saved":     {ZH: "设置已保存", EN: "Settings saved"},
	"no_app_selected":    {ZH: "未选中应用", EN: "No app selected"},
	"check_all_done":     {ZH: "全部检查完成", EN: "All checks complete"},
	"exe_not_found":      {ZH: "未找到 exe", EN: "Exe not found"},
	"launched":           {ZH: "已启动", EN: "Launched"},
	"deleted":            {ZH: "已删除", EN: "Deleted"},
	"category_official":  {ZH: "── 官方网站 ──", EN: "── Official ──"},
	"category_mirror":    {ZH: "── 镜像站 ──", EN: "── Mirror ──"},
	"category_github":    {ZH: "── GitHub ──", EN: "── GitHub ──"},
	"not_installed":      {ZH: " (未安装)", EN: " (not installed)"},
	"checking_prefix":    {ZH: "⟳ 检查中: ", EN: "⟳ Checking: "},
	"checking_dots":      {ZH: "⟳ 检查中...", EN: "⟳ Checking..."},
	"waiting":            {ZH: "等待中...", EN: "Waiting..."},
	"extracting":         {ZH: "📦 解压中...", EN: "📦 Extracting..."},
	"any_key_close":      {ZH: " (任意键关闭)", EN: " (any key to close)"},

	"help_line1":       {ZH: "w/s:导航  Enter/d:安装  o:打开  a/u:检查  Ctrl+U:检查全部", EN: "w/s:Nav  Enter/d:Install  o:Open  a/u:Check  Ctrl+U:Check All"},
	"help_line2":       {ZH: "x:删除  p:代理  g:GitHub加速  r:刷新", EN: "x:Delete  p:Proxy  g:GitHub Proxy  r:Refresh"},
	"help_line3":       {ZH: "Esc:取消  Ctrl+P:设置  Ctrl+L:语言  q:退出", EN: "Esc:Cancel  Ctrl+P:Settings  Ctrl+L:Lang  q:Quit"},
	"proxy_not_set":    {ZH: "未设置", EN: "Not set"},
	"proxy_disabled":   {ZH: "已关闭", EN: "Disabled"},
	"proxy_enabled":    {ZH: "开启", EN: "Enabled"},
	"info_status":      {ZH: "目录: %s  代理: %s  GitHub加速: %s", EN: "Dir: %s  Proxy: %s  GitHub Proxy: %s"},
	"settings_title":   {ZH: "设置", EN: "Settings"},
	"settings_install_dir":  {ZH: "安装目录", EN: "Install Dir"},
	"settings_proxy_port":   {ZH: "代理端口", EN: "Proxy Port"},
	"settings_gh_proxy":     {ZH: "GitHub加速", EN: "GitHub Proxy"},
	"settings_none":          {ZH: "(无)", EN: "(none)"},
	"settings_enter_confirm": {ZH: "Enter:确认  Esc:取消", EN: "Enter:Confirm  Esc:Cancel"},
	"settings_nav":           {ZH: "w/s:导航  Enter:编辑  Esc/q:返回", EN: "w/s:Nav  Enter:Edit  Esc/q:Back"},
	"settings_proxy_hint":    {ZH: "代理只需填端口，如 10808", EN: "Proxy: port only, e.g. 10808"},
	"settings_gh_hint":       {ZH: "GitHub加速: 填前缀如 https://gh-proxy.org/", EN: "GitHub: prefix, e.g. https://gh-proxy.org/"},

	"dl_fetch_page_fail":  {ZH: "获取下载页面失败: %v", EN: "Failed to fetch download page: %v"},
	"dl_no_link_found":    {ZH: "未找到下载链接: %v", EN: "Download link not found: %v"},
	"dl_redirect_fail":    {ZH: "获取跳转页面失败: %v", EN: "Failed to fetch redirect page: %v"},
	"dl_no_download_url":  {ZH: "没有可用的下载地址", EN: "No download URL available"},
	"dl_lanzou_fail":      {ZH: "蓝奏链接解析失败: %v", EN: "Lanzou link parse failed: %v"},
	"dl_extract_fail":     {ZH: "解压失败: %v", EN: "Extract failed: %v"},
	"dl_invalid_bytes":    {ZH: "收到 %d 字节，不是有效文件", EN: "Received %d bytes, not a valid file"},
	"dl_html_not_file":    {ZH: "收到 HTML 页面，不是下载文件", EN: "Received HTML page, not a download file"},

	"lz_parse_url_fail":     {ZH: "解析蓝奏URL失败: %v", EN: "Failed to parse Lanzou URL: %v"},
	"lz_fetch_page_fail":    {ZH: "获取蓝奏页面失败: %v", EN: "Failed to fetch Lanzou page: %v"},
	"lz_no_iframe":          {ZH: "蓝奏页面未找到下载iframe", EN: "Lanzou page: no download iframe found"},
	"lz_fetch_iframe_fail":  {ZH: "获取蓝奏iframe失败: %v", EN: "Failed to fetch Lanzou iframe: %v"},
	"lz_no_sign":            {ZH: "蓝奏iframe未找到签名参数", EN: "Lanzou iframe: no sign params found"},
	"lz_request_fail":       {ZH: "蓝奏请求失败: %v", EN: "Lanzou request failed: %v"},
	"lz_parse_result_fail":  {ZH: "蓝奏返回解析失败: %s", EN: "Lanzou response parse failed: %s"},
	"lz_unknown_error":      {ZH: "未知错误", EN: "Unknown error"},
	"lz_error_return":       {ZH: "蓝奏返回错误: %s", EN: "Lanzou error: %s"},
	"lz_no_download_url":    {ZH: "蓝奏返回无下载地址: %s", EN: "Lanzou: no download URL: %s"},
}
