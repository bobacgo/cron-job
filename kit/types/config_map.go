package types

import "fmt"

const _configMapDef = "default"

type Config interface {
	// Validate 校验配置项是否有效，返回错误信息
	Validate() error
}

// ConfigMap 是一个通用的配置映射类型，键为字符串，值为实现了 Config 接口的指针类型 T。
type ConfigMap[T Config] map[string]*T

// Get 获取指定 key 的配置项，如果不存在则返回 nil
func (c ConfigMap[T]) Get(key string) *T {
	return c[key]
}

// Default 获取默认配置项，key 固定为 "default"，如果不存在则返回 nil
func (c ConfigMap[T]) Default() *T {
	return c[_configMapDef]
}

// Validate 校验配置项是否有效，返回第一个无效项的错误
func (c ConfigMap[T]) Validate() error {
	for k, v := range c {
		if v == nil {
			return fmt.Errorf("config for key %s is nil", k)
		}
		// 调用配置对象的验证方法
		if err := (*v).Validate(); err != nil {
			return fmt.Errorf("config for key %s is invalid: %w", k, err)
		}
	}
	return nil
}
