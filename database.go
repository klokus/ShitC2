package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/alexedwards/argon2id"

	_ "github.com/mattn/go-sqlite3"
)

var Database *sql.DB

var HashParams = &argon2id.Params{
	Memory:      1024 << 9,
	Iterations:  4,
	Parallelism: 2,
	SaltLength:  128,
	KeyLength:   256,
}

func OpenDatabase() {
	var err error

	Database, err = sql.Open("sqlite3", Settings.Authentication.Database)
	if err != nil {
		log.Fatalf("[ERROR] Could not open the database. Error: %s\n", err.Error())
	}

	log.Println("[INFO] Successfully opened the database.")
	log.Println("[INFO] Checking whether the `users` and `logs` tables exist otherwise, creating them.")

	_, err = Database.Exec("CREATE TABLE IF NOT EXISTS `users` (username TEXT NOT NULL PRIMARY KEY, password TEXT NOT NULL, administrator INTEGER NOT NULL, max_time INTEGER NOT NULL,cooldown INTEGER NOT NULL, expires_at INTEGER NOT NULL, banned INTEGER NOT NULL, ban_reason TEXT NOT NULL)")
	if err != nil {
		log.Fatalf("[ERROR] Could not create the users table. Error: %s\n", err.Error())
	}
	_, err = Database.Exec("CREATE TABLE IF NOT EXISTS `logs` ( identifier TEXT NOT NULL PRIMARY KEY, username TEXT NOT NULL, target TEXT NOT NULL, method TEXT NOT NULL, duration TEXT NOT NULL, timestamp TEXT NOT null)")
	if err != nil {
		log.Fatalf("[ERROR] Could not create the logs table. Error: %s\n", err.Error())
	}

	go KeepAlive()
}

func KeepAlive() {
	for {
		if err := Database.Ping(); err != nil {
			log.Fatalf("[ERROR] Could not ping the database connection. Error: %s\n", err.Error())
		}

		time.Sleep(5 * time.Second)
	}
}

func DoesUserExist(username string) (string, bool) {
	var u string

	result := Database.QueryRow("SELECT username FROM users WHERE username=?", username).Scan(&u)
	if result == sql.ErrNoRows {
		return "User does not exists.", false
	} else if result != nil {
		log.Printf("[ERROR] Could not check if the user: %s exists. Error: %s\n", username, result.Error())

		return "Something went wrong", false
	}

	return "User found", true
}

type UserObject struct {
	Username  string
	Admin     bool
	MaxTime   int64
	Cooldown  int64
	ExpiresAt int64
	Banned    bool
	BanReason string
}

func GetUserInfo(username string) (UserObject, string) {
	var user UserObject
	row := Database.QueryRow("SELECT username, administrator, max_time, cooldown, expires_at, banned, ban_reason FROM users")

	err := row.Scan(
		&user.Username,
		&user.Admin,
		&user.MaxTime,
		&user.Cooldown,
		&user.ExpiresAt,
		&user.Banned,
		&user.BanReason,
	)

	if err != nil {
		log.Printf("[ERROR] Something went wrong while fetching a users info. Error: %s\n", err.Error())
		return user, "Something went wrong querying the DB. Try again later"
	}

	return user, ""
}

func GetAllUsers() ([]UserObject, string, bool) {
	var users []UserObject

	rows, err := Database.Query("SELECT username, administrator, max_time, cooldown, expires_at, banned, ban_reason FROM users")
	if err != nil {
		return nil, "", false
	}

	for rows.Next() {
		user := UserObject{}
		err = rows.Scan(
			&user.Username,
			&user.Admin,
			&user.MaxTime,
			&user.Cooldown,
			&user.ExpiresAt,
			&user.Banned,
			&user.BanReason,
		)
		if err != nil {
			log.Printf("[ERROR] Could not scan a user. Error: %s\n", err.Error())
			return nil, "Something went wrong. Check the console", false
		}

		users = append(users, user)
	}

	return users, "", true
}

