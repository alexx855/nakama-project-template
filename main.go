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
	"context"
	"database/sql"
	"encoding/json"
	"time"

	firebase "firebase.google.com/go"
	"github.com/golang/protobuf/jsonpb"
	"github.com/heroiclabs/nakama-common/api"
	"github.com/heroiclabs/nakama-common/runtime"
	"google.golang.org/api/option"
)

var (
	errInternalError  = runtime.NewError("internal server error", 13) // INTERNAL
	errMarshal        = runtime.NewError("cannot marshal type", 13)   // INTERNAL
	errNoInputAllowed = runtime.NewError("no input allowed", 3)       // INVALID_ARGUMENT
	errNoUserIdFound  = runtime.NewError("no user ID in context", 3)  // INVALID_ARGUMENT
	errUnmarshal      = runtime.NewError("cannot unmarshal type", 13) // INTERNAL
)

const (
	rpcIdRefresh   = "refreshes"
	rpcIdRewards   = "rewards"
	rpcIdFindMatch = "find_match"
	rpcIdGetMatch  = "get_match"
)

// func SetSessionVars(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, in *api.AuthenticateCustomRequest) (*api.AuthenticateCustomRequest, error) {
// 	logger.Info("User session contains key-value pairs set the client: %v", in.GetAccount().Vars)

// 	if in.GetAccount().Vars == nil {
// 		in.GetAccount().Vars = map[string]string{}
// 	}
// 	in.GetAccount().Vars["firebase_uid"] = "firebase_uid"

// 	return in, nil
// }

func AccessSessionVars(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule) error {
	vars, ok := ctx.Value(runtime.RUNTIME_CTX_VARS).(map[string]string)
	if !ok {
		logger.Info("User session does not contain any key-value pairs set")
		return nil
	}

	logger.Info("User session contains key-value pairs set by both the client and the before authentication hook: %v", vars)
	return nil
}

func beforeAuthenticateCustom(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, in *api.AuthenticateCustomRequest) (*api.AuthenticateCustomRequest, error) {
	customIDAuthToken := in.GetAccount().GetId()
	// call our custom server to authenticate with the customIDAuthToken and receive a user ID back

	logger.Info("customIDAuthToken", customIDAuthToken)

	// TODO: get service account from env.
	ctx, opt := context.Background(), option.WithCredentialsFile("/nakama/data/modules/service-account.json")
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		logger.Debug("error initializing app: %v\n", err)
		return in, err
	}

	logger.Debug("Firebase admin ready")

	client, err := app.Auth(ctx)
	if err != nil {
		logger.Error("error getting Auth client: %v\n", err)
		return in, err
	}

	firebaseIDToken, err := client.VerifyIDToken(ctx, customIDAuthToken)
	if err != nil {
		// Auth token not valid or expired.
		logger.Error("error verifying ID token: %v\n", err)
		return in, err
	}

	logger.Debug("Verified ID token:")
	logger.Debug(firebaseIDToken.UID)

	if in.GetAccount().Vars == nil {
		in.GetAccount().Vars = map[string]string{}
	}
	// in.GetAccount().Vars["firebase_uid"] = "firebase_uid"

	customID := firebaseIDToken.UID
	// Replace token with the verified custom ID so Nakama can persist it
	in.GetAccount().Id = customID
	// set this in the sessions Vars so Nakama can embed it in every authentication token.
	in.GetAccount().Vars["firebase_uid"] = customID

	// in.Account.Id = customID                   // Replace token with the verified custom ID so Nakama can persist it
	// in.Account.Vars["firebase_uid"] = customID // set this in the sessions Vars so Nakama can embed it in every authentication token.

	return in, nil
}

func beforeLeaderboardWrite(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, in *api.WriteLeaderboardRecordRequest) *api.WriteLeaderboardRecordRequest {
	vars, ok := ctx.Value(runtime.RUNTIME_CTX_VARS).(map[string]string)
	if !ok || vars["firebase_uid"] == "" {
		logger.Info("session vars expected but missing.", runtime.NewError("session vars expected but missing", 3))
		return nil
	}

	customID := vars["firebase_uid"]
	// insert the session vars with the customID into each LeaderboardRecord
	// which makes it very easy for you read later when getting/listing the records.
	jsonBytes, _ := json.Marshal(map[string]string{"firebase_uid": customID})
	in.Record.Metadata = string(jsonBytes)
	return in
}

//noinspection GoUnusedExportedFunction
func InitModule(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, initializer runtime.Initializer) error {
	initStart := time.Now()

	// ctx, opt := context.Background(), option.WithCredentialsFile("/nakama/data/modules/service-account.json")
	// app, err := firebase.NewApp(ctx, nil, opt)
	// if err != nil {
	// 	logger.Info("error initializing app: %v\n", err)
	// 	return err
	// }

	// logger.Info("Firebase admin ready", app)

	if err := initializer.RegisterBeforeAuthenticateCustom(beforeAuthenticateCustom); err != nil {
		logger.Error("Unable to register: %v", err)
		return err
	}

	// if err := initializer.RegisterBeforeWriteLeaderboardRecord(beforeLeaderboardWrite); err != nil {
	// 	return err
	// }

	if err := initializer.RegisterBeforeGetAccount(AccessSessionVars); err != nil {
		logger.Error("Unable to register: %v", err)
		return err
	}

	// ?? implement
	// if err := initializer.RegisterAfterGetAccount(afterGetAccount); err != nil {
	// 	return err
	// }

	marshaler := &jsonpb.Marshaler{
		EnumsAsInts: true,
	}
	unmarshaler := &jsonpb.Unmarshaler{
		AllowUnknownFields: false,
	}

	if err := initializer.RegisterBeforeRt("ChannelJoin", beforeChannelJoin); err != nil {
		return err
	}

	if err := initializer.RegisterRpc(rpcIdRefresh, rpcRefresh); err != nil {
		return err
	}

	if err := initializer.RegisterRpc(rpcIdRewards, rpcRewards); err != nil {
		return err
	}

	if err := initializer.RegisterRpc(rpcIdFindMatch, rpcFindMatch(marshaler, unmarshaler)); err != nil {
		return err
	}

	if err := initializer.RegisterRpc(rpcIdGetMatch, rpcGetMatch(marshaler, unmarshaler)); err != nil {
		return err
	}

	if err := initializer.RegisterMatch(moduleName, func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule) (runtime.Match, error) {
		return &MatchHandler{
			marshaler:   marshaler,
			unmarshaler: unmarshaler,
		}, nil
	}); err != nil {
		return err
	}

	if err := registerSessionEvents(db, nk, initializer); err != nil {
		return err
	}

	logger.Info("Plugin loaded in '%d' msec.", time.Now().Sub(initStart).Milliseconds())
	return nil
}
