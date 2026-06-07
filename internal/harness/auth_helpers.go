package harness

import (
	"fmt"
	"strings"
	"time"
)

func slugID(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else if b.Len() == 0 || strings.HasSuffix(b.String(), "_") {
			continue
		} else {
			b.WriteByte('_')
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return "id_" + fmt.Sprint(time.Now().UnixNano())
	}
	return out
}
