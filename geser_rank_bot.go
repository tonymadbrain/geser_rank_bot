package main

import (
	"container/heap"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"go.uber.org/zap"
)

var (
	rankDumpFile          string = "db/grb_rank.dump"
	messageToVoteDumpFile string = "db/grb_messageToVote.dump"
	rank                         = map[int64]map[int]int64{}
	messageToVote                = map[int64]map[int]map[int]bool{}
)

func fullPath(file string) (string, error) {
	path, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s", path, file), nil
}

func dumpLoop(logger *zap.SugaredLogger, mutex *sync.RWMutex) {
	// infite loop
	for {
		logger.Info("dump started")
		// rank
		logger.Info("dump rank")
		mutex.RLock()
		rankData, err := json.Marshal(rank)
		mutex.RUnlock()

		if err != nil {
			logger.Errorf("can't marshal rank: %s", err)
		} else {
			// rankDataStr := string(rankData)
			// log.Println("rankDump: " + rankDataStr)
			rankDumpPath, _ := fullPath(rankDumpFile)
			rankDumpBakPath, _ := fullPath(fmt.Sprintf("%s.bak", rankDumpFile))
			err := os.Rename(rankDumpPath, rankDumpBakPath)
			if err != nil {
				logger.Warnf("can't move previous rank dump: %s", err)
			}

			err = ioutil.WriteFile(rankDumpPath, rankData, 0644)
			if err != nil {
				logger.Errorf("can't dump rank: %s", err)
			}
		}

		// messageToVote
		logger.Info("dump messageToVote")
		mutex.RLock()
		messageToVoteData, err := json.Marshal(messageToVote)
		mutex.RUnlock()

		if err != nil {
			logger.Errorf("can't marshal messageToVote: %s", err)
		} else {
			messageToVoteDumpPath, _ := fullPath(messageToVoteDumpFile)
			messageToVoteDumpBakPath, _ := fullPath(fmt.Sprintf("%s.bak", messageToVoteDumpFile))
			err := os.Rename(messageToVoteDumpPath, messageToVoteDumpBakPath)
			if err != nil {
				logger.Warnf("can't move previous messageToVote dump: %s", err)
			}
			err = ioutil.WriteFile(messageToVoteDumpPath, messageToVoteData, 0644)
			if err != nil {
				logger.Errorf("can't dump messageToVote: %s", err)
			}
		}

		logger.Info("dump finished")
		// timer
		time.Sleep(60 * time.Second)
	}
}

func dumpLoadFromFile(dumpFile string, destination interface{}) error {
	dumpPath, _ := fullPath(dumpFile)
	dump, err := ioutil.ReadFile(dumpPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("can't read dump file: %s", err)
	}

	err = json.Unmarshal(dump, destination)
	if err != nil {
		return fmt.Errorf("can't unmarshal rank dump: %s", err)
	}

	return nil
}

func dumpLoad() error {
	wg := &sync.WaitGroup{}
	var rankErr, messageToVoteErr error

	wg.Add(1)
	go func() {
		rankErr = dumpLoadFromFile(rankDumpFile, &rank)
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		messageToVoteErr = dumpLoadFromFile(messageToVoteDumpFile, &messageToVote)
		wg.Done()
	}()

	wg.Wait()
	if rankErr != nil || messageToVoteErr != nil {
		return fmt.Errorf("dump load error: %s, %s", rankErr, messageToVoteErr)
	}

	return nil
}

func getHeap(m map[int]int64) *KVHeap {
	h := &KVHeap{}
	heap.Init(h)
	for k, v := range m {
		heap.Push(h, kv{k, v})
	}
	return h
}

type kv struct {
	Key   int
	Value int64
}

// KVHeap type for converting from Map to Heap
type KVHeap []kv

