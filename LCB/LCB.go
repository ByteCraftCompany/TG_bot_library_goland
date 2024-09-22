package LCB

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"mime/multipart"
	"path/filepath"
	"time"
)

type Bot struct {
	Token        string
	updatesChan  chan Update
	handlers     []Handler
	lastUpdateId int64
	getText   map[int64]string
	state 	 map[int64]map[string]interface{}
	Mu sync.Mutex
}

type Handler struct {
	Filter   Filter
	Callback func(update Update)
}

type Update struct {
	Update_id     int64          `json:"update_id"`
	Message       *Message       `json:"message"`
	CallbackQuery *CallbackQuery `json:"callback_query,omitempty"`
	InlineQuery   *InlineQuery   `json:"inline_query"`
}

type ResponsePostMessage struct {
	Ok bool                      `json:"ok"`
	Result struct {
		MessageID int            `json:"message_id"`
	}                            `json:"result"`
}

type InlineQuery struct {
	ID       string `json:"id"`
	From     *User  `json:"from"`
	Query    string `json:"query"`
	Offset   string `json:"offset"`
	ChatType string `json:"chat_type"`
}

type Message struct {
	Message_id  int64         `json:"message_id"`
	From        *From         `json:"from"`
	Text        *string        `json:"text"`
	Chat        *Chat         `json:"chat"`
	Photo       []PhotoSize   `json:"photo"`
	Caption     string        `json:"caption"`
	ReplyMarkup *reply_markup `json:"reply_markup"`
	Dice 		*Dice 		  `json:"dice"`
}

type Dice struct {
	Emoji string              `json:"emoji"`
	Value int                 `json:"value"`
}

type From struct {
	ID           int64  `json:"id"`
	IsBot        bool   `json:"is_bot"`
	FirstName    string `json:"first_name"`
	LanguageCode string `json:"language_code"`
}

type reply_markup struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data"`
}

type PhotoSize struct {
	File_id      string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	FileSize     int    `json:"file_size"`
}

type Chat struct {
	ID int64 `json:"id"`
}

type CallbackQuery struct {
	ID      string   `json:"id"`
	From    *User    `json:"from"`
	Message *Message `json:"message"`
	Data    string   `json:"data"`
}

type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

type TelegramResponse struct {
	Ok     bool     `json:"ok"`
	Result []Update `json:"result"`
}

type FileResponse struct {
	Ok     bool  `json:"ok"`
	Result *File `json:"result"`
}

type File struct {
	FileID   string `json:"file_id"`
	FilePath string `json:"file_path"`
	FileSize int    `json:"file_size"`
}

type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

type InlineKeyboardButton struct {
	Text         string      `json:"text"`
	URL          string      `json:"url"`
	CallbackData string      `json:"callback_data"`
	WebApp       *WebAppInfo `json:"web_app"`
}

type ReplyKeyboardMarkup struct {
	ReplyKeyboard [][]ReplyKeyboardButton `json:"keyboard"`
	ResizeKeyboard  bool                  `json:"resize_keyboard"`
	OneTimeKeyboard bool                  `json:"one_time_keyboard"`
}

type ReplyKeyboardButton struct {
	Text string                           `json:"text"`
}

type WebAppInfo struct {
	URL string `json:"url"`
}

type Keyboards struct {
	Inline *InlineKeyboardMarkup
	Reply *ReplyKeyboardMarkup
	Delete *DeleteKeyboard
}

type DeleteKeyboard struct {
	Remove_keyboard bool            `json:"remove_keyboard"`
}

type Filter interface {
	Match(update Update) bool
}

type FilterText struct {
	Text string
}

type FilterPhoto struct{}

type FilterCallback struct {
	Callback string
}

type FilterDice struct {
	Emoji string
	Value int
}

func (f FilterText) Match(update Update) bool {
	if update.Message == nil || update.Message.Text == nil {
		return false
	}
	if f.Text == "" {
		return true
	}
	return *update.Message.Text == f.Text
}

