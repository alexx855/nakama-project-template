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
	"fmt"
	"time"

	"github.com/golang/protobuf/jsonpb"
	"github.com/heroiclabs/nakama-common/runtime"
	"github.com/heroiclabs/nakama-project-template/api"
)

func rpcFindMatch(marshaler *jsonpb.Marshaler, unmarshaler *jsonpb.Unmarshaler) func(context.Context, runtime.Logger, *sql.DB, runtime.NakamaModule, string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {

		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok {
			return "", errNoUserIdFound
		}

		exp, ok := ctx.Value(runtime.RUNTIME_CTX_USER_SESSION_EXP).(int64)
		if !ok || time.Now().Sub(time.Unix(exp, 0)) < 6*time.Hour {
			// 0 uses system expiry settings.
			exp = 0
		}

		vars, ok := ctx.Value(runtime.RUNTIME_CTX_VARS).(map[string]string)
		if !ok {
			vars = map[string]string{} // No session vars so set default.
		}

		users, err := nk.UsersGetId(ctx, []string{userID})
		if err != nil {
			logger.Error("UsersGetId error: %v", err)
			return "", errInternalError
		}

		// Use the latest username in the new token.
		token, exp, err := nk.AuthenticateTokenGenerate(userID, users[0].GetUsername(), exp, vars)
		if err != nil {
			logger.Error("AuthenticateTokenGenerate error: %v", err)
			return "", errInternalError
		}

		logger.Debug("New session with %d expiry time: %v", exp, token)

		var resp struct {
			Session  string   `json:"token"`
			MatchIds []string `protobuf:"bytes,1,rep,name=match_ids,json=matchIds,proto3" json:"match_ids,omitempty"`
		}
		resp.Session = token

		request := &api.RpcFindMatchRequest{}
		if err := unmarshaler.Unmarshal(bytes.NewReader([]byte(payload)), request); err != nil {
			return "", errUnmarshal
		}

		maxSize := 1
		var fast int
		if request.Fast {
			fast = 1
		}
		query := fmt.Sprintf("+label.open:1 +label.fast:%d", fast)

		matchIDs := make([]string, 0, 10)
		matches, err := nk.MatchList(ctx, 10, true, "", nil, &maxSize, query)
		if err != nil {
			logger.Error("error listing matches: %v", err)
			return "", errInternalError
		}
		if len(matches) > 0 {
			// There are one or more ongoing matches the user could join.
			for _, match := range matches {
				matchIDs = append(matchIDs, match.MatchId)
			}
		} else {
			// No available matches found, create a new one.
			matchID, err := nk.MatchCreate(ctx, moduleName, map[string]interface{}{"fast": request.Fast})
			if err != nil {
				logger.Error("error creating match: %v", err)
				return "", errInternalError
			}
			matchIDs = append(matchIDs, matchID)
		}

		resp.MatchIds = matchIDs

		out, err := json.Marshal(resp)
		if err != nil {
			logger.Error("Marshal error: %v", err)
			return "", errMarshal
		}

		logger.Debug("rpcFindMatch resp: %v", string(out))
		return string(out), nil
	}
}

// TODO: implement rpc function that returns the game state, precences and current tick, like the console does
// func (s *ConsoleServer) GetMatchState(ctx context.Context, in *console.MatchStateRequest) (*console.MatchState, error) {
// 	// Validate the match ID.
// 	matchIDComponents := strings.SplitN(in.GetId(), ".", 2)
// 	if len(matchIDComponents) != 2 {
// 		return nil, status.Error(codes.InvalidArgument, "Invalid match ID.")
// 	}
// 	matchID, err := uuid.FromString(matchIDComponents[0])
// 	if err != nil {
// 		return nil, status.Error(codes.InvalidArgument, "Invalid match ID.")
// 	}
// 	node := matchIDComponents[1]
// 	if node == "" {
// 		// Relayed matches don't have a state.
// 		return &console.MatchState{State: ""}, nil
// 	}

// 	presences, tick, state, err := s.matchRegistry.GetState(ctx, matchID, node)
// 	if err != nil {
// 		if err != context.Canceled && err != ErrMatchNotFound {
// 			s.logger.Error("Error getting match state.", zap.Any("in", in), zap.Error(err))
// 		}
// 		if err == ErrMatchNotFound {
// 			return nil, status.Error(codes.InvalidArgument, "Match not found, or match handler already stopped.")
// 		}
// 		return nil, status.Error(codes.Internal, "Error listing matches.")
// 	}

// 	return &console.MatchState{Presences: presences, Tick: tick, State: state}, nil
// }

func rpcGetMatch(marshaler *jsonpb.Marshaler, unmarshaler *jsonpb.Unmarshaler) func(context.Context, runtime.Logger, *sql.DB, runtime.NakamaModule, string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {

		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok {
			return "", errNoUserIdFound
		}

		// if len(payload) > 0 {
		// 	return "", errNoInputAllowed
		// }

		logger.Debug(payload)

		data, err := json.Marshal(payload)
		if err != nil {
			logger.Error("Marshal error: %v", err)
			return "", errMarshal
		}

		logger.Debug("payload data", data)
		// logger.Debug("payload data", json.Unmarshal(data))
		// logger.Debug(payload.matchId)

		// TODO: how  extend &api
		// request := &api.RpcGetMatchRequest{}
		// if err := unmarshaler.Unmarshal(bytes.NewReader([]byte(payload)), request); err != nil {
		// 	return "", errUnmarshal
		// }
		// logger.Debug(request)

		exp, ok := ctx.Value(runtime.RUNTIME_CTX_USER_SESSION_EXP).(int64)
		if !ok || time.Now().Sub(time.Unix(exp, 0)) < 6*time.Hour {
			// 0 uses system expiry settings.
			exp = 0
		}

		vars, ok := ctx.Value(runtime.RUNTIME_CTX_VARS).(map[string]string)
		if !ok {
			vars = map[string]string{} // No session vars so set default.
		}

		users, err := nk.UsersGetId(ctx, []string{userID})
		if err != nil {
			logger.Error("UsersGetId error: %v", err)
			return "", errInternalError
		}

		// Use the latest username in the new token.
		token, exp, err := nk.AuthenticateTokenGenerate(userID, users[0].GetUsername(), exp, vars)
		if err != nil {
			logger.Error("AuthenticateTokenGenerate error: %v", err)
			return "", errInternalError
		}

		logger.Debug("New session with %d expiry time: %v", exp, token)

		var resp struct {
			Session  string   `json:"token"`
			MatchIds []string `protobuf:"bytes,1,rep,name=match_ids,json=matchIds,proto3" json:"match_ids,omitempty"`
		}
		// resp.Session = token
		// resp.MatchIds = matchIDs

		out, err := json.Marshal(resp)
		if err != nil {
			logger.Error("Marshal error: %v", err)
			return "", errMarshal
		}

		return string(out), nil
		// return &MatchState{Presences: presences, Tick: tick, State: state}, nil
	}
}
