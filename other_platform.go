//go:build linux

package main

import (
	"io"
	"os"
	"os/exec"
)

func convertM3U8ToMP3(ffmpegPath string, m3u8URL string) (io.Reader, *exec.Cmd, error) {
	cmd := exec.Command(ffmpegPath, "-i", m3u8URL, "-f", "mp3", "-")

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
