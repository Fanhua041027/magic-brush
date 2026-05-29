package shortcut

import (
	"fmt"
	"sort"
	"strings"
)

// KeyBinding 快捷键绑定
type KeyBinding struct {
	ComboID string `json:"vkCode"`
	KeyName string `json:"keyName"`
}

// GetComboID 根据按键 map 生成唯一的组合 ID
func GetComboID(keys map[uint32]bool) string {
	var codes []int
	for k := range keys {
		codes = append(codes, int(k))
	}
	// 排序是为了ID唯一
	sort.Ints(codes)

	var idBuilder strings.Builder
	for i, code := range codes {
		if i > 0 {
			idBuilder.WriteString("+")
		}
		idBuilder.WriteString(fmt.Sprintf("%d", code))
	}
	return idBuilder.String()
}

// GetReadableName 根据按键 map 生成可读的快捷键名称
func GetReadableName(keys map[uint32]bool) string {
	uniqueNames := make(map[string]bool)
	var parts []string

	for k := range keys {
		name := getKeyName(k)
		if !uniqueNames[name] {
			uniqueNames[name] = true
			parts = append(parts, name)
		}
	}

	// 排序逻辑：Ctrl, Shift, Alt, Win 在前，其他在后
	sort.Slice(parts, func(i, j int) bool {
		order := map[string]int{
			"Ctrl": 0, "Shift": 1, "Alt": 2, "Win": 3,
		}

		w1, ok1 := order[parts[i]]
		w2, ok2 := order[parts[j]]

		if ok1 && ok2 {
			return w1 < w2
		}
		if ok1 {
			return true
		}
		if ok2 {
			return false
		}
		return parts[i] < parts[j]
	})

	return strings.Join(parts, "+")
}

// getKeyName 简单的键名映射辅助函数
func getKeyName(vkCode uint32) string {
	switch vkCode {
	// --- 鼠标侧键 ---
	case 0x05: // VK_XBUTTON1
		return "MouseBack"
	case 0x06: // VK_XBUTTON2
		return "MouseForward"

	// --- 功能键 ---
	case 0x08:
		return "Back"
	case 0x09:
		return "Tab"
	case 0x0D:
		return "Enter"
	case 0x10, 0xA0, 0xA1:
		return "Shift"
	case 0x11, 0xA2, 0xA3:
		return "Ctrl"
	case 0x12, 0xA4, 0xA5:
		return "Alt"
	case 0x13:
		return "Pause"
	case 0x14:
		return "Caps"
	case 0x1B:
		return "Esc"
	case 0x20:
		return "Space"
	case 0x21:
		return "PgUp"
	case 0x22:
		return "PgDn"
	case 0x23:
		return "End"
	case 0x24:
		return "Home"
	case 0x25:
		return "←"
	case 0x26:
		return "↑"
	case 0x27:
		return "→"
	case 0x28:
		return "↓"
	case 0x2C:
		return "PrtSc"
	case 0x2D:
		return "Ins"
	case 0x2E:
		return "Del"
	case 0x5B, 0x5C:
		return "Win"
	case 0x5D:
		return "Menu"

	// --- 小键盘运算符 ---
	case 0x6A:
		return "Num*"
	case 0x6B:
		return "Num+"
	case 0x6C:
		return "NumEnter" // 有些键盘没有
	case 0x6D:
		return "Num-"
	case 0x6E:
		return "Num."
	case 0x6F:
		return "Num/"

	// --- 主键盘标点符号 (OEM Keys) ---
	// 注意：这些键名基于美式标准键盘
	case 0xBA:
		return ";"
	case 0xBB:
		return "="
	case 0xBC:
		return ","
	case 0xBD:
		return "-"
	case 0xBE:
		return "."
	case 0xBF:
		return "/"
	case 0xC0:
		return "`" // 波浪号键
	case 0xDB:
		return "["
	case 0xDC:
		return "\\"
	case 0xDD:
		return "]"
	case 0xDE:
		return "'"

	default:
		// --- 字母 A-Z ---
		if vkCode >= 'A' && vkCode <= 'Z' {
			return string(rune(vkCode))
		}
		// --- 主键盘数字 0-9 ---
		if vkCode >= '0' && vkCode <= '9' {
			return string(rune(vkCode))
		}
		// --- 小键盘数字 0-9 (VK_NUMPAD0 - VK_NUMPAD9) ---
		if vkCode >= 0x60 && vkCode <= 0x69 {
			return fmt.Sprintf("Num%d", vkCode-0x60)
		}
		// --- F1 - F12 ---
		if vkCode >= 0x70 && vkCode <= 0x7B {
			return fmt.Sprintf("F%d", vkCode-0x6F)
		}
		// --- F13 - F24 (极少用到) ---
		if vkCode >= 0x7C && vkCode <= 0x87 {
			return fmt.Sprintf("F%d", vkCode-0x7C+13)
		}

		// 未知键
		return fmt.Sprintf("Key%d", vkCode)
	}
}
