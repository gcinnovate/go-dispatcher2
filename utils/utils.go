package utils

import (
	"bytes"
	"fmt"
	"math"
	"math/rand"
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

// GetServer returns the ID of a server/app
func GetServer(serverName string) int {
	var i int
	err := db.GetDB().QueryRowx(
		"SELECT id FROM servers WHERE name = $1", serverName).Scan(&i)
	if err != nil {
		fmt.Printf("Error:: getting server: [%v] %v", err, serverName)
		return 0
	}
	return i
}

var letters = `abcdefghijklmnopqrstuvwxyz`

func randomWithMax(x int) int {
	z := math.Floor(rand.Float64() * float64(x))
	return int(z)
}

const alphabet = `abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ`
const allowedCharacters = "0123456789" + alphabet
const codeSize = 11

// GetUID return a Unique ID for our resources
func GetUID() string {
	numberOfCodePoinst := len(allowedCharacters)

	var ret bytes.Buffer
	ret.WriteByte(alphabet[randomWithMax(numberOfCodePoinst)])
	for i := 1; i < codeSize; i++ {
		ret.WriteByte(allowedCharacters[randomWithMax(numberOfCodePoinst)])
	}

	return ret.String()
}
