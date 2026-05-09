package main

import "shell-mate/cmd"

// main 是 shell-mate 的入口函数，直接委托 cmd 包执行
func main() {
	cmd.Execute()
}
