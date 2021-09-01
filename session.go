package main

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"net"
	"strings"
	"sync"
	"time"
)

var ActiveSessions = sync.Map{}

type Session struct {
	Username      string
	ConnectedAt   int64
	RemoteAddress string
	User          User
	Reader        *bufio.Reader
	Conn          net.Conn
}

type User struct {
	Admin    bool
	MaxTime  int64
	Cooldown int64
}

func (s Session) Resize() bool {
	_, err := s.Conn.Write([]byte("\x1b[8;25;120t"))
	if err != nil {
		log.Printf("[ERROR] Could not resize the terminal of: %s. Error: %s", s.RemoteAddress, err.Error())
	}

	return err == nil
}

func (s Session) LiveTitle() {
	var attackStatus string
	for {
		if Settings.Attack.Enabled {
			attackStatus = "enabled"
		} else {
			attackStatus = "disabled"
		}

		title := fmt.Sprintf("%s | Attack slots: %d/%d | Attack Status: %s",
			s.Username, GetAttackSlots(), Settings.Attack.MaxSlots, attackStatus)

		if !s.ChangeTitle(title) {
			break
		}

		sleepTime, _ := rand.Int(rand.Reader, big.NewInt(5))
		time.Sleep(time.Duration(sleepTime.Int64()) * time.Second)
	}
}

func (s Session) ChangeTitle(title string) bool {
	return s.Write(fmt.Sprintf("\033]0;%s\007", title))
}

func (s Session) ClearScreen() {
	s.Write("\033c")
}

func (s *Session) Disconnect(message string) {
	_ = s.Write(fmt.Sprintf("\r\n%s\r\n", message))

	time.Sleep(5 * time.Second)
	_ = s.Conn.Close()

	ActiveSessions.Delete(s.Username)
}

func (s *Session) Write(message string) bool {
	msg := []byte(message)

	_, err := s.Conn.Write(msg)
	if err != nil {
		log.Printf("[ERROR] Could not write to user: %s's connection. Error: %s\n",
			s.Username, err.Error())
	}

	return err == nil
}

func (s *Session) Receive() (string, bool) {
	data, err := s.Reader.ReadString('\n')
	if err != nil {
		if s.Username == "" {
			log.Printf("[ERROR] Could not read from connection: %s. Error: %s\n",
				s.RemoteAddress, err.Error())
		} else {
			log.Printf("[ERROR] Could not read from user: %s's connection. Error: %s\n",
				s.Username, err.Error())
		}

		return "", false
	}

	if data[len(data)-2:] == "\r\n" {
		data = data[:len(data)-2]
	}

	return data, true
}

func (s *Session) WriteAndReceive(prompt string) (string, bool) {
	ok := s.Write(prompt)
	if !ok {
		return "", ok
	}

	return s.Receive()
}

func (s Session) HandleCommand(command string) {
	commandSlice := strings.Split(command, " ")

	switch commandSlice[0] {
	case "cls":
		s.ClearScreen()
		break
	case "clear":
		s.ClearScreen()
		break
	case "c":
		s.ClearScreen()
		break
	case "help":
		s.Write(HelpCommand(s.User.Admin))
		break
	case "?":
		s.Write(HelpCommand(s.User.Admin))
		break
	case "changepass":
		s.Write(ResetSelfPasswordCommand(commandSlice, s.Username) + "\r\n")
		break
	case "methods":
		s.Write(MethodsCommand())
		break
	case "info":
		s.Write(ShowAccountInformation(s))
	}

	if !s.User.Admin {
		return
	}

	switch commandSlice[0] {
	case "adduser":
		s.Write(CreateUserCommand(commandSlice) + "\r\n")
		break
	case "resetpass":
		s.Write(ResetUserPasswordCommand(commandSlice) + "\r\n")
	case "extenduser":
		s.Write(ExtendUserCommand(commandSlice) + "\r\n")
		break
	case "deluser":
		s.Write(DeleteUserCommand(commandSlice) + "\r\n")
		break
	case "sessions":
		s.Write(GetActiveSessions() + "\r\n")
	case "ongoing":
		s.Write(GetOngoingAttacks() + "\r\n")
	case "getusers":
		s.Write(GetUsersCommand() + "\r\n")
		break
	case "ban":
		s.Write(BanUserCommand(commandSlice) + "\r\n")
		break
	case "unban":
		s.Write(UnbanUserCommand(commandSlice) + "\r\n")
	case "dcall":
		s.Write(DisconnectAllCommand(s.Username) + "\r\n")
		break
	case "attacks":
		s.Write(ToggleAttacksCommand(commandSlice) + "\r\n")
		break
	case "reload":
		s.Write(ReloadCommand(commandSlice) + "\r\n")
		break
	}
}
