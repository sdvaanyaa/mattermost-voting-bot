package bot

import (
	"encoding/json"
	"fmt"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/sdvaanyaa/mattermost-voting-bot/internal/config"
	"github.com/sdvaanyaa/mattermost-voting-bot/internal/models"
	"github.com/sdvaanyaa/mattermost-voting-bot/internal/storage"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"strings"
)

type Bot struct {
	Client   *model.Client4
	WsClient *model.WebSocketClient
	User     *model.User
	URL      string
	Token    string
	Logger   *slog.Logger
	Storage  *storage.Storage
}

func New(cfg *config.Config, storage *storage.Storage, log *slog.Logger) (*Bot, error) {
	const op = "bot.NewBot"

	client := model.NewAPIv4Client(cfg.MattermostURL)
	client.SetToken(cfg.BotToken)

	user, _, err := client.GetUser("me", "")
	if err != nil {
		log.Error("Failed to log in to Mattermost", slog.Any("error", err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	log.Info("Logged in to Mattermost", slog.String("user", user.Username))

	bot := &Bot{
		Client:  client,
		URL:     cfg.MattermostURL,
		User:    user,
		Token:   cfg.BotToken,
		Logger:  log,
		Storage: storage,
	}

	return bot, nil
}

func (b *Bot) Run() {
	go b.ListenToEvents()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	b.Logger.Info("Shutting down")
	b.Close()
	os.Exit(0)
}

func convertToWebsocketURL(serverURL string) string {
	if strings.HasPrefix(serverURL, "http://") {
		return strings.Replace(serverURL, "http://", "ws://", 1)
	}
	if strings.HasPrefix(serverURL, "https://") {
		return strings.Replace(serverURL, "https://", "wss://", 1)
	}
	return serverURL
}

func (b *Bot) ListenToEvents() {
	const op = "bot.ListenToEvents"

	wsURL := convertToWebsocketURL(b.URL)
	wsClient, err := model.NewWebSocketClient4(wsURL, b.Token)
	if err != nil {
		b.Logger.Error(op, err)
	}

	b.WsClient = wsClient

	b.Logger.Info("Connected to Mattermost WebSocket")

	wsClient.Listen()
	for event := range wsClient.EventChannel {
		go b.handleWebSocketEvent(event)
	}
}

func (b *Bot) Close() {
	if b.WsClient != nil {
		b.WsClient.Close()
		b.Logger.Info("Closed Mattermost WebSocket")
	}
}

func (b *Bot) handleWebSocketEvent(event *model.WebSocketEvent) {
	if event.EventType() != model.WebsocketEventPosted {
		return
	}

	post := &model.Post{}
	if err := json.Unmarshal([]byte(event.GetData()["post"].(string)), post); err != nil {
		b.Logger.Error("Failed to unmarshal post", slog.Any("error", err))
		return
	}

	// Игнорируем сообщения от самого бота
	if post.UserId == b.User.Id {
		return
	}

	b.handlePost(post)
}

func (b *Bot) handlePost(post *model.Post) {
	b.Logger.Debug("Received message", slog.String("message", post.Message))

	switch {
	case strings.HasPrefix(post.Message, "/poll create"):
		b.createPoll(post)
	case strings.HasPrefix(post.Message, "/poll vote"):
		b.vote(post)
	case strings.HasPrefix(post.Message, "/poll results"):
		b.showResults(post)
	case strings.HasPrefix(post.Message, "/poll close"):
		b.closePoll(post)
	case strings.HasPrefix(post.Message, "/poll delete"):
		b.deletePoll(post)
	}
}

func (b *Bot) createPoll(post *model.Post) {
	parts := strings.SplitN(post.Message, " ", 4)
	if len(parts) < 4 {
		b.reply(post, "Usage: /poll create <question> <option1,option2,...>")
		return
	}
	question := parts[2]
	options := strings.Split(parts[3], ",")
	poll := models.NewPoll(question, options, post.UserId, post.ChannelId)
	if err := b.Storage.CreatePoll(poll); err != nil {
		b.Logger.Error("Failed to create poll", slog.Any("error", err))
		b.reply(post, "Error creating poll")
		return
	}
	b.reply(post, fmt.Sprintf("Poll #%s created: %s\nOptions: %v", poll.ID, poll.Question, poll.Options))
	b.Logger.Info("Poll created", slog.String("id", poll.ID))
}

func (b *Bot) vote(post *model.Post) {
	parts := strings.SplitN(post.Message, " ", 4)
	if len(parts) < 4 {
		b.reply(post, "Usage: /poll vote <poll_id> <option_index>")
		return
	}
	pollID := parts[2]
	optionIdx, _ := strconv.Atoi(parts[3]) // Простая конверсия, без проверки ошибок для упрощения
	poll, err := b.Storage.GetPoll(pollID)
	if err != nil || !poll.Active {
		b.reply(post, "Poll not found or closed")
		return
	}
	if optionIdx < 0 || optionIdx >= len(poll.Options) {
		b.reply(post, "Invalid option index")
		return
	}
	vote := &models.Vote{PollID: pollID, UserID: post.UserId, Option: optionIdx}
	if err := b.Storage.CreateVote(vote); err != nil {
		b.Logger.Error("Failed to save vote", slog.Any("error", err))
		b.reply(post, "Error voting")
		return
	}
	b.reply(post, fmt.Sprintf("Voted for '%s' in poll #%s", poll.Options[optionIdx], pollID))
}

func (b *Bot) showResults(post *model.Post) {
	parts := strings.SplitN(post.Message, " ", 3)
	if len(parts) < 3 {
		b.reply(post, "Usage: /poll results <poll_id>")
		return
	}
	pollID := parts[2]
	poll, err := b.Storage.GetPoll(pollID)
	if err != nil {
		b.reply(post, "Poll not found")
		return
	}
	votes, err := b.Storage.GetVotes(pollID)
	if err != nil {
		b.Logger.Error("Failed to get votes", slog.Any("error", err))
		b.reply(post, "Error retrieving results")
		return
	}
	results := make([]int, len(poll.Options))
	for _, vote := range votes {
		results[vote.Option]++
	}
	msg := fmt.Sprintf("Results for #%s: %s\n", pollID, poll.Question)
	for i, opt := range poll.Options {
		msg += fmt.Sprintf("%d. %s: %d votes\n", i, opt, results[i])
	}
	b.reply(post, msg)
}

func (b *Bot) closePoll(post *model.Post) {
	parts := strings.SplitN(post.Message, " ", 3)
	if len(parts) < 3 {
		b.reply(post, "Usage: /poll close <poll_id>")
		return
	}
	pollID := parts[2]
	poll, err := b.Storage.GetPoll(pollID)
	if err != nil || poll.CreatedBy != post.UserId {
		b.reply(post, "Poll not found or you are not the creator")
		return
	}
	if err := b.Storage.ClosePoll(pollID); err != nil {
		b.Logger.Error("Failed to close poll", slog.Any("error", err))
		b.reply(post, "Error closing poll")
		return
	}
	b.reply(post, fmt.Sprintf("Poll #%s closed", pollID))
}

func (b *Bot) deletePoll(post *model.Post) {
	parts := strings.SplitN(post.Message, " ", 3)
	if len(parts) < 3 {
		b.reply(post, "Usage: /poll delete <poll_id>")
		return
	}
	pollID := parts[2]
	poll, err := b.Storage.GetPoll(pollID)
	if err != nil || poll.CreatedBy != post.UserId {
		b.reply(post, "Poll not found or you are not the creator")
		return
	}
	if err := b.Storage.DeletePoll(pollID); err != nil {
		b.Logger.Error("Failed to delete poll", slog.Any("error", err))
		b.reply(post, "Error deleting poll")
		return
	}
	b.reply(post, fmt.Sprintf("Poll #%s deleted", pollID))
}

func (b *Bot) reply(post *model.Post, msg string) {
	b.Client.CreatePost(&model.Post{
		ChannelId: post.ChannelId,
		Message:   msg,
	})
}
