package cmd

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// spinner 终端旋转动画组件，在等待 LLM/搜索时提供可视化反馈
type spinner struct {
	mu      sync.Mutex
	done    chan struct{}
	message string
	active  bool
}

// 旋转动画的每一帧字符
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// startSpinner 启动旋转动画，message 为初始提示文字，返回 spinner 实例
func startSpinner(message string) *spinner {
	s := &spinner{
		done:    make(chan struct{}),
		message: message,
		active:  true,
	}

	go func() {
		i := 0
		for {
			select {
			case <-s.done:
				// 清空 spinner 行，确保不留残影
				clearLen := len(s.message) + 4
				if clearLen < 40 {
					clearLen = 40
				}
				fmt.Fprint(os.Stderr, "\r"+strings.Repeat(" ", clearLen)+"\r")
				return
			default:
				s.mu.Lock()
				msg := s.message
				s.mu.Unlock()
				fmt.Fprintf(os.Stderr, "\r%s %s", spinnerFrames[i%len(spinnerFrames)], msg)
				i++
				time.Sleep(80 * time.Millisecond)
			}
		}
	}()

	return s
}

// update 更新 spinner 的提示文字（线程安全）
func (s *spinner) update(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.message = message
}

// stop 停止旋转动画，可选地在停止前打印一行完成消息
// doneMsg 为空时不打印完成消息
func (s *spinner) stop(doneMsg string) {
	if s == nil || !s.active {
		return
	}
	close(s.done)
	// 给 goroutine 一点时间清理
	time.Sleep(120 * time.Millisecond)
	s.active = false
	if doneMsg != "" {
		fmt.Fprintln(os.Stderr, doneMsg)
	}
}
