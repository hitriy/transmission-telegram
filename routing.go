package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/hitriy/transmission-telegram/bot"
	"github.com/hitriy/transmission-telegram/rutracker"
	"github.com/pyed/transmission"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

var (
	RuTrackerUser string
	RuTrackerPass string
	RTHandler     *bot.RuTrackerHandler
)

const transmissionDisabledMsg = "Transmission is disabled (started with -no-transmission)"

var commandsWithoutTransmission = map[string]bool{
	"help": true, "/help": true,
	"version": true, "/version": true, "ver": true, "/ver": true,
}

var knownCommands = map[string]bool{
	"list": true, "/list": true, "li": true, "/li": true, "ls": true, "/ls": true,
	"head": true, "/head": true, "he": true, "/he": true,
	"tail": true, "/tail": true, "ta": true, "/ta": true,
	"downs": true, "/downs": true, "dg": true, "/dg": true,
	"seeding": true, "/seeding": true, "sd": true, "/sd": true,
	"paused": true, "/paused": true, "pa": true, "/pa": true,
	"checking": true, "/checking": true, "ch": true, "/ch": true,
	"active": true, "/active": true, "ac": true, "/ac": true,
	"errors": true, "/errors": true, "er": true, "/er": true,
	"sort": true, "/sort": true, "so": true, "/so": true,
	"trackers": true, "/trackers": true, "tr": true, "/tr": true,
	"downloaddir": true, "dd": true,
	"add": true, "/add": true, "ad": true, "/ad": true,
	"search": true, "/search": true, "se": true, "/se": true,
	"latest": true, "/latest": true, "la": true, "/la": true,
	"info": true, "/info": true, "in": true, "/in": true,
	"stop": true, "/stop": true, "sp": true, "/sp": true,
	"start": true, "/start": true, "st": true, "/st": true,
	"check": true, "/check": true, "ck": true, "/ck": true,
	"stats": true, "/stats": true, "sa": true, "/sa": true,
	"downlimit": true, "dl": true,
	"uplimit": true, "ul": true,
	"speed": true, "/speed": true, "ss": true, "/ss": true,
	"count": true, "/count": true, "co": true, "/co": true,
	"del": true, "/del": true, "rm": true, "/rm": true,
	"deldata": true, "/deldata": true,
	"help": true, "/help": true,
	"version": true, "/version": true, "ver": true, "/ver": true,
}

func initRuTracker() {
	if RuTrackerUser == "" {
		RuTrackerUser = os.Getenv("RT_USER")
	}
	if RuTrackerPass == "" {
		RuTrackerPass = os.Getenv("RT_PASS")
	}

	if RuTrackerUser == "" || RuTrackerPass == "" {
		logger.Printf("[WARN] RuTracker credentials not set; search and link features disabled")
		return
	}

	client := rutracker.NewClient(RuTrackerUser, RuTrackerPass)
	if err := client.Login(); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] RuTracker login: %s\n", err)
		os.Exit(1)
	}
	logger.Printf("[INFO] RuTracker: logged in as %s", RuTrackerUser)

	RTHandler = &bot.RuTrackerHandler{
		Client:             client,
		TBot:               Bot,
		Trans:              Client,
		Sessions:           bot.NewSessionStore(),
		Logger:             logger,
		EnsureTransmission: ensureTransmission,
	}
}

func ensureTransmission() error {
	if Client != nil {
		return nil
	}

	var err error
	Client, err = transmission.New(RPCURL, Username, Password)
	if err != nil {
		return err
	}
	logger.Printf("[INFO] Transmission connected: %s", RPCURL)

	if RTHandler != nil {
		RTHandler.Trans = Client
	}
	return nil
}

func isKnownCommand(cmd string) bool {
	return knownCommands[strings.ToLower(cmd)]
}

func isRuTrackerURL(text string) bool {
	if RTHandler == nil {
		return false
	}
	_, ok := RTHandler.ExtractTopicID(text)
	return ok
}

func containsMagnetOrHTTP(text string) bool {
	lower := strings.ToLower(text)
	if strings.Contains(lower, "magnet:") {
		return true
	}
	if strings.Contains(lower, "http://") || strings.Contains(lower, "https://") {
		if isRuTrackerURL(text) {
			return false
		}
		return true
	}
	return false
}

