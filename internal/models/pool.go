package models

import "github.com/google/uuid"

type Poll struct {
	ID        string
	Question  string
	Options   []string
	CreatedBy string
	ChannelID string
	Active    bool
}

func NewPoll(question string, options []string, userID, channelID string) *Poll {
	return &Poll{
		ID:        uuid.New().String(),
		Question:  question,
		Options:   options,
		CreatedBy: userID,
		ChannelID: channelID,
		Active:    true,
	}
}
