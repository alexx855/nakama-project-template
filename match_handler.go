// Copyright 2020 The Nakama Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"math/rand"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"

	"github.com/heroiclabs/nakama-common/rtapi"
	"github.com/heroiclabs/nakama-common/runtime"
	"github.com/heroiclabs/nakama-project-template/api"
)

const (
	moduleName = "tic-tac-toe"

	tickRate = 5

	maxEmptySec = 30

	delayBetweenGamesSec = 5
	turnTimeFastSec      = 10
	turnTimeNormalSec    = 20
)

var winningPositions = [][]int32{
	{0, 1, 2},
	{3, 4, 5},
	{6, 7, 8},
	{0, 3, 6},
	{1, 4, 7},
	{2, 5, 8},
	{0, 4, 8},
	{2, 4, 6},
}

// Compile-time check to make sure all required functions are implemented.
var _ runtime.Match = &MatchHandler{}

type MatchLabel struct {
	Open int `json:"open"`
	Fast int `json:"fast"`
}

type MatchHandler struct {
	marshaler   *jsonpb.Marshaler
	unmarshaler *jsonpb.Unmarshaler
}

type MatchState struct {
	debug      bool
	random     *rand.Rand
	label      *MatchLabel
	emptyTicks int

	// Currently connected users, or reserved spaces.
	presences map[string]runtime.Presence
	// Number of users currently in the process of connecting to the match.
	joinsInProgress int

	// True if there's a game currently in progress.
	playing bool
	// Current state of the board.
	board []api.Mark
	// Mark assignments to player user IDs.
	marks map[string]api.Mark
	// Whose turn it currently is.
	mark api.Mark
	// Ticks until they must submit their move.
	deadlineRemainingTicks int64
	// The winner of the current game.
	winner api.Mark
	// The winner positions.
	winnerPositions []int32
	// Ticks until the next game starts, if applicable.
	nextGameRemainingTicks int64
}

func (m *MatchHandler) MatchInit(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, params map[string]interface{}) (interface{}, int, string) {
	app, err := firebase.NewApp(context.Background(), nil)
	if err != nil {
		logger.Debug("error initializing app: %v\n", err)
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		logger.Error("err", err)
	}

	defer client.Close()
	// logger.Debug("Firebase admin ready")

	var debug bool
	if d, ok := params["debug"]; ok {
		if dv, ok := d.(bool); ok {
			debug = dv
		}
	}

	if debug {
		logger.Info("match init, starting with debug: %v", debug)
	}

	fast, ok := params["fast"].(bool)
	if !ok {
		logger.Error("invalid match init parameter \"fast\"")
		return nil, 0, ""
	}

	label := &MatchLabel{
		Open: 1,
	}

	if fast {
		label.Fast = 1
		logger.Info("match init with Fast param", label.Fast)
	}

	labelJSON, err := json.Marshal(label)
	if err != nil {
		logger.WithField("error", err).Error("match init failed")
		labelJSON = []byte("{}")
	}

	_, err = client.Collection("tictactoe").Doc(ctx.Value(runtime.RUNTIME_CTX_MATCH_ID).(string)).Set(ctx, map[string]interface{}{
		"playing": false,
		"debug":   debug,
		"random":  rand.New(rand.NewSource(time.Now().UnixNano())),
		"label":   label,
		// "Presences": make(map[string]runtime.Presence, 2),
		"tickRate": tickRate,
	}, firestore.MergeAll)

	return &MatchState{
		debug:     debug,
		random:    rand.New(rand.NewSource(time.Now().UnixNano())),
		label:     label,
		presences: make(map[string]runtime.Presence, 2),
	}, tickRate, string(labelJSON)
}

func (m *MatchHandler) MatchJoinAttempt(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, dispatcher runtime.MatchDispatcher, tick int64, state interface{}, presence runtime.Presence, metadata map[string]string) (interface{}, bool, string) {
	s := state.(*MatchState)

	if s.debug {
		logger.Info("match join attempt username %v user_id %v session_id %v node %v with metadata %v", presence.GetUsername(), presence.GetUserId(), presence.GetSessionId(), presence.GetNodeId(), metadata)
	}

	// Check if it's a user attempting to rejoin after a disconnect.
	if presence, ok := s.presences[presence.GetUserId()]; ok {
		if presence == nil {
			// User rejoining after a disconnect.
			s.joinsInProgress++
			return s, true, ""
		} else {
			// TODO: implement "use here", like whatsapp web
			// User attempting to join from 2 different devices at the same time.
			return s, false, "already joined"
		}
	}

	// Check if match is full.
	if len(s.presences)+s.joinsInProgress >= 2 {
		return s, false, "match full"
	}

	// New player attempting to connect.
	s.joinsInProgress++
	return s, true, ""
}

