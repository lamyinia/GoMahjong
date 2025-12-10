package main

import (
	"fmt"
	"image/color"
	"sync"
	"time"

	"github.com/charmbracelet/log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ChineseTheme 支持中文的自定义主题
type ChineseTheme struct{}

func (t *ChineseTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	return theme.DefaultTheme().Color(name, variant)
}

func (t *ChineseTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *ChineseTheme) Font(style fyne.TextStyle) fyne.Resource {
	// 在 Windows 上使用系统字体支持中文
	return theme.DefaultTheme().Font(style)
}

func (t *ChineseTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}

// PlayerWindow 代表一个玩家的 GUI 窗口
type PlayerWindow struct {
	userID       string
	client       *TestClient
	window       fyne.Window
	logLabel     *widget.Label
	commandEntry *widget.Entry
	statusLabel  *widget.Label
	logMutex     sync.Mutex
	logLines     []string
}

// NewPlayerWindow 创建一个新的玩家窗口
func NewPlayerWindow(fyneApp fyne.App, userID string) *PlayerWindow {
	pw := &PlayerWindow{
		userID:   userID,
		logLines: make([]string, 0),
	}

	// 创建窗口
	pw.window = fyneApp.NewWindow(fmt.Sprintf("GoMahjong - %s", userID))
	pw.window.SetTitle(fmt.Sprintf("GoMahjong - %s", userID))

	// 创建 UI 组件
	pw.createUI()

	// 设置窗口关闭时的回调
	pw.window.SetOnClosed(func() {
		if pw.client != nil {
			pw.client.Close()
		}
	})

	return pw
}

// createUI 创建 UI 组件
func (pw *PlayerWindow) createUI() {
	// 状态标签
	pw.statusLabel = widget.NewLabel("Status: Disconnected")
	pw.statusLabel.Alignment = fyne.TextAlignCenter

	// 日志显示区域
	pw.logLabel = widget.NewLabel("")
	pw.logLabel.Wrapping = fyne.TextWrapWord
	pw.logLabel.Alignment = fyne.TextAlignLeading
	logScroll := container.NewScroll(pw.logLabel)
	logScroll.SetMinSize(fyne.NewSize(580, 50))

	// 命令输入框
	pw.commandEntry = widget.NewEntry()
	pw.commandEntry.SetPlaceHolder("Enter command (join, play 5p, peng, gang, hu, skip, help, quit)...")
	pw.commandEntry.OnSubmitted = pw.onCommandSubmitted

	// 快捷按钮
	joinBtn := widget.NewButton("Join", func() {
		pw.executeCommand("join")
	})
	playBtn := widget.NewButton("Play", func() {
		// 简单示例：出 5 筒
		pw.executeCommand("play 5p")
	})
	pengBtn := widget.NewButton("Peng", func() {
		pw.executeCommand("peng")
	})
	gangBtn := widget.NewButton("Gang", func() {
		pw.executeCommand("gang")
	})
	huBtn := widget.NewButton("Hu", func() {
		pw.executeCommand("hu")
	})
	skipBtn := widget.NewButton("Skip", func() {
		pw.executeCommand("skip")
	})
	helpBtn := widget.NewButton("Help", func() {
		pw.executeCommand("help")
	})
	clearBtn := widget.NewButton("Clear", func() {
		pw.clearLogs()
	})

	// 按钮容器
	buttonContainer := container.NewHBox(
		joinBtn, playBtn, pengBtn, gangBtn, huBtn, skipBtn, helpBtn, clearBtn,
	)

	// 主容器
	mainContainer := container.NewVBox(
		pw.statusLabel,
		&widget.Card{
			Title:   "Logs",
			Content: logScroll,
		},
		&widget.Card{
			Title:   "Command",
			Content: pw.commandEntry,
		},
		&widget.Card{
			Title:   "Quick Actions",
			Content: buttonContainer,
		},
	)

	pw.window.SetContent(mainContainer)
}

// Connect 连接到服务器
func (pw *PlayerWindow) Connect() error {
	pw.client = NewTestClient(pw.userID)
	if err := pw.client.Connect(); err != nil {
		pw.addLog("ERROR", fmt.Sprintf("Connect failed: %v", err))
		return err
	}

	pw.updateStatus("Connected", color.NRGBA{0, 200, 0, 255})
	pw.addLog("INFO", "Connected to Connector")

	// 启动后台消息处理
	go pw.messageLoop()

	return nil
}

// messageLoop 后台处理服务器消息
func (pw *PlayerWindow) messageLoop() {
	for {
		select {
		case <-pw.client.done:
			return
		case msg := <-pw.client.messageQueue:
			if msg != nil {
				log.Info("RECV", fmt.Sprintf("%v", msg))
				//pw.addLog("RECV", fmt.Sprintf("%v", msg))
				//for k, v := range msg.Payload {
				//	pw.addLog("", fmt.Sprintf("  %s: %v", k, v))
				//}
			}
		}
	}
}

// onCommandSubmitted 处理命令提交
func (pw *PlayerWindow) onCommandSubmitted(cmd string) {
	pw.executeCommand(cmd)
	pw.commandEntry.SetText("")
}

// executeCommand 执行命令
func (pw *PlayerWindow) executeCommand(cmd string) {
	if cmd == "" {
		return
	}

	pw.addLog("SEND", cmd)

	if cmd == "quit" {
		pw.window.Close()
		return
	}

	if err := pw.client.HandleCommand(cmd); err != nil {
		pw.addLog("ERROR", fmt.Sprintf("Command error: %v", err))
	}
}

// addLog 添加日志
func (pw *PlayerWindow) addLog(level, message string) {
	pw.logMutex.Lock()
	defer pw.logMutex.Unlock()

	timestamp := time.Now().Format("15:04:05")
	var logLine string

	if level == "" {
		logLine = fmt.Sprintf("  %s", message)
	} else {
		logLine = fmt.Sprintf("[%s] %s %s", timestamp, level, message)
	}

	pw.logLines = append(pw.logLines, logLine)

	// 限制日志行数（最多 1000 行）
	if len(pw.logLines) > 1000 {
		pw.logLines = pw.logLines[len(pw.logLines)-1000:]
	}

	// 更新日志显示
	pw.updateLogDisplay()
}

// updateLogDisplay 更新日志显示
func (pw *PlayerWindow) updateLogDisplay() {
	content := ""
	for _, line := range pw.logLines {
		content += line + "\n"
	}
	pw.logLabel.SetText(content)
}

// clearLogs 清空日志
func (pw *PlayerWindow) clearLogs() {
	pw.logMutex.Lock()
	defer pw.logMutex.Unlock()

	pw.logLines = make([]string, 0)
	pw.logLabel.SetText("")
}

// updateStatus 更新状态
func (pw *PlayerWindow) updateStatus(status string, color color.Color) {
	pw.statusLabel.SetText(fmt.Sprintf("Status: %s", status))
	if color != nil {
		pw.statusLabel.Alignment = fyne.TextAlignCenter
	}
}

// Show 显示窗口
func (pw *PlayerWindow) Show() {
	pw.window.Show()
}

// Close 关闭窗口
func (pw *PlayerWindow) Close() {
	if pw.client != nil {
		pw.client.Close()
	}
	pw.window.Close()
}
