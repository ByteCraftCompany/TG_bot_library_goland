# LCB Bot

The LCB (Lightweight Chat Bot) is a Go library for building Telegram bots with a focus on simplicity and flexibility. It supports message handling, file uploads, various types of keyboards, and more. This README provides a guide to getting started, as well as detailed explanations of the key components and functions provided by the library.

## Table of Contents
1. [Getting Started](#getting-started)
    - [Prerequisites](#prerequisites)
    - [Installation](#installation)
2. [Usage](#usage)
    - [Creating a Bot](#creating-a-bot)
    - [Adding Handlers](#adding-handlers)
    - [Sending Messages](#sending-messages)
    - [Sending Photos](#sending-photos)
    - [Working with Keyboards](#working-with-keyboards)
3. [Advanced Features](#advanced-features)
    - [Handling States](#handling-states)
    - [Downloading Files](#downloading-files)
    - [Custom Filters](#custom-filters)
4. [Contributing](#contributing)
5. [License](#license)

## Getting Started

### Prerequisites
- Go 1.16 or later
- A Telegram bot token, which can be obtained from [BotFather](https://t.me/BotFather)

### Installation
To install the LCB library, use the following command:

```bash
go get -u github.com/yourusername/lcb
```

## Usage

### Creating a Bot
To create a new bot, you need to initialize the `Bot` struct with your Telegram bot token:

```go
package main

import (
    "log"
    "LCB"
)

func main() {
    bot := LCB.NewBot("YOUR_TELEGRAM_BOT_TOKEN")
    bot.Start()
    
    // Add handlers here

    // Keep the application running
    select {}
}
```

### Adding Handlers
Handlers allow you to define how your bot should respond to different types of updates. Use the `AddHandler` method to add handlers with specific filters:

```go
bot.AddHandler(LCB.FilterText{Text: "Hello"}, func(update LCB.Update) {
    bot.SendMessage(update.Message.Chat.ID, "Hello, world!", "", nil)
})

bot.AddHandler(LCB.FilterDice{Emoji: "🎲"}, func(update LCB.Update) {
    bot.SendMessage(update.Message.Chat.ID, "Nice roll!", "", nil)
})
```

### Sending Messages
You can send messages using the `SendMessage` method. This method supports optional parameters such as `parseMode` and keyboards:

```go
bot.SendMessage(chatID, "Welcome to the bot!", "", nil)
```

### Sending Photos
To send photos, use the `SendPhoto` method. You can send either a file path or a file ID obtained from previous uploads:

```go
bot.SendPhoto(chatID, "/path/to/photo.jpg", "Here's your photo!", "", nil)
```

### Working with Keyboards
LCB supports inline and reply keyboards. To send a message with a keyboard, create a `Keyboards` struct and pass it to the `SendMessage` or `SendPhoto` methods:

```go
inlineKeyboard := LCB.Keyboards{
    Inline: &LCB.InlineKeyboardMarkup{
        InlineKeyboard: [][]LCB.InlineKeyboardButton{
            {{Text: "Option 1", CallbackData: "option1"}},
            {{Text: "Option 2", CallbackData: "option2"}},
        },
    },
}

bot.SendMessage(chatID, "Choose an option:", "", &inlineKeyboard)
```

## Advanced Features

### Handling States
You can use the state management functions to store and retrieve data for users. This is useful for managing user sessions:

```go
bot.SetState(userID, "step", 1)
step := bot.GetState(userID, "step").(int)

if step == 1 {
    bot.SendMessage(userID, "What is your name?", "", nil)
} else if step == 2 {
    name := bot.GetDataFromUser(userID)
    bot.SendMessage(userID, "Hello, "+name+"!", "", nil)
}
```

### Downloading Files
You can download files sent to your bot using the `DownloadFile` method:

```go
err := bot.DownloadFile("local_path.jpg", fileID)
if err != nil {
    log.Println("Error downloading file:", err)
}
```

### Custom Filters
You can implement your own filters by creating a struct that implements the `Filter` interface:

```go
type FilterByUsername struct {
    Username string
}

func (f FilterByUsername) Match(update LCB.Update) bool {
    return update.Message != nil && update.Message.From.Username == f.Username
}

bot.AddHandler(FilterByUsername{Username: "specific_user"}, func(update LCB.Update) {
    bot.SendMessage(update.Message.Chat.ID, "Hello, special user!", "", nil)
})
```

## Contributing
We welcome contributions! Please follow these steps to contribute to the project:
1. Fork the repository.
2. Create a feature branch (`git checkout -b feature-name`).
3. Commit your changes (`git commit -am 'Add new feature'`).
4. Push to the branch (`git push origin feature-name`).
5. Open a pull request.

Please ensure that your code adheres to the existing code style and includes relevant tests.

## License
This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
