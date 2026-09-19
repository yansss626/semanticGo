package config

import (
	"log"

	"github.com/spf13/viper"
)

type Config struct {
	Server struct {
		IP   string
		Port int
	} `mapstructure:"server"`
	Log struct {
		Level   string
		LogPath string `mapstructure:"logPath"`
	} `mapstructure:"log"`
	Embedding struct {
		ApiKey           string  `mapstructure:"api_key"`
		BaseUrl          string  `mapstructure:"base_url"`
		Model            string  `mapstructure:"model"`
		VectorDimensions int     `mapstructure:"vector_dimensions"`
		RecallScore      float64 `mapstructure:"recall_score"`
		TopK             int     `mapstructure:"topK"`
	} `mapstructure:"embedding"`
	Rerank struct {
		ApiKey      string  `mapstructure:"api_key"`
		BaseUrl     string  `mapstructure:"base_url"`
		Model       string  `mapstructure:"model"`
		Instruct    string  `mapstructure:"instruct"`
		RerankScore float64 `mapstructure:"rerank_score"`
	} `mapstructure:"rerank"`
	Cache struct {
		IP   string `mapstructure:"ip"`
		Port int    `mapstructure:"port"`
	} `mapstructure:"cache"`
}

var conf *Config

func InitConfig(filePath string, typ ...string) {
	v := viper.New()
	v.SetConfigFile(filePath)
	if len(typ) > 0 {
		v.SetConfigType(typ[0])
	}
	err := v.ReadInConfig()
	if err != nil {
		log.Fatal(err)
	}
	conf = &Config{}
	err = v.Unmarshal(conf)
	if err != nil {
		log.Fatal(err)
	}

}

func GetConfig() *Config {
	return conf
}