func (m *MatchHandler) MatchJoin(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, dispatcher runtime.MatchDispatcher, tick int64, state interface{}, presences []runtime.Presence) interface{} {
	s := state.(*MatchState)
	t := time.Now().UTC()

	app, err := firebase.NewApp(context.Background(), nil)
	if err != nil {
		logger.Debug("error initializing app: %v\n", err)
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		logger.Error("err", err)
	}

	defer client.Close()
	// logger.Debug("Firebase admin ready")

	if s.debug {
		for _, presence := range presences {
			logger.Info("match join username %v user_id %v session_id %v node %v", presence.GetUsername(), presence.GetUserId(), presence.GetSessionId(), presence.GetNodeId())
		}
	}

	for _, presence := range presences {
		s.emptyTicks = 0
		s.presences[presence.GetUserId()] = presence
		s.joinsInProgress--

		// Check if we must send a message to this user to update them on the current game state.
		var opCode api.OpCode
		var msg proto.Message
		if s.playing {
			// There's a game still currently in progress, the player is re-joining after a disconnect. Give them a state update.
			opCode = api.OpCode_OPCODE_UPDATE
			msg = &api.Update{
				Board:    s.board,
				Mark:     s.mark,
				Marks:    s.marks,
				Deadline: t.Add(time.Duration(s.deadlineRemainingTicks/tickRate) * time.Second).Unix(),
			}
		} else if s.board != nil && s.marks != nil && s.marks[presence.GetUserId()] > api.Mark_MARK_UNSPECIFIED {
			// There's no game in progress but we still have a completed game that the user was part of.
			// They likely disconnected before the game ended, and have since forfeited because they took too long to return.
			opCode = api.OpCode_OPCODE_DONE
			msg = &api.Done{
				Board: s.board,
				// Mark:            s.mark,
				Marks:           s.marks,
				Winner:          s.winner,
				WinnerPositions: s.winnerPositions,
				NextGameStart:   t.Add(time.Duration(s.nextGameRemainingTicks/tickRate) * time.Second).Unix(),
			}
		}

		// Send a message to the user that just joined, if one is needed based on the logic above.
		if msg != nil {
			var buf bytes.Buffer
			if err := m.marshaler.Marshal(&buf, msg); err != nil {
				logger.Error("error encoding message: %v", err)
			} else {
				dispatcher.BroadcastMessage(int64(opCode), buf.Bytes(), []runtime.Presence{presence}, nil, true)
			}
		}
	}

	// Check if match was open to new players, but should now be closed.
	if len(s.presences) >= 2 && s.label.Open != 0 {
		s.label.Open = 0
		if labelJSON, err := json.Marshal(s.label); err != nil {
			logger.Error("error encoding label: %v", err)
		} else {
			if err := dispatcher.MatchLabelUpdate(string(labelJSON)); err != nil {
				logger.Error("error updating label: %v", err)
			}
		}
	}

	// Update firestore match label
	_, err = client.Collection("tictactoe").Doc(ctx.Value(runtime.RUNTIME_CTX_MATCH_ID).(string)).Set(ctx, map[string]interface{}{
		"label":     s.label,
		"presences": s.presences,
	}, firestore.MergeAll)

	if err != nil {
		return err
	}

	// Update firestore match presences
	// ? implmenet presences as collection, just for fun and test
	// TODO: add status and parse presences data
	// for _, p := range s.presences {
	// 	_, err := client.Collection("tictactoe").Doc(ctx.Value(runtime.RUNTIME_CTX_MATCH_ID).(string)).Collection("presences").Doc(p.GetUserId()).Set(ctx, p)
	// 	if err != nil {
	// 		return err
	// 	}
	// }

	return s
}

