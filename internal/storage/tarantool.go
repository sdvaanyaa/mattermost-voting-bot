package storage

import (
	"context"
	"fmt"
	"github.com/sdvaanyaa/mattermost-voting-bot/internal/config"
	"github.com/sdvaanyaa/mattermost-voting-bot/internal/models"
	"github.com/tarantool/go-tarantool/v2"
)

type Storage struct {
	conn *tarantool.Connection
}

func New(cfg *config.Config) (*Storage, error) {
	const op = "storage.NewStorage"

	dialer := tarantool.NetDialer{
		Address: cfg.TarantoolAddress,
		User:    "guest",
	}

	conn, err := tarantool.Connect(context.Background(), dialer, tarantool.Opts{})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{conn: conn}, nil
}

func (s *Storage) CreatePoll(poll *models.Poll) error {
	const op = "storage.CreatePoll"

	request := tarantool.NewInsertRequest("polls").Tuple([]interface{}{
		poll.ID,
		poll.Question,
		poll.Options,
		poll.CreatedBy,
		poll.ChannelID,
	})

	_, err := s.conn.Do(request).Get()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Storage) GetPoll(pollID string) (*models.Poll, error) {
	const op = "storage.GetPoll"

	data, err := s.conn.Do(
		tarantool.NewSelectRequest("polls").
			Limit(1).
			Iterator(tarantool.IterEq).
			Key([]interface{}{tarantool.StringKey{S: pollID}}),
	).Get()

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// TODO check if len == 0

	poll := &models.Poll{
		ID:        pollID,
		Question:  data[0].(string),
		Options:   data[1].([]string),
		CreatedBy: data[2].(string),
		ChannelID: data[3].(string),
		Active:    data[4].(bool),
	}

	return poll, nil
}

func (s *Storage) ClosePoll(pollID string) error {
	const op = "storage.ClosePoll"

	_, err := s.conn.Do(
		tarantool.NewUpdateRequest("polls").
			Key(tarantool.StringKey{S: pollID}).
			Operations(tarantool.NewOperations().
				Assign(5, true))).
		Get()

	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Storage) DeletePoll(pollID string) error {
	const op = "storage.DeletePoll"

	_, err := s.conn.Do(
		tarantool.NewDeleteRequest("polls").
			Key(tarantool.StringKey{S: pollID})).
		Get()

	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Storage) CreateVote(vote *models.Vote) error {
	const op = "storage.CreateVote"

	request := tarantool.NewInsertRequest("votes").Tuple([]interface{}{
		vote.PollID,
		vote.UserID,
		vote.Option,
	})

	_, err := s.conn.Do(request).Get()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Storage) GetVotes(pollID string) ([]*models.Vote, error) {
	const op = "storage.GetVotes"

	data, err := s.conn.Do(
		tarantool.NewSelectRequest("votes").
			Iterator(tarantool.IterEq).
			Key(tarantool.StringKey{S: pollID})).
		Get()

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	votes := make([]*models.Vote, len(data))
	for i := range data {
		votes[i] = data[i].(*models.Vote)
	}

	return votes, nil
}
