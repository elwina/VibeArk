package apps

// AppSource defines where to check for version updates
type AppSource struct {
	Type               string `yaml:"type"`                  // "web-scrape" | "github" | "direct-download"
	URL                string `yaml:"url,omitempty"`          // URL to scrape for version
	VersionPattern     string `yaml:"version_pattern"`        // Regex to extract version
	DownloadURLPattern string `yaml:"download_url_pattern,omitempty"` // Regex to extract download URL from source page
	Repo               string `yaml:"repo,omitempty"`         // GitHub repo (owner/repo)
	AssetPattern       string `yaml:"asset_pattern"`          // Glob pattern for GitHub assets
}

// AppDownload defines how to download the app
type AppDownload struct {
	Type               string `yaml:"type"`                          // "direct" | "github-release" | "redirect-page"
	URL                string `yaml:"url"`
	RedirectURLPattern string `yaml:"redirect_url_pattern,omitempty"` // Regex to extract real URL from redirect page
	UA                 string `yaml:"ua,omitempty"`                   // Custom User-Agent for download
}

// AppEntry is a complete app definition
type AppEntry struct {
	Name          string      `yaml:"name"`
	ID            string      `yaml:"id"`
	Description   string      `yaml:"description"`
	DescriptionEn string      `yaml:"descriptionEn,omitempty"`
	Source        AppSource   `yaml:"source"`
	Download      AppDownload `yaml:"download"`
	ExePath       string      `yaml:"exe_path"` // Relative to install dir, supports wildcards
	Tags          []string    `yaml:"tags"`
}

// DefaultInstallDir is the default installation directory
const DefaultInstallDir = "D:/ArkApps"

