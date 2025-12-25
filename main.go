//go:generate goversioninfo
package main

import (
	_ "embed"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/fufuok/favicon"
	"github.com/getlantern/systray"
	"github.com/gin-gonic/gin"

	"context"
	"sync"
)

// AppVersion 版本信息
const AppVersion = "2025.12.25.1"

// BroadcastStream 广播流管理器 - 用于将单个 ffmpeg 输出分发给多个客户端
type BroadcastStream struct {
	readers  map[chan []byte]bool
	mutex    sync.RWMutex
	cmd      *exec.Cmd
	ctx      context.Context
	cancel   context.CancelFunc
	refCount int
	mu       sync.Mutex
}

func NewBroadcastStream(m3u8URL, ffmpegPath string) (*BroadcastStream, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// 启动 ffmpeg 进程
	cmd := exec.Command(ffmpegPath, "-i", m3u8URL, "-f", "mp3", "-")

	// windows下 隐藏调用ffmpeg产生的黑窗口
	winHiddenCMDFrom(cmd)

	cmd.Stderr = os.Stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, err
	}

	bs := &BroadcastStream{
		readers: make(map[chan []byte]bool),
		ctx:     ctx,
		cancel:  cancel,
		cmd:     cmd,
	}

	// 启动广播协程
	go bs.broadcast(stdout)

	return bs, nil
}

func (bs *BroadcastStream) broadcast(reader io.Reader) {
	buffer := make([]byte, 32768) // 32KB buffer

	for {
		select {
		case <-bs.ctx.Done():
			return
		default:
			n, err := reader.Read(buffer)
			if err != nil {
				if err != io.EOF {
					log.Printf("Broadcast error: %v", err)
				}
				break
			}

			data := make([]byte, n)
			copy(data, buffer[:n])

			bs.mutex.RLock()
			for ch := range bs.readers {
				select {
				case ch <- data:
				case <-time.After(100 * time.Millisecond): // 防止阻塞
					// 客户端读取太慢，跳过
				}
			}
			bs.mutex.RUnlock()
		}
	}
}

func (bs *BroadcastStream) AddReader() chan []byte {
	bs.mutex.Lock()
	defer bs.mutex.Unlock()

	ch := make(chan []byte, 100) // 带缓冲的通道
	bs.readers[ch] = true

	return ch
}

func (bs *BroadcastStream) RemoveReader(ch chan []byte) {
	bs.mutex.Lock()
	defer bs.mutex.Unlock()

	if _, exists := bs.readers[ch]; exists {
		delete(bs.readers, ch)
		close(ch)
	}
}

func (bs *BroadcastStream) Close() {
	bs.cancel()
	if bs.cmd != nil && bs.cmd.Process != nil {
		err := bs.cmd.Process.Kill()
		if err != nil {
			return
		}
	}
}

// StreamManager 修改全局流管理器
type StreamManager struct {
	mutex   sync.RWMutex
	streams map[string]*BroadcastStream
}

var streamManager = &StreamManager{
	streams: make(map[string]*BroadcastStream),
}

func getOrCreateBroadcastStream(m3u8URL string, ffmpegPath string) (*BroadcastStream, func(), error) {
	streamManager.mutex.Lock()

	if stream, exists := streamManager.streams[m3u8URL]; exists {
		// 流已存在，增加引用计数
		stream.mu.Lock()
		stream.refCount++
		stream.mu.Unlock()

		streamManager.mutex.Unlock()

		cleanup := func() {
			stream.mu.Lock()
			stream.refCount--
			currentRef := stream.refCount
			stream.mu.Unlock()

			if currentRef <= 0 {
				// 没有更多引用时，清理流
				go func() {
					time.Sleep(5 * time.Second) // 延迟清理
					streamManager.mutex.Lock()
					stream.mu.Lock()
					if stream.refCount <= 0 {
						stream.Close()
						delete(streamManager.streams, m3u8URL)
					}
					stream.mu.Unlock()
					streamManager.mutex.Unlock()
				}()
			}
		}

		return stream, cleanup, nil
	}

	// 创建新的广播流
	newStream, err := NewBroadcastStream(m3u8URL, ffmpegPath)
	if err != nil {
		streamManager.mutex.Unlock()
		return nil, nil, err
	}

	newStream.refCount = 1
	streamManager.streams[m3u8URL] = newStream
	streamManager.mutex.Unlock()

	cleanup := func() {
		streamManager.mutex.Lock()
		if stream, exists := streamManager.streams[m3u8URL]; exists {
			stream.mu.Lock()
			stream.refCount--
			currentRef := stream.refCount
			stream.mu.Unlock()

			if currentRef <= 0 {
				go func() {
					time.Sleep(5 * time.Second)
					streamManager.mutex.Lock()
					if stream, exists := streamManager.streams[m3u8URL]; exists {
						stream.mu.Lock()
						if stream.refCount <= 0 {
							stream.Close()
							delete(streamManager.streams, m3u8URL)
						}
						stream.mu.Unlock()
					}
					streamManager.mutex.Unlock()
				}()
			}
		}
		streamManager.mutex.Unlock()
	}

	return newStream, cleanup, nil
}