func (m *MatchHandler) MatchLeave(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, dispatcher runtime.MatchDispatcher, tick int64, state interface{}, presences []runtime.Presence) interface{} {
	s := state.(*MatchState)

	if s.debug {
		for _, presence := range presences {
			logger.Info("match leave username %v user_id %v session_id %v node %v", presence.GetUsername(), presence.GetUserId(), presence.GetSessionId(), presence.GetNodeId())
		}
	}

	for _, presence := range presences {
		s.presences[presence.GetUserId()] = nil
	}

	app, err := firebase.NewApp(context.Background(), nil)
	if err != nil {
		logger.Debug("error initializing app: %v\n", err)
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		logger.Error("err", err)
	}

	defer client.Close()
	// logger.Debug("Firebase admin ready")

	// Update firestore match label
	_, err = client.Collection("tictactoe").Doc(ctx.Value(runtime.RUNTIME_CTX_MATCH_ID).(string)).Set(ctx, map[string]interface{}{
		"label":     s.label,
		"presences": s.presences,
	}, firestore.MergeAll)

	// Update firestore match presences
	// ? implmenet presences as collection, just for fun and test
	// for _, p := range s.presences {
	// 	_, err := client.Collection("tictactoe").Doc(ctx.Value(runtime.RUNTIME_CTX_MATCH_ID).(string)).Collection("presences").Doc(p.GetUserId()).Set(ctx, p)
	// 	if err != nil {
	// 		return err
	// 	}
	// }

	return s
}