func CreateUser(username, password string, admin bool, maxTime, cooldown, accessTime int64) (string, bool) {
	_, exists := DoesUserExist(username)
	if exists {
		return "Username is already in use", false
	}

	hashedPass, err := argon2id.CreateHash(password, HashParams)
	if err != nil {
		log.Printf("[ERROR] Could not hash the password on user creation. Error: %s\n", err.Error())
		return "Error hashing the password, check the console..", false
	}

	expiry := time.Now().Add(time.Duration(accessTime) * time.Second)

	_, err = Database.Exec("INSERT INTO users (username, password, max_time, administrator, cooldown, expires_at, banned, ban_reason) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		username, hashedPass, maxTime, admin, cooldown, expiry.Unix(), false, "")
	if err != nil {
		log.Printf("[ERROR] Could not create the new user: %s. Error: %s\n", username, err.Error())
		return "Error occurred creating user, check the console..", false
	}

	log.Printf("[INFO] User: %s has been created. MaxTime: %d Cooldown: %d", username, maxTime, cooldown)
	return "Successfully created the user. Expires on the: " + expiry.Format("01/02/2006 15:04"), true
}

func DeleteUser(username string) (string, bool) {
	msg, exists := DoesUserExist(username)
	if !exists {
		return msg, false
	}

	_, err := Database.Exec("DELETE FROM users WHERE username=?", username)
	if err != nil {
		log.Printf("[ERROR] Could not delete the user: %s. Error: %s\n", username, err.Error())
		return "Error occurred deleting the user, check the console..", false
	}

	log.Printf("[INFO] User: %s has been deleted.", username)
	return "Successfully deleted the user,", true
}

func RenewUser(username string, duration int64) (string, bool) {
	var expiresAt int64

	result := Database.QueryRow("SELECT expires_at FROM users WHERE username=?",
		username).Scan(&expiresAt)
	if result == sql.ErrNoRows {
		log.Printf("[WARNING] You have tried to renew a user that does not exist. Username: %s\n", username)

		return "User does not exists.", false
	}

	newExpiry := time.Now().Add(time.Duration(duration) * time.Second)

	_, err := Database.Exec("UPDATE users SET expires_at=? WHERE username=?", newExpiry.Unix(), username)
	if err != nil {
		log.Printf("[ERROR] Could not alter: %s's expiry date. Error: %s\n", username, err.Error())
		return "Something went wrong, check the console.", false
	}

	return "Successfully renewed the users access until: " + newExpiry.Format("01/02/2006 15:04"), true
}

func BanUser(username, reason string) (string, bool) {
	msg, exists := DoesUserExist(username)
	if !exists {
		return msg, false
	}

	_, err := Database.Exec("UPDATE users SET banned=?, ban_reason=? WHERE username=?",
		true, reason, username)
	if err != nil {
		return "Something went wrong banning the user, check the console.", false
	}

	return "User has been banned successfully for: %s" + reason, true
}

func UnbanUser(username string) (string, bool) {
	msg, exists := DoesUserExist(username)
	if !exists {
		return msg, false
	}

	_, err := Database.Exec("UPDATE users SET banned=?, ban_reason=? WHERE username=?",
		false, "", username)
	if err != nil {
		return "Something went wrong unbanning the user, check the console.", false
	}

	return "User has been unbanned successfully", true
}

func AuthenticateUser(address, username, password string) (string, *User) {
	var (
		plan       User
		dbPassword string
		expiresAt  int64
		banned     bool
		banReason  string
	)

	row := Database.QueryRow("SELECT password, expires_at, banned, ban_reason, administrator, max_time, cooldown FROM users WHERE username=?", username)

	err := row.Scan(&dbPassword, &expiresAt, &banned, &banReason, &plan.Admin, &plan.MaxTime, &plan.Cooldown)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("[ERROR] IP: %s has attempted to login with the incorrect username: %s\n",
				address, username)
		} else {
			log.Printf("[ERROR] Could not authenticate user: %s. Error: %s",
				username, err.Error())
		}

		return "Invalid details,", nil
	}

	passMatch, err := argon2id.ComparePasswordAndHash(password, dbPassword)
	if err != nil {
		log.Printf("[ERROR] Could not check: %s's password. Error: %s",
			username, err.Error())
		return "Internal error, try again.", nil
	}

	if !passMatch {
		log.Printf("[ERROR] IP: %s has attempted to login as the user: %s using an incorrect password.\n",
			address, username)
		return "Invalid details.", nil
	}

	if banned {
		log.Printf("[INFO] Banned user: %s has tried to login. Ban reason: %s\n",
			username, banReason)

		return "You are banned. Reason: " + banReason, nil
	}

	if expiresAt < time.Now().Unix() {
		log.Printf("[INFO] User %s has tried to login eventhough his access has expired.",
			username)

		return "Your access has expired. Contact an admin to renew it.", nil
	}

	log.Printf("[INFO] User %s has logged in.", username)

	return "", &plan
}

func ChangeUserPassword(username, password string) (string, bool) {
	if msg, exists := DoesUserExist(username); !exists {
		return msg, exists
	}

	// hashing the password
	hashedPass, err := argon2id.CreateHash(password, HashParams)
	if err != nil {
		log.Printf("[ERROR] Could not hash the password on passwod change of user: %s. Error: %s\n",
			username, err.Error())
		return "Error hashing the password, check the console.", false
	}

	_, err = Database.Exec("UPDATE users SET password=? WHERE username=?",
		hashedPass, username)
	if err != nil {
		log.Printf("[ERROR] Could not update: %s's password in the database. Error: %s\n",
			username, err.Error())
		return "Error updating the password, check the console or contact an admin.", false
	}

	return fmt.Sprintf("Password updated successfully. New Password: '%s'",
		password), true
}

func LogAttack(a Attack) (string, bool) {
	_, err := Database.Exec("INSERT INTO logs VALUES (?, ?, ?, ?, ?, ?)",
		a.Identifier, a.Attacker, a.Target, a.Method, a.Duration, time.Now().Unix())

	if err != nil {
		log.Printf("[ERROR] Could not log attack of user: %s. Error: %s\n",
			a.Attacker, err.Error())
		return "Something went wrong, try again later or contact an admin", false
	}

	return "", true
}
