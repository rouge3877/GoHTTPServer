package config

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	_ "github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	Server struct {
		Port        int
		Proto       string
		IsCgi       bool
		Workdir     string
		IPv4        string
		IPv6        string
		IsDualStack bool
		IsGzip      bool
		DeadLine    time.Duration
	}

	Logger struct {
		LogToFile bool
		FilePath  string
		WithTime  bool
	}
	StartTime time.Time
}

var Cfg Config

func InitConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("yml")
	viper.AddConfigPath("./core/config")

	if error := viper.ReadInConfig(); error != nil {
		fmt.Printf("Error reading config file: %s\n", error)
		os.Exit(1)
	}
	if err := viper.Unmarshal(&Cfg); err != nil {
		fmt.Printf("Unable to decode into struct: %v\n", err)
		os.Exit(1)
	}
}

// ReloadConfig 重新加载配置文件
func ReloadConfig() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yml")
	viper.AddConfigPath("./core/config")

	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("Error reading config file: %s\n", err)
		return err
	}
	if err := viper.Unmarshal(&Cfg); err != nil {
		fmt.Printf("Unable to decode into struct: %v\n", err)
		return err
	}
	return nil
}

// GoVersion 返回Go版本
func GoVersion() string {
	go_v := runtime.Version()
	go_v = strings.TrimPrefix(go_v, "go")
	return go_v
}

var __VERSION__ = "0.01"
var __SERVER_NAME__ = "GoHTTPServer"

func GoHTTPServerVersion() string {
	return __VERSION__
}

func GoHTTPServerName() string {
	return __SERVER_NAME__
}
