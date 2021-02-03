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

	firebase "firebase.google.com/go"
	"google.golang.org/api/option"

	"github.com/golang/protobuf/jsonpb"
	"github.com/heroiclabs/nakama-common/runtime"
)

func rpcFindMatch(marshaler *jsonpb.Marshaler, unmarshaler *jsonpb.Unmarshaler) func(context.Context, runtime.Logger, *sql.DB, runtime.NakamaModule, string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		
		opt := option.WithCredentialsFile("/nakama/data/modules/service-account.json")
		app, err := firebase.NewApp(context.Background(), nil, opt)
	
		if err != nil {
			logger.Info("error initializing app: %v\n", err)
			return "1", errInternalError
		}
	
		logger.Info("Firebase admin ready", app)
	
		client, err := app.Auth(ctx)
		if err != nil {
				logger.Error("error getting Auth client: %v\n", err)
				return "2", errInternalError
		}

		// Obtener del body como en el pokemon https://heroiclabs.com/docs/runtime-code-basics/#context
		// logger.Info(ctx.Value())
		
		// TODO: get ID token from header authorization
		idToken:= "eyJhbGciOiJSUzI1NiIsImtpZCI6IjljZTVlNmY1MzBiNDkwMTFiYjg0YzhmYWExZWM1NGM1MTc1N2I2NTgiLCJ0eXAiOiJKV1QifQ.eyJpc3MiOiJodHRwczovL3NlY3VyZXRva2VuLmdvb2dsZS5jb20vbWVkaWV2YWxnb2RzLTVjYmMzIiwiYXVkIjoibWVkaWV2YWxnb2RzLTVjYmMzIiwiYXV0aF90aW1lIjoxNjExODQ3NDUyLCJ1c2VyX2lkIjoiSXlxWVNaOGN5T1A1c0NOQndIbElQcFkxYUdFMiIsInN1YiI6Ikl5cVlTWjhjeU9QNXNDTkJ3SGxJUHBZMWFHRTIiLCJpYXQiOjE2MTIzNTU2OTQsImV4cCI6MTYxMjM1OTI5NCwiZW1haWwiOiJpbmZvQGFsZXhwZWRlcnNlbi5uZXQiLCJlbWFpbF92ZXJpZmllZCI6ZmFsc2UsImZpcmViYXNlIjp7ImlkZW50aXRpZXMiOnsiZW1haWwiOlsiaW5mb0BhbGV4cGVkZXJzZW4ubmV0Il19LCJzaWduX2luX3Byb3ZpZGVyIjoicGFzc3dvcmQifX0.TVZXHc8REYe-4SvrRID_tXSJup6dGZhQUQFHwBN1svCjcxrQdekhNJDo5Ti8_Su74JwULLjJLhvK8W8rQhKuiXBSxbOvgO8qdQHD9nnl7l2Fsd87f9NdXTHXavQQ9XWs0X5dMnbmIOCh-nSujsr0XkXpeCUc1FVPMwEM059QxkBqqUJwKDzRO8QKWUaffB_YujY56QF9LNYSA_uBUWi2LpuYREUMd1m8aGwk5fMDMd3uWJ36Dm6nPuG6wZLt6cNsa-GUk4Js4CBfRA8ywnYYTcBiBlzUEu37TLeyezFROm_JCIk90GyzuE-BdTJ91HljjFThY7edUhDoedCfRxkIAg"
		firebaseIDToken, err := client.VerifyIDToken(ctx, idToken)
		if err != nil {
				logger.Error("error verifying ID token: %v\n", err)
				return "", errInternalError
		}
		
		logger.Info("Verified ID token: %v\n", firebaseIDToken)

		
		// _, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		// if !ok {
		// 	return "", errNoUserIdFound
		// }
		
		// request := &api.RpcFindMatchRequest{}
		// if err := unmarshaler.Unmarshal(bytes.NewReader([]byte(payload)), request); err != nil {
		// 	return "", errUnmarshal
		// }

		return "4", nil

		// maxSize := 1
		// var fast int
		// if request.Fast {
		// 	fast = 1
		// }
		// query := fmt.Sprintf("+label.open:1 +label.fast:%d", fast)

		// matchIDs := make([]string, 0, 10)
		// matches, err := nk.MatchList(ctx, 10, true, "", nil, &maxSize, query)
		// if err != nil {
		// 	logger.Error("error listing matches: %v", err)
		// 	return "", errInternalError
		// }
		// if len(matches) > 0 {
		// 	// There are one or more ongoing matches the user could join.
		// 	for _, match := range matches {
		// 		matchIDs = append(matchIDs, match.MatchId)
		// 	}
		// } else {
		// 	// No available matches found, create a new one.
		// 	matchID, err := nk.MatchCreate(ctx, moduleName, map[string]interface{}{"fast": request.Fast})
		// 	if err != nil {
		// 		logger.Error("error creating match: %v", err)
		// 		return "", errInternalError
		// 	}
		// 	matchIDs = append(matchIDs, matchID)
		// }

		// response, err := marshaler.MarshalToString(&api.RpcFindMatchResponse{MatchIds: matchIDs})
		// if err != nil {
		// 	logger.Error("error marshaling response payload: %v", err.Error())
		// 	return "", errMarshal
		// }

		// return response, nil
	}
}
