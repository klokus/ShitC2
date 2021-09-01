package main

import (
	"io/ioutil"
	"log"
	"os"
	"strings"

	"gopkg.in/gcfg.v1"
)

type Config struct {
	Server struct {
		Address string
		Port    int
		C2name  string
	}
	Attack struct {
		ApiLink     string
		HttpMethod  string
		MethodsFile string
		Enabled     bool
		MaxSlots    int64
	}
	Authentication struct {
		Database string
	}
}

var Settings Config

func LoadConfig() {
	err := gcfg.ReadFileInto(&Settings, "config.ini")
	if err != nil {
		log.Fatalf("[ERROR] Could not read the config file. Error: %s\n", err.Error())
	}
}

func LoadMethods() {
	methodsFile, err := os.Open(Settings.Attack.MethodsFile)
	if err != nil {
		log.Fatalf("[ERROR] Could not read the methods file. Error: %s\n", err.Error())
	}

	methodsData, err := ioutil.ReadAll(methodsFile)
	if err != nil {
		log.Fatalf("[ERROR] Could not read the methods file. Error: %s\n", err.Error())
	}

	if err = methodsFile.Close(); err != nil {
		log.Fatalf("[ERROR] Could not close the file object. Error: %s\n", err.Error())
	}

	methods := strings.Split(string(methodsData), "\n")
	for _, method := range methods {
		methodData := strings.Split(method, " ")
		Methods[methodData[0]] = strings.Join(methodData[1:], " ")
	}

	log.Printf("[INFO] Loaded %d methods from: %s\n", len(Methods), Settings.Attack.MethodsFile)
}
