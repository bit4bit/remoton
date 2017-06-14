package remoton

import (
	"fmt"
	"time"
)

func GenerateAuthUser() string {
	now := fmt.Sprintf("%d", time.Now().UnixNano())
	return now[len(now)-8 : len(now)]
}
