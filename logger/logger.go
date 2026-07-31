package logger

import (
	"bluebell/setting"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var lg *zap.Logger

// Init 初始化日志器
func Init(cfg *setting.LogConfig, mode string) (err error) {
	// 1. 创建日志写入器（带轮转切割功能）
	writeSyncer := getLogWriter(cfg.FileName, cfg.MaxSize, cfg.MaxBackups, cfg.MaxAge)
	// 2. 创建日志编码器（JSON格式）
	encoder := getEncoder()
	// 3. 解析日志级别（从字符串 "info"/"debug"/"warn"/"error" 转成 zapcore.Level）
	var l = new(zapcore.Level)
	err = l.UnmarshalText([]byte(cfg.Level))
	if err != nil {
		return
	}
	// 4. 根据模式决定日志输出到哪里
	var core zapcore.Core
	if mode == "dev" {
		// 开发模式：同时输出到文件和终端
		consoleEncoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
		core = zapcore.NewTee(
			zapcore.NewCore(encoder, writeSyncer, l),                                     // 文件输出
			zapcore.NewCore(consoleEncoder, zapcore.Lock(os.Stdout), zapcore.DebugLevel), // 终端输出
		)
	} else {
		// 发布模式：只输出到文件
		core = zapcore.NewCore(encoder, writeSyncer, l)
	}
	// 5. 创建 Logger，AddCaller() 会记录调用日志的代码位置
	lg = zap.New(core, zap.AddCaller())
	// 6. 替换 zap 的全局 Logger，之后直接用 zap.L() 就能拿到这个 logger
	zap.ReplaceGlobals(lg)
	zap.L().Info("init logger success")
	return
}

// getEncoder 获取日志编码器
func getEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder         // ISO8601 时间格式
	encoderConfig.TimeKey = "time"                                // 时间字段的 key 名
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder       // 日志级别大写
	encoderConfig.EncodeDuration = zapcore.SecondsDurationEncoder // 耗时用秒表示
	encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder       // 调用者信息用包名/文件:行号 的短格式
	return zapcore.NewJSONEncoder(encoderConfig)
}

// getLogWriter 创建日志写入器（带自动切割）
func getLogWriter(filename string, maxSize, maxBackup, maxAge int) zapcore.WriteSyncer {
	lumberJackLogger := &lumberjack.Logger{
		Filename:   filename,  // 日志文件路径
		MaxSize:    maxSize,   // 单个文件最大大小（MB），超过则切割
		MaxBackups: maxBackup, // 最多保留的旧日志文件数
		MaxAge:     maxAge,    // 旧日志最长保留天数
	}
	return zapcore.AddSync(lumberJackLogger)
}

// GinLogger 接收 Gin 框架默认的日志（中间件）
func GinLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		c.Next() // 先执行后续的处理函数
		// 计算耗时
		cost := time.Since(start)
		lg.Info(path,
			zap.Int("status", c.Writer.Status()),                                 // 响应状态码
			zap.String("method", c.Request.Method),                               // 请求方法
			zap.String("path", path),                                             // 请求路径
			zap.String("query", query),                                           // 查询参数
			zap.String("ip", c.ClientIP()),                                       // 客户端IP
			zap.String("user-agent", c.Request.UserAgent()),                      // 用户代理
			zap.String("errors", c.Errors.ByType(gin.ErrorTypePrivate).String()), // 错误信息
			zap.Duration("cost", cost),                                           // 请求耗时
		)
	}
}

// GinRecovery 恢复 Panic 的中间件
func GinRecovery(stack bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 检查是否是断开的连接（broken pipe / connection reset）
				var brokenPipe bool
				if ne, ok := err.(*net.OpError); ok {
					if se, ok := ne.Err.(*os.SyscallError); ok {
						if strings.Contains(strings.ToLower(se.Error()), "broken pipe") ||
							strings.Contains(strings.ToLower(se.Error()), "connection reset by peer") {
							brokenPipe = true
						}
					}
				}
				httpRequest, _ := httputil.DumpRequest(c.Request, false)
				if brokenPipe {
					// 客户端断开连接，只记录日志，不需要返回响应
					lg.Error(c.Request.URL.Path,
						zap.Any("error", err),
						zap.String("request", string(httpRequest)),
					)
					c.Error(err.(error))
					c.Abort()
					return
				}
				if stack {
					// 如果需要堆栈信息，也记录下来
					lg.Error("[Recovery from panic]",
						zap.Any("error", err),
						zap.String("request", string(httpRequest)),
						zap.String("stack", string(debug.Stack())),
					)
				} else {
					lg.Error("[Recovery from panic]",
						zap.Any("error", err),
						zap.String("request", string(httpRequest)),
					)
				}
				// 返回 500 状态码
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}
}
