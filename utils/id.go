package utils

import (
	"fmt"
	"math/rand"
	"time"
)

func GenerateRequestID() string {
	return fmt.Sprintf("%d%x", time.Now().UnixNano(), rand.Intn(10000))
}
