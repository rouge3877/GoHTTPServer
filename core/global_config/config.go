package globalconfig

import (
	"fmt"
	"os"

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
	}

	Logger struct {
		LogToFile bool
		FilePath  string
		WithTime  bool
	}
}

var GlobalConfig Config

func InitConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("yml")
	viper.AddConfigPath("./core/global_config")

	if error := viper.ReadInConfig(); error != nil {
		fmt.Printf("Error reading config file: %s\n", error)
		os.Exit(1)
	}
	if err := viper.Unmarshal(&GlobalConfig); err != nil {
		fmt.Printf("Unable to decode into struct: %v\n", err)
		os.Exit(1)
	}
}
