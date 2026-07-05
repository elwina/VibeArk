package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"vibeark/internal/apps"
	"vibeark/internal/config"
	"vibeark/internal/downloader"
	"vibeark/internal/scanner"
	"vibeark/internal/updater"
)

var rootCmd = &cobra.Command{
	Use:   "vibeark",
	Short: "VibeArk — 便携应用管理器",
	Long:  "VibeArk — 便携应用管理器\n开发者 Elwina Vardal",
	Run: func(cmd *cobra.Command, args []string) {
		// Default: launch TUI
		launchTUI()
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有应用",
	Run: func(cmd *cobra.Command, args []string) {
		settings := config.LoadSettings()
		appDefs := apps.AppDefinitions
		scanned := scanner.ScanAll(appDefs, settings.InstallDir)
		lock := config.LoadLock(settings.InstallDir)

		fmt.Println("Installed apps:")
		fmt.Println("")
		for i, s := range scanned {
			app := appDefs[i]
			icon := "○"
			if s.Status == "installed" {
				icon = "✓"
			}

			version := "---"
			if entry, ok := lock.Entries[app.ID]; ok && entry.InstalledVersion != "" {
				version = entry.InstalledVersion
			}

			fmt.Printf("  %s %-20s %-15s %s\n", icon, app.Name, version, s.Status)
		}
	},
}

var checkCmd = &cobra.Command{
	Use:   "check [id]",
	Short: "检查更新",
	Run: func(cmd *cobra.Command, args []string) {
		settings := config.LoadSettings()
		appDefs := apps.AppDefinitions
		lock := config.LoadLock(settings.InstallDir)

		fmt.Println("Checking for updates...")
		fmt.Println("")

		for _, app := range appDefs {
			if len(args) > 0 && args[0] != app.ID {
				continue
			}

			fmt.Printf("  %-20s ", app.Name)

			// Skip direct-download type apps
			if app.Source.Type == "direct-download" {
				s := scanner.ScanApp(app, settings.InstallDir)
				if s.Status == "installed" {
					fmt.Println("installed (no version check)")
				} else {
					fmt.Println("not installed (direct download)")
				}
				continue
			}

			result := updater.CheckVersion(app, settings.Proxy)

			if result.Error != nil {
				fmt.Printf("Error: %v\n", result.Error)
				continue
			}

			if result.RemoteVersion != "" {
				localVersion := ""
				if entry, ok := lock.Entries[app.ID]; ok {
					localVersion = entry.InstalledVersion
				}

				s := scanner.ScanApp(app, settings.InstallDir)
				if s.Status == "installed" {
					if localVersion != "" && updater.CompareVersions(localVersion, result.RemoteVersion) < 0 {
						fmt.Printf("%s → %s (update available)\n", localVersion, result.RemoteVersion)
					} else {
						fmt.Printf("%s (up to date)\n", localVersion)
					}
				} else {
					fmt.Printf("Remote: %s (not installed)\n", result.RemoteVersion)
				}

				config.UpdateLockEntry(lock, app.ID, config.LockEntry{
					LastChecked: config.NowUTC(),
				})

				if entry, ok := lock.Entries[app.ID]; ok && entry.Status == "installed" && entry.InstalledVersion == "" {
					config.UpdateLockEntry(lock, app.ID, config.LockEntry{
						InstalledVersion: result.RemoteVersion,
					})
				}
			} else {
				fmt.Println("No version info available")
			}
		}

		config.SaveLock(settings.InstallDir, lock)
	},
}

var installCmd = &cobra.Command{
	Use:   "install <id>",
	Short: "下载安装应用",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		settings := config.LoadSettings()
		appDefs := apps.AppDefinitions
		lock := config.LoadLock(settings.InstallDir)

		targetID := args[0]
		var targetApp *apps.AppEntry
		for _, app := range appDefs {
			if app.ID == targetID {
				targetApp = &app
				break
			}
		}

		if targetApp == nil {
			fmt.Fprintf(os.Stderr, "App not found: %s\n", targetID)
			os.Exit(1)
		}

		fmt.Printf("Checking remote version...\n")
		updateResult := updater.CheckVersion(*targetApp, settings.Proxy)
		remoteVersion := updateResult.RemoteVersion

		if updateResult.Error != nil {
			fmt.Fprintf(os.Stderr, "  Version check failed: %v\n", updateResult.Error)
		} else if remoteVersion != "" {
			fmt.Printf("  Remote version: %s\n", remoteVersion)
		}

		fmt.Printf("Installing %s...\n", targetApp.Name)
		lastPercent := -1

		result := downloader.DownloadAndInstall(
			*targetApp,
			settings.InstallDir,
			remoteVersion,
			&settings,
			func(transferred, total int64, percent float64) {
				p := int(percent)
				if p != lastPercent {
					lastPercent = p
					if total > 0 {
						fmt.Printf("\r  %d%% (%.1fMB/%.1fMB)", p, float64(transferred)/1024/1024, float64(total)/1024/1024)
					} else {
						fmt.Printf("\r  %.1fMB", float64(transferred)/1024/1024)
					}
				}
			},
			func(status string, url string) {
				if status == "downloading" && url != "" {
					fmt.Printf("\n  URL: %s\n", url)
				}
				if status == "extracting" {
					fmt.Println("\n  Extracting...")
				}
			},
		)
		fmt.Println()

		if result.Success {
			fmt.Printf("Done! %s installed.\n", targetApp.Name)
			if result.ExePath != "" {
				fmt.Printf("  Exe: %s\n", result.ExePath)
			}

			config.UpdateLockEntry(lock, targetApp.ID, config.LockEntry{
				Status:           "installed",
				InstalledVersion: result.Version,
				LastDownloaded:   config.NowUTC(),
				InstalledAt:      config.NowUTC(),
				ExePath:          result.ExePath,
			})

			// Show updated list
			fmt.Println("")
			updatedScanned := scanner.ScanAll(appDefs, settings.InstallDir)
			updatedLock := config.LoadLock(settings.InstallDir)
			for i, s := range updatedScanned {
				app := appDefs[i]
				icon := "○"
				if s.Status == "installed" {
					icon = "✓"
				}
				version := "---"
				if entry, ok := updatedLock.Entries[app.ID]; ok && entry.InstalledVersion != "" {
					version = entry.InstalledVersion
				}
				fmt.Printf("  %s %-20s %-15s %s\n", icon, app.Name, version, s.Status)
			}
		} else {
			fmt.Fprintf(os.Stderr, "Failed: %s\n", result.Error)
			config.UpdateLockEntry(lock, targetApp.ID, config.LockEntry{
				Status: "error",
				Error:  result.Error,
			})
		}

		config.SaveLock(settings.InstallDir, lock)
	},
}