// AppDefinitions contains all hardcoded app definitions
var AppDefinitions = []AppEntry{
	{
		Name:          "Ungoogled Chromium",
		ID:            "chromium",
		Description:   "去 Google 化的 Chromium 浏览器",
		DescriptionEn: "Chromium browser with Google services removed",
		Source: AppSource{
			Type:           "web-scrape",
			URL:            "https://github.com/ungoogled-software/ungoogled-chromium-windows/releases/latest",
			VersionPattern: `releases/tag/(\d+\.\d+\.\d+\.\d+-[^/"]+)`,
		},
		Download: AppDownload{
			Type: "direct",
			URL:  "https://github.com/ungoogled-software/ungoogled-chromium-windows/releases/download/{version}/ungoogled-chromium_{version}_windows_x64.zip",
		},
		ExePath: "chrome.exe",
		Tags:    []string{"browser"},
	},
	{
		Name:          "TreeSize Free",
		ID:            "treesizefree",
		Description:   "磁盘空间分析工具",
		DescriptionEn: "Disk space analyzer",
		Source: AppSource{
			Type:           "web-scrape",
			URL:            "https://www.jam-software.com/treesize_free/changes.shtml",
			VersionPattern: `Version\s+([\d.]+)`,
		},
		Download: AppDownload{
			Type: "direct",
			URL:  "https://downloads.jam-software.de/treesize_free/TreeSizeFree-Portable.zip?language=EN",
		},
		ExePath: "TreeSizeFree.exe",
		Tags:    []string{"system", "disk"},
	},
	{
		Name:          "Geek Uninstaller",
		ID:            "geek",
		Description:   "轻量卸载工具",
		DescriptionEn: "Lightweight uninstaller",
		Source: AppSource{
			Type:           "web-scrape",
			URL:            "https://geekuninstaller.com/download",
			VersionPattern: `<b>([\d.]+)</b>`,
		},
		Download: AppDownload{
			Type: "direct",
			URL:  "https://geekuninstaller.com/geek.zip",
		},
		ExePath: "geek.exe",
		Tags:    []string{"system", "uninstall"},
	},
	{
		Name:          "ScreenToGif",
		ID:            "screentogif",
		Description:   "屏幕录制转 GIF 工具",
		DescriptionEn: "Screen recorder to GIF",
		Source: AppSource{
			Type:           "web-scrape",
			URL:            "https://github.com/NickeManarin/ScreenToGif/releases/latest",
			VersionPattern: `releases/tag/(\d+\.\d+\.\d+)`,
		},
		Download: AppDownload{
			Type: "direct",
			URL:  "https://github.com/NickeManarin/ScreenToGif/releases/download/{version}/ScreenToGif.{version}.Portable.x64.zip",
		},
		ExePath: "ScreenToGif.exe",
		Tags:    []string{"media", "gif"},
	},
	{
		Name:          "Rufus",
		ID:            "rufus",
		Description:   "USB 启动盘制作工具",
		DescriptionEn: "USB bootable drive creator",
		Source: AppSource{
			Type:           "web-scrape",
			URL:            "https://mirror.nju.edu.cn/github-release/pbatard/rufus/",
			VersionPattern: `href="[^"]*Rufus(?:%20|\s)+([\d.]+)/"`,
		},
		Download: AppDownload{
			Type: "direct",
			URL:  "https://mirror.nju.edu.cn/github-release/pbatard/rufus/LatestRelease/rufus-{version}p.exe",
		},
		ExePath: "rufus-*.exe",
		Tags:    []string{"system", "usb"},
	},
	{
		Name:          "PeaZip",
		ID:            "peazip",
		Description:   "开源压缩工具",
		DescriptionEn: "Open source file archiver",
		Source: AppSource{
			Type:           "web-scrape",
			URL:            "https://github.com/peazip/PeaZip/releases/latest",
			VersionPattern: `releases/tag/(\d+\.\d+\.\d+)`,
		},
		Download: AppDownload{
			Type: "direct",
			URL:  "https://github.com/peazip/PeaZip/releases/download/{version}/peazip_portable-{version}.WIN64.zip",
		},
		ExePath: "PeaZip.exe",
		Tags:    []string{"system", "compress"},
	},
	{
		Name:          "imFile",
		ID:            "imfile",
		Description:   "下载管理器",
		DescriptionEn: "Download manager",
		Source: AppSource{
			Type:           "web-scrape",
			URL:            "https://github.com/imfile-io/imfile-desktop/releases/latest",
			VersionPattern: `releases/tag/v?(\d+\.\d+\.\d+)`,
		},
		Download: AppDownload{
			Type: "direct",
			URL:  "https://github.com/imfile-io/imfile-desktop/releases/download/v{version}/imFile-{version}-x64.zip",
		},
		ExePath: "imFile.exe",
		Tags:    []string{"network", "download"},
	},
	{
		Name:          "ShareX",
		ID:            "sharex",
		Description:   "屏幕截图与录制工具",
		DescriptionEn: "Screen capture and recording tool",
		Source: AppSource{
			Type:           "web-scrape",
			URL:            "https://github.com/ShareX/ShareX/releases/latest",
			VersionPattern: `releases/tag/v?(\d+\.\d+\.\d+)`,
		},
		Download: AppDownload{
			Type: "direct",
			URL:  "https://github.com/ShareX/ShareX/releases/download/v{version}/ShareX-{version}-portable-x64.zip",
		},
		ExePath: "ShareX.exe",
		Tags:    []string{"media", "screenshot"},
	},
	{
		Name:          "v2rayN",
		ID:            "v2rayn",
		Description:   "代理客户端",
		DescriptionEn: "Proxy client",
		Source: AppSource{
			Type:           "web-scrape",
			URL:            "https://github.com/2dust/v2rayN/releases/latest",
			VersionPattern: `releases/tag/(\d+\.\d+)`,
		},
		Download: AppDownload{
			Type: "direct",
			URL:  "https://v2rayn.2dust.link/v2rayN-windows-64.zip",
		},
		ExePath: "v2rayN.exe",
		Tags:    []string{"network", "proxy"},
	},
	{
		Name:          "Notepad3",
		ID:            "notepad3",
		Description:   "轻量文本编辑器",
		DescriptionEn: "Lightweight text editor",
		Source: AppSource{
			Type:           "web-scrape",
			URL:            "https://github.com/rizonesoft/Notepad3/releases/latest",
			VersionPattern: `releases/tag/RELEASE_(\d+\.\d+\.\d+\.\d+)`,
		},
		Download: AppDownload{
			Type: "direct",
			URL:  "https://github.com/rizonesoft/Notepad3/releases/download/RELEASE_{version}/Notepad3_{version}_x64_Portable.zip",
		},
		ExePath: "Notepad3.exe",
		Tags:    []string{"editor", "text"},
	},
	{
		Name:          "qView",
		ID:            "qview",
		Description:   "轻量图片浏览器",
		DescriptionEn: "Lightweight image viewer",
		Source: AppSource{
			Type:           "web-scrape",
			URL:            "https://github.com/jurplel/qView/releases/latest",
			VersionPattern: `releases/tag/(\d+(?:\.\d+)*)`,
		},
		Download: AppDownload{
			Type: "direct",
			URL:  "https://github.com/jurplel/qView/releases/download/{version}/qView-{version}-win64.zip",
		},
		ExePath: "qView.exe",
		Tags:    []string{"media", "image"},
	},
	{
		Name:          "DiskGenius",
		ID:            "diskgenius",
		Description:   "硬盘分区与数据恢复软件",
		DescriptionEn: "Disk partition and data recovery tool",
		Source: AppSource{
			Type:           "web-scrape",
			URL:            "https://www.diskgenius.cn/download.php",
			VersionPattern: `V(\d+\.\d+\.\d+\.\d+)`,
		},
		Download: AppDownload{
			Type:               "redirect-page",
			URL:                "https://www.diskgenius.cn/download/downloadURL.php?Name=DG_64",
			RedirectURLPattern: `'(https://[^"']*download[^"']*\.zip)'`,
		},
		ExePath: "DiskGenius.exe",
		Tags:    []string{"system", "disk"},
	},
	{
		Name:          "VLC Media Player",
		ID:            "vlc",
		Description:   "开源多媒体播放器",
		DescriptionEn: "Open source multimedia player",
		Source: AppSource{
			Type:           "web-scrape",
			URL:            "https://www.videolan.org/vlc/download-windows.html",
			VersionPattern: `vlc-(\d+\.\d+\.\d+)`,
		},
		Download: AppDownload{
			Type: "direct",
			URL:  "https://get.videolan.org/vlc/{version}/win64/vlc-{version}-win64.zip",
		},
		ExePath: "vlc.exe",
		Tags:    []string{"media", "player"},
	},
	{
		Name:          "FileZilla",
		ID:            "filezilla",
		Description:   "开源 FTP/FTPS/SFTP 客户端",
		DescriptionEn: "Open source FTP/FTPS/SFTP client",
		Source: AppSource{
			Type:           "web-scrape",
			URL:            "https://community.chocolatey.org/api/v2/Packages()?$filter=tolower(Id)%20eq%20%27filezilla%27&$orderby=Version%20desc&$top=1",
			VersionPattern: `<d:Version>([^<]+)</d:Version>`,
		},
		Download: AppDownload{
			Type: "direct",
			URL:  "https://dl4.cdn.filezilla-project.org/client/FileZilla_{version}_win64.zip",
			UA:   "FileZilla/3.0.0",
		},
		ExePath: "filezilla.exe",
		Tags:    []string{"network", "ftp"},
	},
}
