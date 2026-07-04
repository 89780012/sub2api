// Package dto provides data transfer objects for HTTP handlers.
package dto

import (
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// RedactCredentials 复制一份 in，剥离 service.SensitiveCredentialKeys 列出的所有敏感子键，
// 并产出一个 has_<key> 状态 map 表示哪些敏感键存在且非零值。
//
// 输入 nil 时返回 nil, nil（避免响应里出现空对象）。
// 不修改入参；调用方拿到的 out 可安全序列化进 JSON 返回前端。
func RedactCredentials(in map[string]any) (out map[string]any, status map[string]bool) {
	if in == nil {
		return nil, nil
	}
	return redactCredentialMap(in, nil, nil)
}

func redactCredentialMap(in map[string]any, path []string, status map[string]bool) (map[string]any, map[string]bool) {
	out := make(map[string]any, len(in))
	for k, v := range in {
		if service.IsSensitiveCredentialKey(k) {
			if isCredentialValuePresent(v) {
				if status == nil {
					status = make(map[string]bool, 4)
				}
				status[credentialStatusKey(path, k)] = true
			}
			continue
		}
		if nested, ok := v.(map[string]any); ok {
			var nestedOut map[string]any
			nestedOut, status = redactCredentialMap(nested, append(path, k), status)
			out[k] = nestedOut
			continue
		}
		if nestedSlice, ok := v.([]any); ok {
			var nestedOut []any
			nestedOut, status = redactCredentialSlice(nestedSlice, append(path, k), status)
			out[k] = nestedOut
			continue
		}
		out[k] = v
	}
	return out, status
}

func redactCredentialSlice(in []any, path []string, status map[string]bool) ([]any, map[string]bool) {
	out := make([]any, len(in))
	for i, item := range in {
		if nested, ok := item.(map[string]any); ok {
			var nestedOut map[string]any
			nestedOut, status = redactCredentialMap(nested, append(path, fmt.Sprintf("%d", i)), status)
			out[i] = nestedOut
			continue
		}
		out[i] = item
	}
	return out, status
}

func credentialStatusKey(path []string, key string) string {
	if len(path) == 0 {
		return "has_" + key
	}
	parts := append(append([]string{}, path...), key)
	return "has_" + strings.Join(parts, "_")
}

// isCredentialValuePresent 判断值是否"存在且非零"。空字符串、nil、false 均视为未配置；
// 其余非零类型（数字、对象、字符串等）视为已配置。
func isCredentialValuePresent(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case string:
		return x != ""
	case bool:
		return x
	default:
		return true
	}
}