func (f FilterPhoto) Match(update Update) bool {
	if update.Message == nil {
		return false
	}
	return len(update.Message.Photo) > 0
}

func (f FilterCallback) Match(update Update) bool {
	if update.CallbackQuery == nil || update.CallbackQuery.Data == "" {
		return false
	}
	return update.CallbackQuery.Data == f.Callback
}

func (f FilterDice) Match(update Update) bool {
	if update.Message == nil || update.Message.Dice == nil{
		return false
	}
	if f.Value == 0 {
		return update.Message.Dice.Emoji == f.Emoji
	}
	return update.Message.Dice.Emoji == f.Emoji && update.Message.Dice.Value == f.Value
}

func NewBot(token string) *Bot {
	return &Bot{
		Token:        token,
		updatesChan:  make(chan Update),
		handlers:     []Handler{},
		lastUpdateId: 0,
		getText:   make(map[int64]string),
		state:     make(map[int64]map[string]interface{}),
		Mu: sync.Mutex{},
	}
}

func (b *Bot) AddHandler(filter Filter, callback func(update Update)) {
	b.handlers = append(b.handlers, Handler{Filter: filter, Callback: callback})
}

func (b *Bot) SetState(userID int64, key string, data interface{}) {
	if b.state[userID] == nil {
		b.state[userID] = make(map[string]interface{})
	}

	b.state[userID][key] = data
}

func (b *Bot) GetState(userID int64, key string) interface{} {
	if b.state == nil {
		b.state = make(map[int64]map[string]interface{})
	}

	if _, ok := b.state[userID]; !ok {
		b.state[userID] = make(map[string]interface{})
		return nil
	}

	if _, ok := b.state[userID][key]; !ok {
		return nil
	}
	return b.state[userID][key]
}

func (b *Bot) CleanState(userID int64) {
	b.state[userID] = nil
}

func (b *Bot) Start() {
	go b.pollUpdates()
	go b.processUpdates()
}

func (b *Bot) pollUpdates() {
	defer close(b.updatesChan)
	for {
		updates, err := b.getUpdates(b.lastUpdateId)
		if err != nil {
			log.Println("Error getting updates:", err)
			continue
		}

		for _, update := range updates {
			if b.lastUpdateId <= update.Update_id {
				b.lastUpdateId = update.Update_id + 1
			}
			b.updatesChan <- update
		}
	}
}

func (b *Bot) processUpdates() {
	flag_stop := false
	for update := range b.updatesChan {
		flag_stop = false
		if update.Message != nil && update.Message.Text != nil {
			b.Mu.Lock()
			for key, _ := range b.getText {
				if key == update.Message.From.ID {
					flag_stop = true
					b.getText[key] = *update.Message.Text
					break
				}
			}
			b.Mu.Unlock()
		}

		if !flag_stop {
			for _, handler := range b.handlers {
				if handler.Filter == nil || handler.Callback == nil {
					continue
				}
				if handler.Filter.Match(update) {
					go handler.Callback(update)
				}
			}
		}
	}
}

func (b *Bot) GetDataFromUser(userID int64) string {
	b.Mu.Lock()
	b.getText[userID] = ""
	b.Mu.Unlock()

	defer func () {
		delete(b.getText, userID)
		b.Mu.Unlock()
	}()
	
	for {
		b.Mu.Lock()
		if b.getText[userID] != "" {
			data := b.getText[userID]
			return data
		}
		b.Mu.Unlock()
		time.Sleep(time.Microsecond * 100)
	}
}

