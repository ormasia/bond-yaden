package log

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type logger struct {
	fileName   string //日志文件名
	logLevel   int
	maxSize    int64
	maxFileNum int
	count      int
	file       *os.File
	fileChan   chan *string
	flushTimer *time.Ticker
}

var std = logger{
	fileName:   "",
	logLevel:   LOGLEVEL_WARN,
	maxSize:    128 * 1024 * 1024,
	maxFileNum: 10,
	fileChan:   make(chan *string, 50000),
}

const (
	LOGLEVEL_OFF = iota
	LOGLEVEL_FATAL
	LOGLEVEL_ERROR
	LOGLEVEL_WARN
	LOGLEVEL_INFO
	LOGLEVEL_DEBUG
	LOGLEVEL_VERBOSE
)

func init() {

	std.flushTimer = time.NewTicker(1 * time.Second)
	go run()
}

// 设置日志输出路径及级别
// 如不做任何设置，默认输出到标准输出，日志级别默认为LOGLEVEL_INFO
func SetLog(fileName string, level int) {
	SetLogFileName(fileName)
	SetLogLevel(level)
}

func SetLogFileName(fileName string) error {
	std.fileName = fileName
	// 判断目录是否存在
	index := strings.LastIndex(fileName, "/")
	if index > 0 {
		dir := string([]byte(fileName)[0:index])
		CreateMultiDir(dir)
	}
	var err error
	std.file, err = os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("newFile OpenFile fail, filename=[%s] err=[%v]", fileName, err)
		return err
	}
	return nil
}

func SetLogMaxSize(logSize int) {
	if logSize < 1024*1024 {
		logSize = 1024 * 1024
	}
	std.maxSize = int64(logSize)
}
func SetLogMaxFileNum(maxFileNum int) {
	if maxFileNum > 100 {
		maxFileNum = 100
	}
	if maxFileNum < 1 {
		maxFileNum = 1
	}
	std.maxFileNum = maxFileNum
}
func SetLogLevel(level interface{}) {
	ilevel := LOGLEVEL_INFO

	switch level := level.(type) {
	case int:
		ilevel = level
	case string:
		level = strings.ToUpper(level)
		switch level {
		case "OFF":
			ilevel = LOGLEVEL_OFF
		case "FATAL":
			ilevel = LOGLEVEL_FATAL
		case "ERROR":
			ilevel = LOGLEVEL_ERROR
		case "WARN":
			ilevel = LOGLEVEL_WARN
		case "INFO":
			ilevel = LOGLEVEL_INFO
		case "DEBUG":
			ilevel = LOGLEVEL_DEBUG
		case "VERBOSE":
			ilevel = LOGLEVEL_VERBOSE
		default:
			Warn("Set to unsupport log level: [%s], use default level INFO", level)
		}
	default:
		Warn("Set to unsupport log level, only accept integer or string values")
	}

	std.logLevel = ilevel
}

func Init(format string, v ...interface{}) {
	if std.logLevel == LOGLEVEL_OFF {
		return
	}

	str := Format(2, "INIT", format, v...)
	std.fileChan <- &str
}

func Fatal(format string, v ...interface{}) {
	if std.logLevel < LOGLEVEL_FATAL {
		return
	}
	str := Format(2, "FATAL", format, v...)
	std.fileChan <- &str
}
func Error(format string, v ...interface{}) {
	if std.logLevel < LOGLEVEL_ERROR {
		return
	}
	str := Format(2, "ERROR", format, v...)
	std.fileChan <- &str
}
func Warn(format string, v ...interface{}) {
	if std.logLevel < LOGLEVEL_WARN {
		return
	}
	str := Format(2, "WARN", format, v...)
	std.fileChan <- &str
}
func Info(format string, v ...interface{}) {
	if std.logLevel < LOGLEVEL_INFO {
		return
	}
	str := Format(2, "INFO", format, v...)
	std.fileChan <- &str
}
func Debug(format string, v ...interface{}) {
	if std.logLevel < LOGLEVEL_DEBUG {
		return
	}

	str := Format(2, "DEBUG", format, v...)
	std.fileChan <- &str
}
func Verbose(format string, v ...interface{}) {
	if std.logLevel < LOGLEVEL_VERBOSE {
		return
	}

	str := Format(2, "VERBOSE", format, v...)
	std.fileChan <- &str
}