type Config struct {
	Streams    map[string]string `json:"Streams"`
	IpPort     string            `json:"ipPort"`
	FfmpegPath string            `json:"ffmpegPath"`
}

//go:embed favicon.ico
var favData []byte

var ffmpegApp string

var osType = runtime.GOOS

func main() {
	// 非调试模式
	gin.SetMode(gin.ReleaseMode)

	systray.Run(onReady, onExit)

}

func appCore() {
	//判断系统
	if osType == "windows" {
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
		context.String(http.StatusOK, "Server Run Successful!\nVersion: "+AppVersion)
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

		// 使用共享的广播流
		broadcastStream, cleanup, err := getOrCreateBroadcastStream(m3u8URL, config.FfmpegPath)
		if err != nil {
			c.String(http.StatusInternalServerError, "Error creating stream")
			return
		}
		defer cleanup()

		// 为当前客户端创建一个读取通道
		dataChan := broadcastStream.AddReader()
		defer broadcastStream.RemoveReader(dataChan) // 类型现在匹配

		c.Header("Content-Type", "audio/mpeg")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "close")
		c.Status(http.StatusOK)

		// 检测客户端断开连接
		clientGone := c.Done()

		// 流式传输数据
		for {
			select {
			case data, ok := <-dataChan:
				if !ok {
					// 通道已关闭
					return
				}
				_, err := c.Writer.Write(data)
				if err != nil {
					log.Printf("Write error: %v", err)
					return
				}
				c.Writer.Flush() // 立即发送数据
			case <-clientGone:
				// 客户端断开连接
				return
			case <-time.After(30 * time.Second): // 30秒超时
				return
			}
		}
	})

	log.Printf("[main] %s", "Main函数运行中")
	ipAddr := "localhost:" + config.IpPort
	log.Printf("[main] IpAddress:http://%s/\n", ipAddr)

	err := r.Run(":" + config.IpPort)
	if err != nil {
		return
	}
}

func loadConfig(filename string) (*Config, error) {
	log.Printf("[loadConfig] Loading config from:'%s'", filename)
	file, err := os.Open(filename)
	//file, err := os.OpenFile(filename, os.O_CREATE, 0)
	if err != nil {
		log.Printf("[loadConfig] Config file not found! T_T")

		content := `{
  "ipPort": "24748",
  "ffmpegPath": "` + ffmpegApp + `",
  "Streams": {
    "广东羊城交通台": "http://ls.qingting.fm/live/1262/64k.m3u8?format=aac",
    "广东广播电视台股市广播": "http://ls.qingting.fm/live/4847/64k.m3u8?format=aac",
    "广东珠江经济电台": "http://ls.qingting.fm/live/1259/64k.m3u8?format=aac",
    "广东广播电视台文体广播": "http://ls.qingting.fm/live/471/64k.m3u8?format=aac",
    "广东音乐之声": "http://ls.qingting.fm/live/1260/64k.m3u8?format=aac",
    "番禺电台畅快1017": "http://ls.qingting.fm/live/20212427/64k.m3u8?format=aac",
    "广州MYFM 88.0": "http://ls.qingting.fm/live/20194/64k.m3u8?format=aac"
  }
}`
		err = os.WriteFile(filename, []byte(content), 0666)
		if err != nil {
			return nil, err
		}

		log.Printf("[loadConfig] %s", "Retrying load config")
		config, err := loadConfig(filename)

		return config, err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {

		}
	}(file)

	config := &Config{}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(config)

	if err != nil {
		return nil, err
	}
	return config, nil
}

func onReady() {

	systray.SetIcon(favData)
	systray.SetTitle("convertM3U8ToMP3")
	systray.SetTooltip("convertM3U8ToMP3")

	mQuit := systray.AddMenuItem("退出", "")

	log.Printf("[systray] app starting! ^_^")
	// Sets the icon of a menu item. Only available on Mac and Windows.
	//mQuit.SetIcon(icon.Data)

	//监听点击"退出"按钮
	go func() {
		<-mQuit.ClickedCh
		systray.Quit()
	}()

	appCore()
}

func onExit() {
	// clean up here
	log.Printf("[systray] app quited! ^_^")
	os.Exit(130)
}
