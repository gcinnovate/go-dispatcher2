package helpers

import (
	"fmt"
	"os"

	"github.com/gcinnovate/go-dispatcher2/db"
)

// GetDefaultEnv Returns default value passed if env variable not defined
func GetDefaultEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func authenticateUser() bool {
	return false
}

func getServer(serverName string) int {
	var i int
	err := db.GetDB().QueryRowx(
		"SELECT id FROM servers WHERE id = $1", serverName).Scan(&i)
	if err != nil {
		fmt.Printf("Error geting server: [%v]", err)
		return 0
	}
	return i
}
