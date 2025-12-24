package formatter

import "bytes"

// EscapeJSON 转义 JSON 字符串内容（不含引号）
func EscapeJSON(buf *bytes.Buffer, s string) {
	for _, r := range s {
		switch r {
		case '"':
			buf.WriteString(`\"`)
		case '\\':
			buf.WriteString(`\\`)
		case '\n':
			buf.WriteString(`\n`)
		case '\r':
			buf.WriteString(`\r`)
		case '\t':
			buf.WriteString(`\t`)
		default:
			if r < 0x20 {
				buf.WriteString(`\u00`)
				buf.WriteByte("0123456789abcdef"[r>>4])
				buf.WriteByte("0123456789abcdef"[r&0xf])
			} else {
				buf.WriteRune(r)
			}
		}
	}
}
