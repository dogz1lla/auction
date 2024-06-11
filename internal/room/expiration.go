package room

import (
	"time"
)

func GetMillisTill(end time.Time) int64 {
	return end.Sub(time.Now().UTC()).Milliseconds()
}
