package vk

import (
	"fmt"
	"github.com/spf13/cast"
	viperlib "github.com/spf13/viper" // 自定义包名，避免与内置 viper 实例冲突
)

// viper 库实例
var viper *viperlib.Viper

// ConfigFunc 动态加载配置信息
type ConfigFunc func() map[string]any

// ConfigFuncs 先加载到此数组，loadConfig 在动态生成配置信息
var ConfigFuncs map[string]ConfigFunc

func init() {
	viper = viperlib.New()
	viper.SetConfigFile("config.yml")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		fmt.Println("load local config.toml error")
	}
	// 从docker-compose读取配置
	viper.AutomaticEnv()
	ConfigFuncs = make(map[string]ConfigFunc)
}

// Env 读取环境变量，支持默认值
func Env(envName string, defaultValue ...any) any {
	if len(defaultValue) > 0 {
		return internalGet(envName, defaultValue[0])
	}
	return internalGet(envName)
}

// Add 新增配置项
func Add(name string, configFn ConfigFunc) {
	ConfigFuncs[name] = configFn
}

// Get 获取配置项
// 第一个参数 path 允许使用点式获取，如：app.name
// 第二个参数允许传参默认值
func Get(path string, defaultValue ...any) string {
	return GetString(path, defaultValue...)
}

func internalGet(path string, defaultValue ...any) any {
	// config 或者环境变量不存在的情况
	if !viper.IsSet(path) || Empty(viper.Get(path)) {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return nil
	}
	return viper.Get(path)
}

// GetString 获取 String 类型的配置信息
func GetString(path string, defaultValue ...any) string {
	return cast.ToString(internalGet(path, defaultValue...))
}

// GetInt 获取 Int 类型的配置信息
func GetInt(path string, defaultValue ...any) int {
	return cast.ToInt(internalGet(path, defaultValue...))
}

// GetFloat64 获取 float64 类型的配置信息
func GetFloat64(path string, defaultValue ...any) float64 {
	return cast.ToFloat64(internalGet(path, defaultValue...))
}

// GetInt64 获取 Int64 类型的配置信息
func GetInt64(path string, defaultValue ...any) int64 {
	return cast.ToInt64(internalGet(path, defaultValue...))
}

// GetUint 获取 Uint 类型的配置信息
func GetUint(path string, defaultValue ...any) uint {
	return cast.ToUint(internalGet(path, defaultValue...))
}

// GetBool 获取 Bool 类型的配置信息
func GetBool(path string, defaultValue ...any) bool {
	return cast.ToBool(internalGet(path, defaultValue...))
}

func GetStringMap(path string) map[string]interface{} {
	return viper.GetStringMap(path)
}

// GetStringMapString 获取结构数据
func GetStringMapString(path string) map[string]string {
	return viper.GetStringMapString(path)
}

func GetStringMapStringSlice(key string) map[string][]string {
	return viper.GetStringMapStringSlice(key)
}

func GetStringSlice(key string) []string {
	return viper.GetStringSlice(key)
}

// Get(key string) interface{}
// Sub(key string) *Viper
// GetBool(key string) bool
// GetDuration(key string) time.Duration
// GetFloat64(key string) float64
// GetInt(key string) int
// GetInt32(key string) int32
// GetInt64(key string) int64
// GetIntSlice(key string) []int
// GetSizeInBytes(key string) uint
// GetString(key string) string
// GetStringMap(key string) map[string]interface{}
// GetStringMapString(key string) map[string]string
// GetStringMapStringSlice(key string) map[string][]string
// GetStringSlice(key string) []string
// GetTime(key string) time.Time
// GetUint(key string) uint
// GetUint32(key string) uint32
// GetUint64(key string) uint64
// InConfig(key string) bool
// IsSet(key string) bool
// AllSettings() map[string]interface{}
