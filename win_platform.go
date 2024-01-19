//go:build windows

package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"syscall"
)

func convertM3U8ToMP3(ffmpegPath string, m3u8URL string) (io.Reader, *exec.Cmd, error) {
	cmd := exec.Command(ffmpegPath, "-i", m3u8URL, "-f", "mp3", "-")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true} // windows隐藏黑窗口
	cmd.Stderr = os.Stderr
	mp3Stream, err := cmd.StdoutPipe()

	if err != nil {
		return nil, nil, err
	}

	err = cmd.Start()
	if err != nil {
		return nil, nil, err
	}

	return mp3Stream, cmd, nil
}

// 关闭对应的程序，这里的command只适用于windows
func onExit() {
	// clean up here
	log.Printf("[systray] app quited! ^_^")
	command := exec.Command("taskkill", "/im", "server.exe", "/F")
	output, err := command.CombinedOutput()
	if err != nil {
		fmt.Println("output err:", err)
		return
	}
	fmt.Println("output:", string(output))
	os.Exit(130)

}
