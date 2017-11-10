package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/docker/libkv"
	"github.com/docker/libkv/store"
	"github.com/docker/libkv/store/boltdb"
	"github.com/docker/libkv/store/consul"
	"github.com/docker/libkv/store/etcd"
	"github.com/docker/libkv/store/zookeeper"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

var (
	bot     *tgbotapi.BotAPI
	chatIDs []string
	KVStore store.Store
)

func initDB() {
	var err error
	switch opts.Store {
	case "boltdb":
		boltdb.Register()
	case "consul":
		consul.Register()
	case "etcd":
		etcd.Register()
	case "zookeeper":
		zookeeper.Register()
	}

	KVStore, err = libkv.NewStore(store.Backend(opts.Store), opts.DBPath, &store.Config{
		ConnectionTimeout: 10 * time.Second,
		Bucket: "tgDB",
	})

	if err != nil {
		log.Fatalf("kvstore init failed, err: %v", err)
	}

	initData()
}

func initData() {
	pair, err := KVStore.Get("chat_ids")
	if err != nil {
		return
	}
	chatIDs = strings.Split(string(pair.Value), ",")
}

func setUpdates() <-chan tgbotapi.Update {
	upd := tgbotapi.NewUpdate(0)
	upd.Timeout = 60
	updates, err := bot.GetUpdatesChan(upd)
	if err != nil {
		log.Fatalf("Telegram bot get update failed, err: %v", err)
	}
	return updates
}

func message(msg *tgbotapi.Message) {
	if msg.From.ID == bot.Self.ID {
		return
	}

	if !msg.Chat.IsPrivate() {
		if _, err := bot.LeaveChat(tgbotapi.ChatConfig{ChatID: msg.Chat.ID}); err != nil {
			log.Errorf("bot leave chat failed, err: %v", err)
			return
		}
	}

	isCommand := msg.IsCommand()
	isPrivate := msg.Chat.IsPrivate()
	switch {
	case isCommand:
		switch strings.ToLower(msg.Command()) {
		case "start":
			cmdStart(msg)
		case "help":
			cmdHelp(msg)
		case "settings":
			cmdSettings(msg)
		case "stop":
			cmdStop(msg)
		default:
			cmdEasterEgg(msg)
		}
	case !isCommand && msg.ReplyToMessage != nil:
		if msg.ReplyToMessage.Text == "" || msg.Text == "" {
			log.Error("Message empty")
			return
		}
	case !isCommand && isPrivate && msg.Text == "":
		log.Debugf("msg: %+v", msg)
	default:
		msEasterEgg(msg)
	}
}

func contains(slice []string, item string) bool {
	set := make(map[string]struct{}, len(slice))
	for _, s := range slice {
		set[s] = struct{}{}
	}

	_, ok := set[item]
	return ok
}

func cmdStart(msg *tgbotapi.Message) {
	if !contains(chatIDs, cast.ToString(msg.Chat.ID)) {
		log.Infof("New user: @%v", msg.Chat.UserName)
		chatIDs = append(chatIDs, cast.ToString(msg.Chat.ID))
		KVStore.Put("chat_ids", []byte(strings.Join(chatIDs[:], ",")), nil)
	}
}

func remove(s []string, r string) []string {
	for i, v := range s {
		if v == r {
			return append(s[:i], s[i+1:]...)
		}
	}
	return s
}

func cmdStop(msg *tgbotapi.Message) {
	if contains(chatIDs, cast.ToString(msg.Chat.ID)) {
		log.Infof("Remove user: @%v", msg.Chat.UserName)
		chatIDs = remove(chatIDs, cast.ToString(msg.Chat.ID))
		KVStore.Put("chat_ids", []byte(strings.Join(chatIDs[:], ",")), nil)
	}
}

func cmdHelp(msg *tgbotapi.Message) {
}

func cmdSettings(msg *tgbotapi.Message) {
}

func cmdEasterEgg(msg *tgbotapi.Message) {
}

func msEasterEgg(msg *tgbotapi.Message) {
}

func inline(iq *tgbotapi.InlineQuery) {
}

func chosenResult(cr *tgbotapi.ChosenInlineResult) {
}

func callback(cq *tgbotapi.CallbackQuery) {
}

func channelPost(m *tgbotapi.Message) {
}

func main() {
	initDB()
	var err error
	bot, err = tgbotapi.NewBotAPI(opts.BotToken)
	if err != nil {
		log.Fatalf("Create telegram bot api failed, err: %v", err)
	}
	log.Infof("Authorized telegram robot as @%v", bot.Self.UserName)

	bot.Debug = opts.Debug

	updates := make(<-chan tgbotapi.Update)
	updates = setUpdates()

	go func() {
		for upd := range updates {
			switch {
			case upd.Message != nil:
				go message(upd.Message)
			case upd.InlineQuery != nil && len(upd.InlineQuery.Query) <= 255: // Just don't update results if query exceeds the maximum length
				go inline(upd.InlineQuery)
			case upd.ChosenInlineResult != nil:
				go chosenResult(upd.ChosenInlineResult)
			case upd.CallbackQuery != nil:
				go callback(upd.CallbackQuery)
			case upd.ChannelPost != nil:
				go channelPost(upd.ChannelPost)
			}
		}
	}()

	http.HandleFunc("/", testHandler)

	log.Fatal(http.ListenAndServe(opts.ListenAddr, nil))
}

func testHandler(w http.ResponseWriter, req *http.Request) {
	reqDump, _ := httputil.DumpRequest(req, true)
	fmt.Println(string(reqDump))
	go publish(string(reqDump))
	req.Write(w)
}

func callbackHandler(w http.ResponseWriter, req *http.Request) {
}

func publish(text string) {
	for _, v := range chatIDs {
		msg := tgbotapi.NewMessage(cast.ToInt64(v), text)
		msg.ParseMode = "markdown"
		msg.DisableWebPagePreview = true
		bot.Send(msg)
	}
}
