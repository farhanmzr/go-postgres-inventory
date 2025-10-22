// utils/tr_code.go
package utils

import (
	"fmt"
	"time"
)

func GenTransCode(seq int64, t time.Time) string {
	return fmt.Sprintf("TR-%d-%06d", t.Year(), seq)
}