var updateCmd = &cobra.Command{
	Use:   "update [id]",
	Short: "更新应用",
	Run: func(cmd *cobra.Command, args []string) {
		settings := config.LoadSettings()
		appDefs := apps.AppDefinitions
		lock := config.LoadLock(settings.InstallDir)

		for _, app := range appDefs {
			if len(args) > 0 && args[0] != app.ID {
				continue
			}

			s := scanner.ScanApp(app, settings.InstallDir)
			if s.Status != "installed" {
				fmt.Printf("  %s: not installed, skipping\n", app.Name)
				continue
			}

			fmt.Printf("  %-20s checking... ", app.Name)
			result := updater.CheckVersion(app, settings.Proxy)

			if result.Error != nil {
				fmt.Printf("Error: %v\n", result.Error)
				continue
			}

			lockEntry := lock.Entries[app.ID]
			localVersion := ""
			if lockEntry != nil {
				localVersion = lockEntry.InstalledVersion
			}

			if result.RemoteVersion != "" && localVersion != "" && updater.CompareVersions(localVersion, result.RemoteVersion) < 0 {
				fmt.Printf("%s → %s, downloading...\n", localVersion, result.RemoteVersion)
				lastPercent := -1
				dl := downloader.DownloadAndInstall(app, settings.InstallDir, result.RemoteVersion, &settings, func(transferred, total int64, percent float64) {
					p := int(percent)
					if p != lastPercent {
						lastPercent = p
						if total > 0 {
							fmt.Printf("\r    %d%% (%.1fMB/%.1fMB)", p, float64(transferred)/1024/1024, float64(total)/1024/1024)
						} else {
							fmt.Printf("\r    %.1fMB", float64(transferred)/1024/1024)
						}
					}
				}, func(status string, url string) {
					if status == "downloading" && url != "" {
						fmt.Printf("\n    URL: %s\n", url)
					}
					if status == "extracting" {
						fmt.Println("\n    Extracting...")
					}
				})
				fmt.Println()
				if dl.Success {
					fmt.Println("  Updated!")
					config.UpdateLockEntry(lock, app.ID, config.LockEntry{
						InstalledVersion: result.RemoteVersion,
						LastDownloaded:   config.NowUTC(),
						LastChecked:      config.NowUTC(),
						ExePath:          dl.ExePath,
					})
				} else {
					fmt.Printf("  Update failed: %s\n", dl.Error)
				}
			} else {
				fmt.Printf("up to date (%s)\n", localVersion)
				config.UpdateLockEntry(lock, app.ID, config.LockEntry{
					LastChecked: config.NowUTC(),
				})
			}
		}

		config.SaveLock(settings.InstallDir, lock)
	},
}

var deleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "删除应用",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		settings := config.LoadSettings()
		appDefs := apps.AppDefinitions
		lock := config.LoadLock(settings.InstallDir)

		targetID := args[0]
		var targetApp *apps.AppEntry
		for _, app := range appDefs {
			if app.ID == targetID {
				targetApp = &app
				break
			}
		}

		if targetApp == nil {
			fmt.Fprintf(os.Stderr, "App not found: %s\n", targetID)
			os.Exit(1)
		}

		appDir := fmt.Sprintf("%s/%s", settings.InstallDir, targetApp.ID)
		if err := os.RemoveAll(appDir); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		config.UpdateLockEntry(lock, targetApp.ID, config.LockEntry{
			Status:           "not-found",
			InstalledVersion: "",
			ExePath:          "",
		})
		config.SaveLock(settings.InstallDir, lock)

		fmt.Printf("%s 已删除\n", targetApp.Name)
	},
}

var dirCmd = &cobra.Command{
	Use:   "dir [path]",
	Short: "查看/设置安装目录",
	Run: func(cmd *cobra.Command, args []string) {
		settings := config.LoadSettings()

		if len(args) > 0 {
			settings.InstallDir = args[0]
			config.SaveSettings(settings)
			fmt.Printf("Install directory set to: %s\n", settings.InstallDir)
		} else {
			fmt.Printf("Current install directory: %s\n", settings.InstallDir)
		}
	},
}

var proxyCmd = &cobra.Command{
	Use:   "proxy [port]",
	Short: "查看/设置代理端口",
	Run: func(cmd *cobra.Command, args []string) {
		settings := config.LoadSettings()

		if len(args) > 0 {
			settings.Proxy = config.NormalizeProxy(args[0])
			config.SaveSettings(settings)
			if settings.Proxy == "" {
				fmt.Println("Proxy set to: (none)")
			} else {
				fmt.Printf("Proxy set to: %s\n", settings.Proxy)
			}
		} else {
			if settings.Proxy == "" {
				fmt.Println("Current proxy: (none)")
			} else {
				fmt.Printf("Current proxy: %s\n", settings.Proxy)
			}
		}
	},
}

var ghproxyCmd = &cobra.Command{
	Use:   "ghproxy [url]",
	Short: "查看/设置 GitHub Release 加速前缀",
	Run: func(cmd *cobra.Command, args []string) {
		settings := config.LoadSettings()

		if len(args) > 0 {
			settings.GhProxy = args[0]
			config.SaveSettings(settings)
			if settings.GhProxy == "" {
				fmt.Println("GitHub proxy set to: (none)")
			} else {
				fmt.Printf("GitHub proxy set to: %s\n", settings.GhProxy)
			}
		} else {
			if settings.GhProxy == "" {
				fmt.Println("Current GitHub proxy: (none)")
			} else {
				fmt.Printf("Current GitHub proxy: %s\n", settings.GhProxy)
			}
		}
	},
}

var lockCmd = &cobra.Command{
	Use:   "lock",
	Short: "查看锁文件",
	Run: func(cmd *cobra.Command, args []string) {
		settings := config.LoadSettings()
		lock := config.LoadLock(settings.InstallDir)
		appDefs := apps.AppDefinitions

		fmt.Printf("Lock file: %s/vibeark-lock.yaml\n", settings.InstallDir)
		fmt.Printf("Last updated: %s\n", lock.UpdatedAt)
		fmt.Println("")

		for id, entry := range lock.Entries {
			appName := id
			for _, app := range appDefs {
				if app.ID == id {
					appName = app.Name
					break
				}
			}

			version := "---"
			if entry.InstalledVersion != "" {
				version = entry.InstalledVersion
			}

			fmt.Printf("  %-20s status=%s version=%s\n", appName, entry.Status, version)
			if entry.LastChecked != "" {
				fmt.Printf("    last checked: %s\n", entry.LastChecked)
			}
			if entry.LastDownloaded != "" {
				fmt.Printf("    last downloaded: %s\n", entry.LastDownloaded)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(dirCmd)
	rootCmd.AddCommand(proxyCmd)
	rootCmd.AddCommand(ghproxyCmd)
	rootCmd.AddCommand(lockCmd)
}

func Execute() error {
	return rootCmd.Execute()
}
