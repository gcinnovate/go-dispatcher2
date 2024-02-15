package utils

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"strings"
	"time"

	"go-dispatcher2/db"
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
	source := rand.NewSource(time.Now().UnixNano())
	r := rand.New(source) // Creates a new instance of rand.Rand, safe for concurrent use

	numberOfCodePoints := len(allowedCharacters)

	var s strings.Builder
	s.Grow(codeSize) // Pre-allocate memory to improve performance

	// Ensure the first character is an uppercase letter from the alphabet
	s.WriteByte(allowedCharacters[r.Intn(26)] - 32) // Convert to uppercase

	// Generate the rest of the UID
	for i := 1; i < codeSize; i++ {
		s.WriteByte(allowedCharacters[r.Intn(numberOfCodePoints)])
	}

	return s.String()
}

// SliceContains checks if a string is present in a slice
func SliceContains(s []string, str string) bool {

	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}

// GetFieldsAndRelationships returns the slice of fields in existingFields & the relationships as in passedFields
func GetFieldsAndRelationships(existingFields []string, passedFields string) ([]string, map[string][]string) {

	var filtered []string
	relationships := make(map[string][]string)
	fields := strings.Split(passedFields, "[")

	var prevLast string

	for idx, x := range fields {

		y := strings.Split(x, ",")

		if len(y) > 1 {
			relationships[y[len(y)-1]] = []string{}
		} else {
			if SliceContains(existingFields, y[0]) {
				filtered = append(filtered, y[0])
			}
			return filtered, relationships
		}

		rest := y[:len(y)-1] // ignore last element
		if idx == 0 {
			for _, v := range rest {
				if SliceContains(existingFields, v) {
					filtered = append(filtered, v)
				}
			}
		} else {
			restCombined := strings.Join(rest, ",")
			zz := strings.Split(restCombined, "]")
			if len(zz) > 1 {
				for _, m := range zz[1:] {
					for _, f := range strings.Split(m, ",") {
						if len(f) > 1 && SliceContains(existingFields, f) {
							filtered = append(filtered, f)
						}
					}
				}
			}
		}
		// fmt.Printf("<<<<<<<%#v>>>>>>>>>\n", rest)

		prevLast = y[len(y)-1]
		// fmt.Printf("First: %v  Last: %v  F: %v. Prev:%v\n", rest, last, filtered, prevLast)

		// LOOK AHEAD for fields that were enclosed in []
		if len(fields) > idx+1 {
			nextFields := fields[idx+1]

			m := strings.Split(nextFields, "]")

			rest := m[:len(m)-1]

			// fmt.Printf(">>>>>>>>> %v:%v\n", prevLast, rest)
			var appendFields []string
			if len(rest) > 0 {
				for _, p := range strings.Split(rest[0], ",") {
					if len(p) > 0 {
						appendFields = append(appendFields, p)
					}
				}
			}
			relationships[prevLast] = appendFields
		}

	}

	for k, v := range relationships {

		if len(v) <= 0 && SliceContains(existingFields, k) {
			filtered = append(filtered, k)
			delete(relationships, k)
		}
		if v == nil {
			delete(relationships, k)
		}
	}
	// fmt.Printf("%#v==> %#v, %v\n", relationships, filtered, len(relationships["z"]))
	return filtered, relationships
}