func (h KVHeap) Len() int           { return len(h) }
func (h KVHeap) Less(i, j int) bool { return h[i].Value > h[j].Value }
func (h KVHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

// Push method
func (h *KVHeap) Push(x interface{}) {
	*h = append(*h, x.(kv))
}

// Pop method
func (h *KVHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

func processMessage(logger *zap.SugaredLogger, bot *tgbotapi.BotAPI, mutex *sync.RWMutex, message *tgbotapi.Message) {
	if message.From.IsBot {
		return
	}

	// TODO: Check if message about new chat or left chat
	// if message.NewChatMembers != nil && message.NewChatMembers == bot.Self.ID {
	// 	log.Printf("bot added to new chat ''")
	// }

	if message.Text == "/top_rank" {
		// Create a heap from the map and print the top N values.
		h := getHeap(rank[message.Chat.ID])
		n := 10
		if n > h.Len() {
			n = h.Len()
		}
		res := fmt.Sprintf("Top %d rank for this chat:\n", n)
		for i := 0; i < n; i++ {
			userRank := heap.Pop(h).(kv)
			chatMember, _ := bot.GetChatMember(tgbotapi.ChatConfigWithUser{ChatID: message.Chat.ID, UserID: userRank.Key})
			line := fmt.Sprintf("@%s: %d", chatMember.User.UserName, userRank.Value)
			res = fmt.Sprintf("%s%s\n", res, line)
		}
		msg := tgbotapi.NewMessage(message.Chat.ID, res)
		msg.ReplyToMessageID = message.MessageID
		bot.Send(msg)
		return
	}

	// TODO: check by array of words from config
	if !strings.HasPrefix(message.Text, "+") {
		return
	}

	if message.ReplyToMessage == nil {
		return
	}

	endorsedID := message.ReplyToMessage.From.ID             // what if he is bot?
	endorsedUserName := message.ReplyToMessage.From.UserName // what if he is bot?
	endorserID := message.From.ID
	endorserUserName := message.From.UserName

	if endorsedID == endorserID {
		logger.Infow("self-abuse try detected",
			"chat", message.Chat.Title,
			"message", message.Text,
			"reply", message.ReplyToMessage.Text,
			"endorsed", endorsedUserName,
			"endorser", endorserUserName,
		)
		msgText := fmt.Sprintf("@%s self-abuse not allowed", endorsedUserName)
		msg := tgbotapi.NewMessage(message.Chat.ID, msgText)
		msg.ReplyToMessageID = message.MessageID
		bot.Send(msg)
		return
	}

	if messageToVote[message.Chat.ID][message.ReplyToMessage.MessageID][endorserID] {
		logger.Infow("multiple votes try detected",
			"chat", message.Chat.Title,
			"message", message.Text,
			"origin", message.ReplyToMessage.Text,
			"endorsed", endorsedUserName,
			"endorser", endorserUserName,
		)
		msgText := fmt.Sprintf("@%s multiple votes not allowed", endorserUserName)
		msg := tgbotapi.NewMessage(message.Chat.ID, msgText)
		msg.ReplyToMessageID = message.MessageID
		bot.Send(msg)
		return
	}

	if endorsedID == bot.Self.ID {
		msgText := fmt.Sprintf("@%s my rank is absolute", endorserUserName)
		msg := tgbotapi.NewMessage(message.Chat.ID, msgText)
		msg.ReplyToMessageID = message.MessageID
		bot.Send(msg)
		return
	}

	// check if messages in the same group
	if message.Chat.ID == message.ReplyToMessage.Chat.ID {
		mutex.Lock()
		if rank[message.Chat.ID] == nil {
			logger.Infow("new chat added",
				"chatID", message.Chat.ID,
				"chatTitle", message.Chat.Title,
			)
			rank[message.Chat.ID] = map[int]int64{}
			rank[message.Chat.ID][endorsedID] = 1
		} else {
			if _, ok := rank[message.Chat.ID][endorsedID]; ok {
				rank[message.Chat.ID][endorsedID]++
			} else {
				rank[message.Chat.ID][endorsedID] = 1
			}
		}

		if messageToVote[message.Chat.ID] == nil {
			messageToVote[message.Chat.ID] = map[int]map[int]bool{}
		}

		if _, ok := messageToVote[message.Chat.ID][message.ReplyToMessage.MessageID]; ok {
			messageToVote[message.Chat.ID][message.ReplyToMessage.MessageID][endorserID] = true
		} else {
			messageToVote[message.Chat.ID][message.ReplyToMessage.MessageID] = map[int]bool{}
			messageToVote[message.Chat.ID][message.ReplyToMessage.MessageID][endorserID] = true
		}
		mutex.Unlock()

		msgText := fmt.Sprintf("@%s's rank is now %d", endorsedUserName, rank[message.Chat.ID][endorsedID])
		msg := tgbotapi.NewMessage(message.Chat.ID, msgText)
		// msg.Text = fmt.Sprintf("%s rank is now %d", endorsedUserName, rank[message.Chat.ID][endorsedID])
		msg.ReplyToMessageID = message.MessageID
		bot.Send(msg)
	}
}

func configureLogger() *zap.SugaredLogger {
	rawJSON := []byte(`{
		"level": "info",
		"development": false,
	  "encoding": "json",
	  "outputPaths": ["stdout"],
		"errorOutputPaths": ["stdout"],
	  "encoderConfig": {
			"timeEncoder": "RFC3339TimeEncoder",
			"timeKey": "timestamp",
	    "messageKey": "message",
	    "levelKey": "level",
			"levelEncoder": "lowercase",
			"callerEncoder": "ShortCallerEncoder",
			"callerKey": "caller"
	  }
	}`)

	var cfg zap.Config
	if err := json.Unmarshal(rawJSON, &cfg); err != nil {
		panic(err)
	}

	zapLogger, err := cfg.Build()
	if err != nil {
		panic(err)
	}

	defer zapLogger.Sync()
	zapLogger.Info("logger construction succeeded")
	return zapLogger.Sugar()
}

func main() {
	logger := configureLogger()
	logger.Infof("logger initiated")

	token := os.Getenv("TG_BOT_TOKEN")
	if token == "" {
		logger.Error("no TG_BOT_TOKEN set")
		os.Exit(1)
	}

	err := dumpLoad()
	if err != nil {
		logger.Error(err)
		os.Exit(1)
	}
	logger.Infof("dump loading finished")

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		logger.Error("can't authorize bot: %s", err)
		os.Exit(1)
	}
	logger.Infof("bot authorized on account %s", bot.Self.UserName)

	if os.Getenv("TG_BOT_DEBUG") == "true" {
		bot.Debug = true
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	mutex := &sync.RWMutex{}

	go dumpLoop(logger, mutex)
	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}
		// log.Printf("message: %s", update.Message.Text)
		// litter.Dump(update)

		go processMessage(logger, bot, mutex, update.Message)
	}
}
