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
	"github.com/heroiclabs/nakama-common/runtime"
	"google.golang.org/api/option"
)

// Refresh a session token which is close to expiry.
func rpcRefresh(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	opt := option.WithCredentialsFile("/nakama/data/modules/service-account.json")
	app, err := firebase.NewApp(context.Background(), nil, opt)

	if err != nil {
		logger.Info("error initializing app: %v\n", err)
		return "", errInternalError
	}

	logger.Info("Firebase admin ready", app)

	client, err := app.Auth(ctx)
	if err != nil {
			logger.Error("error getting Auth client: %v\n", err)
			return "", errInternalError
	}
	
	// TODO: get ID token from header authorization
	idToken:= "eyJhbGciOiJSUzI1NiIsImtpZCI6IjljZTVlNmY1MzBiNDkwMTFiYjg0YzhmYWExZWM1NGM1MTc1N2I2NTgiLCJ0eXAiOiJKV1QifQ.eyJpc3MiOiJodHRwczovL3NlY3VyZXRva2VuLmdvb2dsZS5jb20vbWVkaWV2YWxnb2RzLTVjYmMzIiwiYXVkIjoibWVkaWV2YWxnb2RzLTVjYmMzIiwiYXV0aF90aW1lIjoxNjExODQ3NDUyLCJ1c2VyX2lkIjoiSXlxWVNaOGN5T1A1c0NOQndIbElQcFkxYUdFMiIsInN1YiI6Ikl5cVlTWjhjeU9QNXNDTkJ3SGxJUHBZMWFHRTIiLCJpYXQiOjE2MTIzNTU2OTQsImV4cCI6MTYxMjM1OTI5NCwiZW1haWwiOiJpbmZvQGFsZXhwZWRlcnNlbi5uZXQiLCJlbWFpbF92ZXJpZmllZCI6ZmFsc2UsImZpcmViYXNlIjp7ImlkZW50aXRpZXMiOnsiZW1haWwiOlsiaW5mb0BhbGV4cGVkZXJzZW4ubmV0Il19LCJzaWduX2luX3Byb3ZpZGVyIjoicGFzc3dvcmQifX0.TVZXHc8REYe-4SvrRID_tXSJup6dGZhQUQFHwBN1svCjcxrQdekhNJDo5Ti8_Su74JwULLjJLhvK8W8rQhKuiXBSxbOvgO8qdQHD9nnl7l2Fsd87f9NdXTHXavQQ9XWs0X5dMnbmIOCh-nSujsr0XkXpeCUc1FVPMwEM059QxkBqqUJwKDzRO8QKWUaffB_YujY56QF9LNYSA_uBUWi2LpuYREUMd1m8aGwk5fMDMd3uWJ36Dm6nPuG6wZLt6cNsa-GUk4Js4CBfRA8ywnYYTcBiBlzUEu37TLeyezFROm_JCIk90GyzuE-BdTJ91HljjFThY7edUhDoedCfRxkIAg"
	firebaseIDToken, err := client.VerifyIDToken(ctx, idToken)
	if err != nil {
			logger.Error("error verifying ID token: %v\n", err)
			return "", errInternalError
	}
	
	logger.Info("Verified ID token: %v\n", firebaseIDToken)

	userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
	if !ok {
		return "", errNoUserIdFound
	}

	if len(payload) > 0 {
		return "", errNoInputAllowed
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
		Session string `json:"token"`
	}
	resp.Session = token

	out, err := json.Marshal(resp)
	if err != nil {
		logger.Error("Marshal error: %v", err)
		return "", errMarshal
	}

	logger.Debug("rpcRefresh resp: %v", string(out))
	return string(out), nil
}