func (b *Bot) SendPhoto(chatID int64, photoPathOrFileID string, caption string, parseMode string, keyboards *Keyboards) int {
    var requestBody io.Reader
    var err error
    var writer *multipart.Writer

    if isFileID(photoPathOrFileID) {
        message := map[string]interface{}{
            "chat_id": chatID,
            "photo":   photoPathOrFileID,
        }

        if caption != "" {
            message["caption"] = caption
        }

		if parseMode != "" {
			message["parse_mode"] = parseMode
		}

        if keyboards != nil {
            if keyboards.Reply != nil {
                message["reply_markup"] = keyboards.Reply
            }
            if keyboards.Inline != nil {
                message["reply_markup"] = keyboards.Inline
            }
        }

        messageJSON, err := json.Marshal(message)
        if err != nil {
            log.Println("Error marshalling message:", err)
            return 0
        }
        requestBody = bytes.NewBuffer(messageJSON)
    } else {
        file, err := os.Open(photoPathOrFileID)
        if err != nil {
            log.Println("Error opening file:", err)
            return 0
        }
        defer file.Close()

        var buffer bytes.Buffer
        writer = multipart.NewWriter(&buffer)

        photoPart, err := writer.CreateFormFile("photo", filepath.Base(photoPathOrFileID))
        if err != nil {
            log.Println("Error creating form file:", err)
            return 0
        }
        _, err = io.Copy(photoPart, file)
        if err != nil {
            log.Println("Error copying file:", err)
            return 0
        }

        err = writer.WriteField("chat_id", fmt.Sprintf("%d", chatID))
        if err != nil {
            log.Println("Error writing field:", err)
            return 0
        }
        if caption != "" {
            err = writer.WriteField("caption", caption)
            if err != nil {
                log.Println("Error writing caption:", err)
                return 0
            }
        }

        if keyboards != nil {
            if keyboards.Reply != nil {
                err = writer.WriteField("reply_markup", serializeKeyboard(keyboards.Reply))
            } else if keyboards.Inline != nil {
                err = writer.WriteField("reply_markup", serializeKeyboard(keyboards.Inline))
            }
            if err != nil {
                log.Println("Error writing keyboard:", err)
                return 0
            }
        }

        writer.Close()
        requestBody = &buffer
    }

    url := "https://api.telegram.org/bot" + b.Token + "/sendPhoto"
    req, err := http.NewRequest("POST", url, requestBody)
    if err != nil {
        log.Println("Error creating request:", err)
        return 0
    }

    if isFileID(photoPathOrFileID) {
        req.Header.Set("Content-Type", "application/json")
    } else {
        req.Header.Set("Content-Type", writer.FormDataContentType())
    }

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        log.Println("Error sending request:", err)
        return 0
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        log.Fatal(err)
        return 0
    }

    var response ResponsePostMessage
    err = json.Unmarshal(body, &response)
    if err != nil {
        log.Fatal(err)
        return 0
    }
    if !response.Ok {
        return 0
    }

    return response.Result.MessageID
}

func isFileID(pathOrID string) bool {
    return len(pathOrID) > 0 && pathOrID[0] == 'A'
}

func serializeKeyboard(keyboard interface{}) string {
    keyboardJSON, err := json.Marshal(keyboard)
    if err != nil {
        log.Println("Error marshalling keyboard:", err)
        return ""
    }
    return string(keyboardJSON)
}

func (b *Bot) DeleteMessage(chatID int64, messageID int64) {
	message := map[string]interface{}{
		"chat_id": chatID,
		"message_id": messageID,
	}

	messageJSON, err := json.Marshal(message)
	if err != nil {
		log.Println("Error marshalling message:", err)
		return
	}

	url := "https://api.telegram.org/bot" + b.Token + "/deleteMessage"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(messageJSON))
	if err != nil {
		log.Println("Error creating request:", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error sending request:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Error response from Telegram: (Status Code: %d)\n", resp.StatusCode)
		return
	}
}
 