func transmissionCommandBlocked(command string) bool {
	return NoTransmission && !commandsWithoutTransmission[strings.ToLower(command)]
}

func dispatchMessage(update tgbotapi.Update) {
	if update.Message.Document != nil {
		if NoTransmission {
			go send(transmissionDisabledMsg, update.Message.Chat.ID, false)
			return
		}
		go receiveTorrent(update)
		return
	}

	text := strings.TrimSpace(update.Message.Text)

	if text != "" && isRuTrackerURL(text) {
		if RTHandler == nil {
			go send("RuTracker is not configured", update.Message.Chat.ID, false)
			return
		}
		go RTHandler.HandleLink(update)
		return
	}

	tokens := strings.Split(text, " ")
	command := ""
	if len(tokens) > 0 {
		command = strings.ToLower(tokens[0])
	}

	if command != "" && isKnownCommand(command) {
		dispatchCommand(update, tokens, command)
		return
	}

	if containsMagnetOrHTTP(text) {
		if NoTransmission {
			go send(transmissionDisabledMsg, update.Message.Chat.ID, false)
			return
		}
		tokens = append([]string{"add"}, tokens...)
		command = "add"
		dispatchCommand(update, tokens, command)
		return
	}

	if text == "" {
		if NoTransmission {
			return
		}
		go receiveTorrent(update)
		return
	}

	if RTHandler != nil {
		go RTHandler.HandleSearch(update)
		return
	}

	if strings.HasPrefix(command, "/") {
		go send("No such command, try /help", update.Message.Chat.ID, false)
		return
	}

	go send("No such command, try /help", update.Message.Chat.ID, false)
}

func dispatchCommand(update tgbotapi.Update, tokens []string, command string) {
	if transmissionCommandBlocked(command) {
		go send(transmissionDisabledMsg, update.Message.Chat.ID, false)
		return
	}

	switch command {
	case "list", "/list", "li", "/li", "/ls", "ls":
		go list(update, tokens[1:])
	case "head", "/head", "he", "/he":
		go head(update, tokens[1:])
	case "tail", "/tail", "ta", "/ta":
		go tail(update, tokens[1:])
	case "downs", "/downs", "dg", "/dg":
		go downs(update)
	case "seeding", "/seeding", "sd", "/sd":
		go seeding(update)
	case "paused", "/paused", "pa", "/pa":
		go paused(update)
	case "checking", "/checking", "ch", "/ch":
		go checking(update)
	case "active", "/active", "ac", "/ac":
		go active(update)
	case "errors", "/errors", "er", "/er":
		go errors(update)
	case "sort", "/sort", "so", "/so":
		go sort(update, tokens[1:])
	case "trackers", "/trackers", "tr", "/tr":
		go trackers(update)
	case "downloaddir", "dd":
		go downloaddir(update, tokens[1:])
	case "add", "/add", "ad", "/ad":
		go add(update, tokens[1:])
	case "search", "/search", "se", "/se":
		go search(update, tokens[1:])
	case "latest", "/latest", "la", "/la":
		go latest(update, tokens[1:])
	case "info", "/info", "in", "/in":
		go info(update, tokens[1:])
	case "stop", "/stop", "sp", "/sp":
		go stop(update, tokens[1:])
	case "start", "/start", "st", "/st":
		go start(update, tokens[1:])
	case "check", "/check", "ck", "/ck":
		go check(update, tokens[1:])
	case "stats", "/stats", "sa", "/sa":
		go stats(update)
	case "downlimit", "dl":
		go downlimit(update, tokens[1:])
	case "uplimit", "ul":
		go uplimit(update, tokens[1:])
	case "speed", "/speed", "ss", "/ss":
		go speed(update)
	case "count", "/count", "co", "/co":
		go count(update)
	case "del", "/del", "rm", "/rm":
		go del(update, tokens[1:])
	case "deldata", "/deldata":
		go deldata(update, tokens[1:])
	case "help", "/help":
		go send(HELP, update.Message.Chat.ID, true)
	case "version", "/version", "ver", "/ver":
		go getVersion(update)
	default:
		go send("No such command, try /help", update.Message.Chat.ID, false)
	}
}