func (m *MatchHandler) MatchLoop(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, dispatcher runtime.MatchDispatcher, tick int64, state interface{}, messages []runtime.MatchData) interface{} {
	s := state.(*MatchState)

	// TODO: do not initialize admin sdk and fiestore client on every tick
	app, err := firebase.NewApp(context.Background(), nil)
	if err != nil {
		logger.Debug("error initializing app: %v\n", err)
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		logger.Error("err", err)
	}

	defer client.Close()
	logger.Debug("Firebase admin ready")

	// matchID := ctx.Value(runtime.RUNTIME_CTX_MATCH_ID).(string)
	// logger.Debug(matchID)

	if s.debug {
		logger.Info("match loop match_id %v tick %v", ctx.Value(runtime.RUNTIME_CTX_MATCH_ID), tick)
		logger.Info("match loop match_id %v message count %v", ctx.Value(runtime.RUNTIME_CTX_MATCH_ID), len(messages))
	}

	if len(s.presences)+s.joinsInProgress == 0 {
		s.emptyTicks++
		if s.emptyTicks >= maxEmptySec*tickRate {
			// Match has been empty for too long, close it.
			logger.Info("closing idle match")

			s.label.Open = 0

			// Update firestore match state
			_, err = client.Collection("tictactoe").Doc(ctx.Value(runtime.RUNTIME_CTX_MATCH_ID).(string)).Set(ctx, map[string]interface{}{
				"label":         s.label,
				"nextGameStart": nil,
				"winner":        api.Mark_MARK_UNSPECIFIED,
				// "Playing":       s.playing,
				// "Deadline":      nil,
			}, firestore.MergeAll)

			return nil
		}
	}

	t := time.Now().UTC()

	// If there's no game in progress check if we can (and should) start one!
	if !s.playing {
		// Between games any disconnected users are purged, there's no in-progress game for them to return to anyway.
		for userID, presence := range s.presences {
			if presence == nil {
				delete(s.presences, userID)
			}
		}

		// Check if we need to update the label so the match now advertises itself as open to join.
		if len(s.presences) < 2 && s.label.Open != 1 {
			s.label.Open = 1
			if labelJSON, err := json.Marshal(s.label); err != nil {
				logger.Error("error encoding label: %v", err)
			} else {
				if err := dispatcher.MatchLabelUpdate(string(labelJSON)); err != nil {
					logger.Error("error updating label: %v", err)
				}
			}
		}

		// Check if we have enough players to start a game.
		if len(s.presences) < 2 {
			return s
		}

		// Check if enough time has passed since the last game.
		if s.nextGameRemainingTicks > 0 {
			s.nextGameRemainingTicks--
			return s
		}

		// We can start a game! Set up the game state and assign the marks to each player.
		s.playing = true
		s.board = make([]api.Mark, 9, 9)
		s.marks = make(map[string]api.Mark, 2)
		marks := []api.Mark{api.Mark_MARK_X, api.Mark_MARK_O}
		for userID := range s.presences {
			s.marks[userID] = marks[0]
			marks = marks[1:]
		}
		s.mark = api.Mark_MARK_X
		s.winner = api.Mark_MARK_UNSPECIFIED
		s.winnerPositions = nil
		s.deadlineRemainingTicks = calculateDeadlineTicks(s.label)
		s.nextGameRemainingTicks = 0

		// Notify the players a new game has started.
		var buf bytes.Buffer
		if err := m.marshaler.Marshal(&buf, &api.Start{
			Board:    s.board,
			Marks:    s.marks,
			Mark:     s.mark,
			Deadline: t.Add(time.Duration(s.deadlineRemainingTicks/tickRate) * time.Second).Unix(),
		}); err != nil {
			logger.Error("error encoding message: %v", err)
		} else {
			dispatcher.BroadcastMessage(int64(api.OpCode_OPCODE_START), buf.Bytes(), nil, nil, true)
		}

		if err != nil {
			logger.Error("Failed adding data to firestore: %v", err)
		}

		// Update firestore match state
		_, err = client.Collection("tictactoe").Doc(ctx.Value(runtime.RUNTIME_CTX_MATCH_ID).(string)).Set(ctx, map[string]interface{}{
			"label":         s.label,
			"playing":       s.playing,
			"board":         s.board,
			"winner":        s.winner,
			"mark":          s.mark,
			"marks":         s.marks,
			"nextGameStart": nil,
			"deadline":      t.Add(time.Duration(s.deadlineRemainingTicks/tickRate) * time.Second).Unix(),
		}, firestore.MergeAll)

		return s
	}

	// There's a game in progress. Check for input, update match state, and send messages to clients.
	for _, message := range messages {

		switch api.OpCode(message.GetOpCode()) {
		case api.OpCode_OPCODE_MOVE:
			mark := s.marks[message.GetUserId()]
			if s.mark != mark {
				// It is not this player's turn.
				logger.Info("It is not this player's turn.")
				dispatcher.BroadcastMessage(int64(api.OpCode_OPCODE_REJECTED), nil, []runtime.Presence{message}, nil, true)
				continue
			}

			msg := &api.Move{}
			err := m.unmarshaler.Unmarshal(bytes.NewReader(message.GetData()), msg)
			if err != nil {
				// Client sent bad data.
				logger.Info("Client sent bad data.")
				dispatcher.BroadcastMessage(int64(api.OpCode_OPCODE_REJECTED), nil, []runtime.Presence{message}, nil, true)
				continue
			}
			if msg.Position < 0 || msg.Position > 8 || s.board[msg.Position] != api.Mark_MARK_UNSPECIFIED {
				// Client sent a position outside the board, or one that has already been played.
				logger.Info(" Client sent a position outside the board, or one that has already been played.")
				dispatcher.BroadcastMessage(int64(api.OpCode_OPCODE_REJECTED), nil, []runtime.Presence{message}, nil, true)
				continue
			}

			// Update the game state.
			s.board[msg.Position] = mark
			switch mark {
			case api.Mark_MARK_X:
				s.mark = api.Mark_MARK_O
			case api.Mark_MARK_O:
				s.mark = api.Mark_MARK_X
			}
			s.deadlineRemainingTicks = calculateDeadlineTicks(s.label)

			// Check if game is over through a winning move.
		winCheck:
			for _, winningPosition := range winningPositions {
				for _, position := range winningPosition {
					if s.board[position] != mark {
						continue winCheck
					}
				}

				// Update state to reflect the winner, and schedule the next game.
				s.winner = mark
				s.winnerPositions = winningPosition
				s.playing = false
				s.deadlineRemainingTicks = 0
				s.nextGameRemainingTicks = delayBetweenGamesSec * tickRate
			}
			// Check if game is over because no more moves are possible.
			tie := true
			for _, mark := range s.board {
				if mark == api.Mark_MARK_UNSPECIFIED {
					tie = false
					break
				}
			}

			if tie {
				// Update state to reflect the tie, and schedule the next game.
				s.playing = false
				s.winner = api.Mark_MARK_UNSPECIFIED
				s.winnerPositions = nil
				s.deadlineRemainingTicks = 0
				s.nextGameRemainingTicks = delayBetweenGamesSec * tickRate
			}

			var deadline = t.Add(time.Duration(s.deadlineRemainingTicks/tickRate) * time.Second).Unix()
			var nextgamestart = t.Add(time.Duration(s.nextGameRemainingTicks/tickRate) * time.Second).Unix()
			var opCode api.OpCode
			var outgoingMsg proto.Message
			if s.playing {
				opCode = api.OpCode_OPCODE_UPDATE
				outgoingMsg = &api.Update{
					Board:    s.board,
					Mark:     s.mark,
					Marks:    s.marks,
					Deadline: deadline,
				}
			} else {
				opCode = api.OpCode_OPCODE_DONE
				outgoingMsg = &api.Done{
					Board: s.board,
					// Mark:            s.mark,
					Marks:           s.marks,
					Winner:          s.winner,
					WinnerPositions: s.winnerPositions,
					NextGameStart:   nextgamestart,
				}
			}

			var buf bytes.Buffer
			if err := m.marshaler.Marshal(&buf, outgoingMsg); err != nil {
				logger.Error("error encoding message: %v", err)
			} else {
				dispatcher.BroadcastMessage(int64(opCode), buf.Bytes(), nil, nil, true)
			}

			if err != nil {
				logger.Error("Failed adding data to firestore: %v", err)
			}

			// Update firestore match state
			_, err = client.Collection("tictactoe").Doc(ctx.Value(runtime.RUNTIME_CTX_MATCH_ID).(string)).Set(ctx, map[string]interface{}{
				"playing":  s.playing,
				"board":    s.board,
				"winner":   s.winner,
				"mark":     s.mark,
				"marks":    s.marks,
				"deadline": deadline,
				// "nextGameStart": nextgamestart,
			}, firestore.MergeAll)

		default:
			// No other opcodes are expected from the client, so automatically treat it as an error.
			dispatcher.BroadcastMessage(int64(api.OpCode_OPCODE_REJECTED), nil, []runtime.Presence{message}, nil, true)
		}
	}

	// Keep track of the time remaining for the player to submit their move. Idle players forfeit.
	if s.playing {
		s.deadlineRemainingTicks--
		if s.deadlineRemainingTicks <= 0 {
			// The player has run out of time to submit their move.
			s.playing = false

			switch s.mark {
			case api.Mark_MARK_X:
				s.winner = api.Mark_MARK_O
			case api.Mark_MARK_O:
				s.winner = api.Mark_MARK_X
			}

			s.winnerPositions = make([]int32, 3)
			s.deadlineRemainingTicks = 0
			s.nextGameRemainingTicks = delayBetweenGamesSec * tickRate

			var buf bytes.Buffer
			if err := m.marshaler.Marshal(&buf, &api.Done{
				Board: s.board,
				// Mark:            s.mark,
				Marks:           s.marks,
				Winner:          s.winner,
				WinnerPositions: s.winnerPositions,
				NextGameStart:   t.Add(time.Duration(s.nextGameRemainingTicks/tickRate) * time.Second).Unix(),
			}); err != nil {
				logger.Error("error encoding message: %v", err)
			} else {
				dispatcher.BroadcastMessage(int64(api.OpCode_OPCODE_DONE), buf.Bytes(), nil, nil, true)
			}

			// Update firestore match state
			_, err = client.Collection("tictactoe").Doc(ctx.Value(runtime.RUNTIME_CTX_MATCH_ID).(string)).Set(ctx, map[string]interface{}{
				"playing":         s.playing,
				"winner":          s.winner,
				"winnerPositions": s.winnerPositions,
				"nextGameStart":   t.Add(time.Duration(s.nextGameRemainingTicks/tickRate) * time.Second).Unix(),
				"mark":            nil,
				"deadline":        nil,
				// "Board":         s.board,
			}, firestore.MergeAll)

		}
	}

	return s
}

func (m *MatchHandler) MatchTerminate(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, dispatcher runtime.MatchDispatcher, tick int64, state interface{}, graceSeconds int) interface{} {
	if state.(*MatchState).debug {
		logger.Info("match terminate match_id %v tick %v", ctx.Value(runtime.RUNTIME_CTX_MATCH_ID), tick)
		logger.Info("match terminate match_id %v grace seconds %v", ctx.Value(runtime.RUNTIME_CTX_MATCH_ID), graceSeconds)
	}
	return state
}

func calculateDeadlineTicks(l *MatchLabel) int64 {
	if l.Fast == 1 {
		return turnTimeFastSec * tickRate
	} else {
		return turnTimeNormalSec * tickRate
	}
}

func beforeChannelJoin(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, envelope *rtapi.Envelope) (*rtapi.Envelope, error) {
	logger.Info("Intercepted request to join channel '%v'", envelope.GetChannelJoin().Target)
	return envelope, nil
}

// func afterGetAccount(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, in *api.Account) error {
// 	logger.Info("Intercepted response to get account '%v'", in)
// 	return nil
// }
