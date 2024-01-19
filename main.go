//go:generate goversioninfo
package main

import (
	_ "embed"
	"encoding/json"
	"github.com/fufuok/favicon"
	"github.com/gin-gonic/gin"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"time"
)

type Config struct {
	Streams    map[string]string `json:"Streams"`
	IpPort     string            `json:"ipPort"`
	FfmpegPath string            `json:"ffmpegPath"`
}

//go:embed favicon.ico
var favData []byte

var ffmpegApp string

func main() {
	// 非调试模式
	gin.SetMode(gin.ReleaseMode)

	//判断系统
	if runtime.GOOS == "windows" {
		ffmpegApp = "./ffmpeg.exe"
	} else {
		ffmpegApp = "ffmpeg"
	}

	config, _ := loadConfig("./config.json")

	r := gin.Default()
	//favicon.ico

	//部分场景失效 比如mp3Stream
	//r.StaticFile("/favicon.ico", "./favicon.ico")
	//打包后仍需本地有ico文件 不然会报错 thinkerou
	//r.Use(favicon.New("./favicon.ico"))
	//使用嵌入 把图标嵌入到打包的可执行文件 fufuok
	r.Use(favicon.New(favicon.Config{
		FileData: favData,
	}))

	r.GET("/", func(context *gin.Context) {
		context.String(http.StatusOK, "Server Run Successful!\nVersion:2023/8/20")
	})

	r.GET("/:streamID", func(c *gin.Context) {
		config, err := loadConfig("./config.json")
		if err != nil {
			c.String(http.StatusInternalServerError, "Error loading config")
			return
		}

		streamID := c.Param("streamID")
		m3u8URL, found := config.Streams[streamID]
		if !found {
			c.String(http.StatusNotFound, "Stream not found")
			return
		}

		mp3Stream, ffmpegCmd, err := convertM3U8ToMP3(config.FfmpegPath, m3u8URL)
		if err != nil {
			c.String(http.StatusInternalServerError, "Error converting M3U8 to MP3")
			return
		}
		defer func() {
			// 等待一段时间，以确保进程有足够的时间完成任务并退出
			time.Sleep(5 * time.Second) // 等待时间
			ffmpegCmd.Process.Kill()
		}()

		c.Header("Content-Type", "audio/mpeg")
		c.Status(http.StatusOK)
		io.Copy(c.Writer, mp3Stream)
	})

	log.Printf("[main] %s", "Main函数运行中")
	ipAddr := "localhost:" + config.IpPort
	log.Printf("[main] IpAddress:http://%s/\n", ipAddr)

	r.Run(":" + config.IpPort)
}

func loadConfig(filename string) (*Config, error) {
	log.Printf("[loadConfig] Loading config from:'%s'", filename)
	file, err := os.Open(filename)
	//file, err := os.OpenFile(filename, os.O_CREATE, 0)
	if err != nil {
		log.Printf("[loadConfig] Config file not found! T_T")

		os.Create(filename)

		content := `{
  "ipPort": "24748",
  "ffmpegPath": "` + ffmpegApp + `",
  "Streams": {
    "广东羊城交通台": "http://ls.qingting.fm/live/1262/64k.m3u8?format=aac",
    "广东广播电视台股市广播": "http://ls.qingting.fm/live/4847/64k.m3u8?format=aac",
    "广东珠江经济电台": "http://ls.qingting.fm/live/1259/64k.m3u8?format=aac",
    "广东广播电视台文体广播": "http://ls.qingting.fm/live/471/64k.m3u8?format=aac",
    "广东音乐之声": "http://ls.qingting.fm/live/1260/64k.m3u8?format=aac",
    "佛山电台FM906": "http://ls.qingting.fm/live/1264/64k.m3u8?format=aac",
    "番禺电台畅快1017": "http://ls.qingting.fm/live/20212427/64k.m3u8?format=aac",
    "广州MYFM 88.0": "http://ls.qingting.fm/live/20194/64k.m3u8?format=aac"
  }
}`
		os.WriteFile(filename, []byte(content), 0666)

		log.Printf("[loadConfig] %s", "Retrying load config")
		config, err := loadConfig(filename)

		return config, err
	}
	defer file.Close()

	config := &Config{}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(config)

	if err != nil {
		return nil, err
	}
	return config, nil
}

func convertM3U8ToMP3(ffmpegPath string, m3u8URL string) (io.Reader, *exec.Cmd, error) {
	cmd := exec.Command(ffmpegPath, "-i", m3u8URL, "-f", "mp3", "-")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
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