func (b *Bot) SendDice(chatID int64, emoji string) int {
	message := map[string]interface{}{
		"chat_id": chatID,
		"emoji":   emoji,
	}

	messageJSON, err := json.Marshal(message)
	if err != nil {
		log.Println("Error marshalling message:", err)
		return 0
	}

	url := "https://api.telegram.org/bot" + b.Token + "/sendDice"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(messageJSON))
	if err != nil {
		log.Println("Error creating request:", err)
		return 0
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error sending request:", err)
		return 0
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
		return 0
	}

	var response ResponsePostMessage
	err = json.Unmarshal(body, &response)
	if err != nil {
		log.Fatal(err)
		return 0
	}
	if !response.Ok {
		return 0
	}

	return response.Result.MessageID
}

func (b *Bot) EditMessage(chatID int64, messageID int64, text string, parseMode string, keyboards *Keyboards) int {
	if len(text) > 1000 {
		text = text[:1000] + "..."
	}

	message := map[string]interface{}{
		"chat_id":    chatID,
		"message_id": messageID,
		"text":       text,
	}

	if parseMode != "" {
		message["parse_mode"] = parseMode
	}

	if keyboards.Reply != nil {
		message["reply_markup"] = keyboards.Reply
	}
	if keyboards.Inline != nil {
		message["reply_markup"] = keyboards.Inline
	}

	messageJSON, err := json.Marshal(message)
	if err != nil {
		log.Println("Error marshalling message:", err)
		return 0
	}

	url := "https://api.telegram.org/bot" + b.Token + "/editMessageText"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(messageJSON))
	if err != nil {
		log.Println("Error creating request:", err)
		return 0
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error sending request:", err)
		return 0
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
		return 0
	}

	var response ResponsePostMessage
	err = json.Unmarshal(body, &response)
	if err != nil {
		log.Fatal(err)
		return 0
	}
	if !response.Ok {
		return 0
	}

	return response.Result.MessageID
}
	
func (b *Bot) SendMessage(chatID int64, text string, parseMode string, keyboards *Keyboards) int {
	if len(text) > 10000 {
		text = text[:10000] + "..."
	}

	message := map[string]interface{}{
		"chat_id": chatID,
		"text":    text,
	}

	if parseMode != "" {
		message["parse_mode"] = parseMode
	}

	if keyboards.Reply != nil {
		message["reply_markup"] = keyboards.Reply
	}
	if keyboards.Inline != nil {
		message["reply_markup"] = keyboards.Inline
	}
	
	messageJSON, err := json.Marshal(message)
	if err != nil {
		log.Println("Error marshalling message:", err)
		return 0
	}

	url := "https://api.telegram.org/bot" + b.Token + "/sendMessage"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(messageJSON))
	if err != nil {
		log.Println("Error creating request:", err)
		return 0
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error sending request:", err)
		return 0
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
		return 0
	}

	var response ResponsePostMessage
	err = json.Unmarshal(body, &response)
	if err != nil {
		log.Fatal(err)
		return 0
	}
	if !response.Ok {
		return 0
	}

	return response.Result.MessageID
}

func (b *Bot) getUpdates(offset int64) ([]Update, error) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?offset=%d", b.Token, offset)
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	var formattedJSON bytes.Buffer
	if err := json.Indent(&formattedJSON, body, "", "  "); err != nil {
		log.Fatalf("Ошибка форматирования JSON: %v", err)
	}
	fmt.Println(formattedJSON.String())

	var updates TelegramResponse
	err = json.Unmarshal(body, &updates)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	if !updates.Ok {
		return nil, fmt.Errorf("Telegram API returned an error:", updates.Result)
	}

	return updates.Result, nil
}

func (b *Bot) DownloadFile(fileName, fileID string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getFile?file_id=%s", b.Token, fileID)

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var fileResponse FileResponse
	err = json.Unmarshal(body, &fileResponse)
	if err != nil {
		return err
	}

	filePath := fileResponse.Result.FilePath
	fileURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", b.Token, filePath)

	out, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer out.Close()

	resp2, err := http.Get(fileURL)
	if err != nil {
		return err
	}
	defer resp2.Body.Close()

	_, err = io.Copy(out, resp2.Body)
	if err != nil {
		return err
	}

	return nil
}