func Buf(data []byte, msg string) {
	if std.logLevel < LOGLEVEL_DEBUG {
		return
	}
	mtitle := fmt.Sprintf("%s, len=%d\n", msg, len(data))
	strTmp := mtitle + colTitle + hex.Dump(data)

	str := Format(2, "BUF", "%s", strTmp)
	std.fileChan <- &str
}

func Data(format string, v ...interface{}) {
	if std.logLevel == LOGLEVEL_OFF {
		return
	}
	str := Format(2, "DATA", format, v...)
	std.fileChan <- &str
}

func Panic(format string, v ...interface{}) {
	str := fmt.Sprintf(format, v...)
	panic(str)
}

func CanVerbose() bool {
	return std.logLevel <= LOGLEVEL_VERBOSE
}
func CanDebug() bool {
	return std.logLevel <= LOGLEVEL_DEBUG
}

const colTitle = "__________00_01_02_03_04_05_06_07__08_09_0A_0B_0C_0D_0E_0F\n"

func output(file *os.File, content *string) {
	if file != nil {
		file.WriteString(*content)
		std.count++
		if std.count > 100 {
			std.count = 0
			shiftFiles()
		}
	} else if std.fileName == "" {
		fmt.Fprintf(os.Stdout, "%s", *content)
	} else {
		fmt.Printf("output error!!!")
	}
}

func shiftFiles() error {
	fileInfo, err := os.Stat(std.fileName)
	if err != nil {
		return err
	}

	if fileInfo.Size() < std.maxSize {
		return nil
	}

	std.file.Close()
	//shift file
	for i := std.maxFileNum - 2; i >= 0; i-- {
		var nameOld string
		if i == 0 {
			nameOld = std.fileName
		} else {
			nameOld = fmt.Sprintf("%s.%d", std.fileName, i)
		}
		fileInfo, err := os.Stat(nameOld)
		if err != nil {
			continue
		}
		if fileInfo.IsDir() {
			continue
		}
		nameNew := fmt.Sprintf("%s.%d", std.fileName, i+1)
		os.Rename(nameOld, nameNew)
	}

	std.file, err = os.OpenFile(std.fileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("newFile OpenFile fail, filename=[%s] err=[%v]", std.fileName, err)
		return err
	}
	return nil
}

func Format(callerSkip int, logLevel string, format string, params ...interface{}) string {
	nowTime := time.Now()
	_, fileName, fileLine, _ := runtime.Caller(callerSkip)
	strTag := fmt.Sprintf("{\"level\":\"%s\",\"time\":\"%s\", \"file\":\"%s:%d\",\"msg\": \"%s\"}",
		logLevel, nowTime.Format("2006-01-02 15:04:05.000"), filepath.Base(fileName), fileLine, fmt.Sprintf(format, params...))
	strLine := strTag + "\n"
	return strLine
}

func Format_(callerSkip int, logLevel string, format string, params ...interface{}) string {
	nowTime := time.Now()
	_, fileName, fileLine, _ := runtime.Caller(callerSkip)
	strTag := fmt.Sprintf("%s [%s] %s:%d | ",
		nowTime.Format("2006-01-02 15:04:05.000"), logLevel, filepath.Base(fileName), fileLine)

	/// 格式化
	strContent := fmt.Sprintf(format, params...)

	strLine := strTag + strContent + "\n"
	return strLine
}

func run() {
	for {
		select {
		case content := <-std.fileChan:
			output(std.file, content)
		case <-std.flushTimer.C:
			flush()
		}
	}
}

func flush() {

	if std.file != nil {
		f := bufio.NewWriter(std.file)
		f.Flush()
	}

}

func isExist(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		return os.IsExist(err)
	}
	return true
}

// 支持多级目录创建
func CreateMultiDir(filePath string) error {
	if !isExist(filePath) {
		err := os.MkdirAll(filePath, os.ModePerm)
		if err != nil {
			fmt.Printf("Create dir failed ,error:%s, dir:%s.", err.Error(), filePath)
			return err
		}
		return err
	}
	return nil
}
